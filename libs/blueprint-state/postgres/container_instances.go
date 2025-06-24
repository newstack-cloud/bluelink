package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	commoncore "github.com/newstack-cloud/bluelink/libs/common/core"
)

type instancesContainerImpl struct {
	connPool *pgxpool.Pool
}

func (c *instancesContainerImpl) Get(
	ctx context.Context,
	instanceID string,
) (state.InstanceState, error) {
	instance, err := c.getInstance(ctx, instanceID)
	if err != nil {
		return state.InstanceState{}, err
	}

	if instance.InstanceID == "" {
		return state.InstanceState{}, state.InstanceNotFoundError(instanceID)
	}

	descendantInstances, err := c.getDescendantInstances(ctx, instanceID)
	if err != nil {
		return state.InstanceState{}, err
	}

	c.wireDescendantInstances(&instance, descendantInstances)

	return instance, nil
}

func (c *instancesContainerImpl) LookupIDByName(
	ctx context.Context,
	instanceName string,
) (string, error) {
	var instanceID string
	err := c.connPool.QueryRow(
		ctx,
		blueprintInstanceIDLookupQuery(),
		&pgx.NamedArgs{
			"blueprintInstanceName": instanceName,
		},
	).Scan(&instanceID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.Is(err, pgx.ErrNoRows) ||
			(errors.As(err, &pgErr) && isAltNotFoundPostgresErrorCode(pgErr.Code)) {
			return "", state.InstanceNotFoundError(instanceName)
		}

		return "", err
	}

	return instanceID, nil
}

func (c *instancesContainerImpl) wireDescendantInstances(
	parentInstance *state.InstanceState,
	descendants []*descendantBlueprintInfo,
) {
	instanceLookup := map[string]*state.InstanceState{
		parentInstance.InstanceID: parentInstance,
	}
	for _, descendant := range descendants {
		instanceLookup[descendant.childInstanceID] = &descendant.instance
	}

	for _, descendant := range descendants {
		parent, ok := instanceLookup[descendant.parentInstanceID]
		if ok {
			if parent.ChildBlueprints == nil {
				parent.ChildBlueprints = make(map[string]*state.InstanceState)
			}
			parent.ChildBlueprints[descendant.childInstanceName] = &descendant.instance
		}
	}
}

func (c *instancesContainerImpl) getInstance(ctx context.Context, instanceID string) (state.InstanceState, error) {
	var instance state.InstanceState
	err := c.connPool.QueryRow(
		ctx,
		blueprintInstanceQuery(),
		&pgx.NamedArgs{
			"blueprintInstanceId": instanceID,
		},
	).Scan(&instance)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.Is(err, pgx.ErrNoRows) ||
			(errors.As(err, &pgErr) && isAltNotFoundPostgresErrorCode(pgErr.Code)) {
			return state.InstanceState{}, state.InstanceNotFoundError(instanceID)
		}

		return state.InstanceState{}, err
	}

	return instance, nil
}

func (c *instancesContainerImpl) getDescendantInstances(ctx context.Context, instanceID string) ([]*descendantBlueprintInfo, error) {
	rows, err := c.connPool.Query(
		ctx,
		blueprintInstanceDescendantsQuery(),
		&pgx.NamedArgs{
			"parentInstanceId": instanceID,
		},
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var descendants []*descendantBlueprintInfo
	for rows.Next() {
		var descendant descendantBlueprintInfo
		err = rows.Scan(
			&descendant.parentInstanceID,
			&descendant.childInstanceName,
			&descendant.childInstanceID,
			&descendant.instance,
		)
		if err != nil {
			return nil, err
		}

		descendants = append(descendants, &descendant)
	}

	return descendants, nil
}

func (c *instancesContainerImpl) Save(
	ctx context.Context,
	instanceState state.InstanceState,
) error {
	tx, err := c.connPool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	err = c.save(ctx, tx, &instanceState)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (c *instancesContainerImpl) save(
	ctx context.Context,
	tx pgx.Tx,
	instanceState *state.InstanceState,
) error {
	err := c.upsertInstance(ctx, tx, instanceState)
	if err != nil {
		return err
	}

	resources := commoncore.MapToSlice(instanceState.Resources)
	err = upsertResources(ctx, tx, resources)
	if err != nil {
		return err
	}

	err = upsertBlueprintResourceRelations(
		ctx,
		tx,
		instanceState.InstanceID,
		resources,
	)
	if err != nil {
		return err
	}

	links := commoncore.MapToSlice(instanceState.Links)
	err = upsertLinks(ctx, tx, links)
	if err != nil {
		return err
	}

	err = upsertBlueprintLinkRelations(
		ctx,
		tx,
		instanceState.InstanceID,
		links,
	)
	if err != nil {
		return err
	}

	childBlueprints := commoncore.MapToSlice(instanceState.ChildBlueprints)
	err = c.upsertInstances(ctx, tx, childBlueprints)
	if err != nil {
		return err
	}

	return c.upsertChildBlueprintRelations(
		ctx,
		tx,
		instanceState.InstanceID,
		instanceState.ChildBlueprints,
	)
}

func (c *instancesContainerImpl) upsertInstance(
	ctx context.Context,
	tx pgx.Tx,
	instanceState *state.InstanceState,
) error {
	qInfo := prepareUpsertInstanceQuery(instanceState)
	_, err := tx.Exec(
		ctx,
		qInfo.sql,
		qInfo.params,
	)
	if err != nil {
		return err
	}

	return nil
}

func (c *instancesContainerImpl) upsertChildBlueprintRelations(
	ctx context.Context,
	tx pgx.Tx,
	instanceID string,
	instances map[string]*state.InstanceState,
) error {
	query := upsertBlueprintInstanceRelationsQuery()
	batch := &pgx.Batch{}
	for childName, instance := range instances {
		args := pgx.NamedArgs{
			"parentInstanceId":  instanceID,
			"childInstanceName": childName,
			"childInstanceId":   instance.InstanceID,
		}
		batch.Queue(query, args)
	}

	return tx.SendBatch(ctx, batch).Close()
}

func (c *instancesContainerImpl) upsertInstances(
	ctx context.Context,
	tx pgx.Tx,
	instances []*state.InstanceState,
) error {
	for _, instance := range instances {
		err := c.save(ctx, tx, instance)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *instancesContainerImpl) UpdateStatus(
	ctx context.Context,
	instanceID string,
	statusInfo state.InstanceStatusInfo,
) error {
	qInfo := prepareUpdateInstanceStatusQuery(instanceID, &statusInfo)
	cTag, err := c.connPool.Exec(
		ctx,
		qInfo.sql,
		qInfo.params,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && isAltNotFoundPostgresErrorCode(pgErr.Code) {
			return state.InstanceNotFoundError(instanceID)
		}

		return err
	}

	if cTag.RowsAffected() == 0 {
		return state.InstanceNotFoundError(instanceID)
	}

	return nil
}

func (c *instancesContainerImpl) Remove(
	ctx context.Context,
	instanceID string,
) (state.InstanceState, error) {
	tx, err := c.connPool.Begin(ctx)
	if err != nil {
		return state.InstanceState{}, err
	}
	defer tx.Rollback(ctx)

	stateToRemove, err := c.Get(ctx, instanceID)
	if err != nil {
		return state.InstanceState{}, err
	}

	err = c.removeResources(ctx, tx, stateToRemove.Resources)
	if err != nil {
		return state.InstanceState{}, err
	}

	err = c.removeLinks(ctx, tx, stateToRemove.Links)
	if err != nil {
		return state.InstanceState{}, err
	}

	err = c.removeInstance(ctx, tx, instanceID)
	if err != nil {
		return state.InstanceState{}, err
	}

	return stateToRemove, tx.Commit(ctx)
}

func (c *instancesContainerImpl) removeInstance(
	ctx context.Context,
	tx pgx.Tx,
	instanceID string,
) error {
	query := removeInstanceQuery()
	_, err := tx.Exec(
		ctx,
		query,
		&pgx.NamedArgs{
			"instanceId": instanceID,
		},
	)
	if err != nil {
		return err
	}

	return nil
}

func (c *instancesContainerImpl) removeResources(
	ctx context.Context,
	tx pgx.Tx,
	resources map[string]*state.ResourceState,
) error {
	resourceSlice := commoncore.MapToSlice(resources)
	resourceIDs := commoncore.Map(
		resourceSlice,
		func(r *state.ResourceState, _ int) string {
			return r.ResourceID
		},
	)
	queryInfo := prepareRemoveResourcesQuery(resourceIDs)
	_, err := tx.Exec(
		ctx,
		queryInfo.sql,
		queryInfo.params,
	)
	if err != nil {
		return err
	}

	return nil
}

func (c *instancesContainerImpl) removeLinks(
	ctx context.Context,
	tx pgx.Tx,
	links map[string]*state.LinkState,
) error {
	linkSlice := commoncore.MapToSlice(links)
	linkIDs := commoncore.Map(
		linkSlice,
		func(l *state.LinkState, _ int) string {
			return l.LinkID
		},
	)
	queryInfo := prepareRemoveLinksQuery(linkIDs)
	_, err := tx.Exec(
		ctx,
		queryInfo.sql,
		queryInfo.params,
	)
	if err != nil {
		return err
	}

	return nil
}

func prepareRemoveResourcesQuery(resourceIDs []string) *queryInfo {
	idParamNames := make([]string, len(resourceIDs))
	params := pgx.NamedArgs{}
	for i, resourceID := range resourceIDs {
		idParamName := fmt.Sprintf("id%d", i+1)
		idParamNames[i] = idParamName
		params[idParamName] = resourceID
	}

	sql := removeMultipleQuery("resources", idParamNames)

	return &queryInfo{
		sql:    sql,
		params: &params,
	}
}

func prepareRemoveLinksQuery(linkIDs []string) *queryInfo {
	idParamNames := make([]string, len(linkIDs))
	params := pgx.NamedArgs{}
	for i, linkID := range linkIDs {
		idParamName := fmt.Sprintf("id%d", i+1)
		idParamNames[i] = idParamName
		params[idParamName] = linkID
	}

	sql := removeMultipleQuery("links", idParamNames)

	return &queryInfo{
		sql:    sql,
		params: &params,
	}
}

func prepareUpsertInstanceQuery(instanceState *state.InstanceState) *queryInfo {
	sql := upsertInstanceQuery()

	params := buildInstanceArgs(instanceState)

	return &queryInfo{
		sql:    sql,
		params: params,
	}
}

func buildInstanceArgs(instanceState *state.InstanceState) *pgx.NamedArgs {
	return &pgx.NamedArgs{
		"id":                         instanceState.InstanceID,
		"name":                       instanceState.InstanceName,
		"status":                     instanceState.Status,
		"lastStatusUpdateTimestamp":  toNullableTimestamp(instanceState.LastStatusUpdateTimestamp),
		"lastDeployedTimestamp":      toUnixTimestamp(instanceState.LastDeployedTimestamp),
		"lastDeployAttemptTimestamp": toUnixTimestamp(instanceState.LastDeployAttemptTimestamp),
		"metadata":                   instanceState.Metadata,
		"exports":                    instanceState.Exports,
		"childDependencies":          instanceState.ChildDependencies,
		"durations":                  instanceState.Durations,
	}
}

func prepareUpdateInstanceStatusQuery(
	instanceID string,
	statusInfo *state.InstanceStatusInfo,
) *queryInfo {
	sql := updateInstanceStatusQuery(statusInfo)

	params := buildUpdateInstanceStatusArgs(instanceID, statusInfo)

	return &queryInfo{
		sql:    sql,
		params: params,
	}
}

func buildUpdateInstanceStatusArgs(
	instanceID string,
	statusInfo *state.InstanceStatusInfo,
) *pgx.NamedArgs {
	namedArgs := pgx.NamedArgs{
		"instanceId": instanceID,
		"status":     statusInfo.Status,
	}

	if statusInfo.LastDeployedTimestamp != nil {
		namedArgs["lastDeployedTimestamp"] = toUnixTimestamp(
			*statusInfo.LastDeployedTimestamp,
		)
	}

	if statusInfo.LastDeployAttemptTimestamp != nil {
		namedArgs["lastDeployAttemptTimestamp"] = toUnixTimestamp(
			*statusInfo.LastDeployAttemptTimestamp,
		)
	}

	if statusInfo.LastStatusUpdateTimestamp != nil {
		namedArgs["lastStatusUpdateTimestamp"] = toUnixTimestamp(
			*statusInfo.LastStatusUpdateTimestamp,
		)
	}

	if statusInfo.Durations != nil {
		namedArgs["durations"] = statusInfo.Durations
	}

	return &namedArgs
}
