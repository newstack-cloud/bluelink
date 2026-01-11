package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
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

func (c *instancesContainerImpl) List(
	ctx context.Context,
	params state.ListInstancesParams,
) (state.ListInstancesResult, error) {
	totalCount, err := c.getInstanceCount(ctx, params.Search)
	if err != nil {
		return state.ListInstancesResult{}, err
	}

	instances, err := c.getInstanceList(ctx, params)
	if err != nil {
		return state.ListInstancesResult{}, err
	}

	return state.ListInstancesResult{
		Instances:  instances,
		TotalCount: totalCount,
	}, nil
}

func (c *instancesContainerImpl) getInstanceCount(ctx context.Context, search string) (int, error) {
	args := buildListQueryArgs(search)
	var totalCount int
	err := c.connPool.QueryRow(ctx, listInstancesCountQuery(search), args).Scan(&totalCount)
	if err != nil {
		return 0, err
	}
	return totalCount, nil
}

func (c *instancesContainerImpl) getInstanceList(
	ctx context.Context,
	params state.ListInstancesParams,
) ([]state.InstanceSummary, error) {
	args := buildListQueryArgs(params.Search)
	query := listInstancesQuery(params.Search, params.Limit, params.Offset)

	rows, err := c.connPool.Query(ctx, query, args)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var instances []state.InstanceSummary
	for rows.Next() {
		inst, err := scanInstanceSummary(rows)
		if err != nil {
			return nil, err
		}
		instances = append(instances, inst)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return instances, nil
}

func scanInstanceSummary(rows pgx.Rows) (state.InstanceSummary, error) {
	var inst state.InstanceSummary
	var status core.InstanceStatus
	var lastDeployedTs *time.Time

	err := rows.Scan(&inst.InstanceID, &inst.InstanceName, &status, &lastDeployedTs)
	if err != nil {
		return state.InstanceSummary{}, err
	}

	inst.Status = status
	if lastDeployedTs != nil {
		inst.LastDeployedTimestamp = lastDeployedTs.Unix()
	}

	return inst, nil
}

func buildListQueryArgs(search string) *pgx.NamedArgs {
	args := pgx.NamedArgs{}
	if search != "" {
		args["searchPattern"] = "%" + search + "%"
	}
	return &args
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

func (c *instancesContainerImpl) GetBatch(
	ctx context.Context,
	instanceIDsOrNames []string,
) ([]state.InstanceState, error) {
	if len(instanceIDsOrNames) == 0 {
		return []state.InstanceState{}, nil
	}

	instances, err := c.getInstancesBatch(ctx, instanceIDsOrNames)
	if err != nil {
		return nil, err
	}

	missingIDsOrNames := c.findMissingInstances(instances, instanceIDsOrNames)
	if len(missingIDsOrNames) > 0 {
		return nil, state.NewInstancesNotFoundError(missingIDsOrNames)
	}

	instanceIDs := make([]string, len(instances))
	for i := range instances {
		instanceIDs[i] = instances[i].InstanceID
	}

	descendants, err := c.getBatchDescendantInstances(ctx, instanceIDs)
	if err != nil {
		return nil, err
	}

	c.wireBatchDescendantInstances(instances, descendants)

	return c.orderByInput(instances, instanceIDsOrNames), nil
}

func (c *instancesContainerImpl) getInstancesBatch(
	ctx context.Context,
	instanceIDsOrNames []string,
) ([]state.InstanceState, error) {
	rows, err := c.connPool.Query(
		ctx,
		blueprintInstanceBatchQuery(),
		&pgx.NamedArgs{
			"instanceIds":   instanceIDsOrNames,
			"instanceNames": instanceIDsOrNames,
		},
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var instances []state.InstanceState
	for rows.Next() {
		var instance state.InstanceState
		if err := rows.Scan(&instance); err != nil {
			return nil, err
		}
		instances = append(instances, instance)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return instances, nil
}

func (c *instancesContainerImpl) getBatchDescendantInstances(
	ctx context.Context,
	instanceIDs []string,
) ([]*descendantBlueprintInfo, error) {
	rows, err := c.connPool.Query(
		ctx,
		blueprintInstanceBatchDescendantsQuery(),
		&pgx.NamedArgs{
			"parentInstanceIds": instanceIDs,
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

func (c *instancesContainerImpl) wireBatchDescendantInstances(
	instances []state.InstanceState,
	descendants []*descendantBlueprintInfo,
) {
	instanceLookup := make(map[string]*state.InstanceState, len(instances)+len(descendants))
	for i := range instances {
		instanceLookup[instances[i].InstanceID] = &instances[i]
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

func (c *instancesContainerImpl) findMissingInstances(
	found []state.InstanceState,
	requested []string,
) []string {
	foundSet := make(map[string]bool, len(found)*2)
	for _, inst := range found {
		foundSet[inst.InstanceID] = true
		foundSet[inst.InstanceName] = true
	}

	var missing []string
	for _, idOrName := range requested {
		if !foundSet[idOrName] {
			missing = append(missing, idOrName)
		}
	}

	return missing
}

func (c *instancesContainerImpl) orderByInput(
	instances []state.InstanceState,
	inputOrder []string,
) []state.InstanceState {
	instanceByID := make(map[string]*state.InstanceState, len(instances))
	instanceByName := make(map[string]*state.InstanceState, len(instances))
	for i := range instances {
		instanceByID[instances[i].InstanceID] = &instances[i]
		instanceByName[instances[i].InstanceName] = &instances[i]
	}

	result := make([]state.InstanceState, 0, len(inputOrder))
	for _, idOrName := range inputOrder {
		if inst, ok := instanceByID[idOrName]; ok {
			result = append(result, *inst)
		} else if inst, ok := instanceByName[idOrName]; ok {
			result = append(result, *inst)
		}
	}

	return result
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

// SaveBatch efficiently saves multiple instances in a single transaction.
// It handles nested children at any depth and deduplicates instances that
// appear both at the top level and as children.
//
// The operation flattens the entire instance tree first, then uses pgx.Batch
// to minimize database round-trips. The order respects FK constraints:
// 1. All instances (flattened, deduplicated)
// 2. All resources
// 3. All instance-resource relations
// 4. All links
// 5. All instance-link relations
// 6. All parent-child relations
func (c *instancesContainerImpl) SaveBatch(
	ctx context.Context,
	instances []state.InstanceState,
) error {
	if len(instances) == 0 {
		return nil
	}

	tx, err := c.connPool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Flatten and deduplicate the entire instance tree
	flatInstances, childRelations := flattenInstanceTree(instances)

	if err := c.saveFlattenedBatch(ctx, tx, flatInstances, childRelations); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

type childRelation struct {
	parentInstanceID string
	childName        string
	childInstanceID  string
}

// flattenInstanceTree recursively collects all instances and parent-child
// relations from the tree, deduplicating by instance ID.
func flattenInstanceTree(instances []state.InstanceState) ([]state.InstanceState, []childRelation) {
	seen := make(map[string]bool)
	var flatInstances []state.InstanceState
	var relations []childRelation

	var flatten func(inst *state.InstanceState)
	flatten = func(inst *state.InstanceState) {
		if seen[inst.InstanceID] {
			return
		}
		seen[inst.InstanceID] = true
		flatInstances = append(flatInstances, *inst)

		for childName, child := range inst.ChildBlueprints {
			relations = append(relations, childRelation{
				parentInstanceID: inst.InstanceID,
				childName:        childName,
				childInstanceID:  child.InstanceID,
			})
			flatten(child)
		}
	}

	for i := range instances {
		flatten(&instances[i])
	}

	return flatInstances, relations
}

func (c *instancesContainerImpl) saveFlattenedBatch(
	ctx context.Context,
	tx pgx.Tx,
	instances []state.InstanceState,
	childRelations []childRelation,
) error {
	// Step 1: Batch upsert all instances
	if err := c.upsertInstancesBatch(ctx, tx, instances); err != nil {
		return err
	}

	// Step 2: Collect and batch upsert all resources
	allResources, allLinks := collectResourcesAndLinks(instances)

	if err := upsertResources(ctx, tx, allResources); err != nil {
		return err
	}

	// Step 3: Batch upsert instance-resource relations
	if err := c.upsertResourceRelationsBatch(ctx, tx, instances); err != nil {
		return err
	}

	// Step 4: Batch upsert all links
	if err := upsertLinks(ctx, tx, allLinks); err != nil {
		return err
	}

	// Step 5: Batch upsert instance-link relations
	if err := c.upsertLinkRelationsBatch(ctx, tx, instances); err != nil {
		return err
	}

	// Step 6: Batch upsert parent-child relations
	if len(childRelations) > 0 {
		return c.upsertChildRelationsBatch(ctx, tx, childRelations)
	}

	return nil
}

func (c *instancesContainerImpl) upsertInstancesBatch(
	ctx context.Context,
	tx pgx.Tx,
	instances []state.InstanceState,
) error {
	query := upsertInstanceQuery()
	batch := &pgx.Batch{}
	for i := range instances {
		args := buildInstanceArgs(&instances[i])
		batch.Queue(query, args)
	}

	return tx.SendBatch(ctx, batch).Close()
}

func collectResourcesAndLinks(
	instances []state.InstanceState,
) ([]*state.ResourceState, []*state.LinkState) {
	var allResources []*state.ResourceState
	var allLinks []*state.LinkState

	for i := range instances {
		resources := commoncore.MapToSlice(instances[i].Resources)
		allResources = append(allResources, resources...)

		links := commoncore.MapToSlice(instances[i].Links)
		allLinks = append(allLinks, links...)
	}

	return allResources, allLinks
}

func (c *instancesContainerImpl) upsertResourceRelationsBatch(
	ctx context.Context,
	tx pgx.Tx,
	instances []state.InstanceState,
) error {
	query := upsertBlueprintResourceRelationsQuery()
	batch := &pgx.Batch{}
	for i := range instances {
		resources := commoncore.MapToSlice(instances[i].Resources)
		for _, resource := range resources {
			args := pgx.NamedArgs{
				"instanceId":   instances[i].InstanceID,
				"resourceName": resource.Name,
				"resourceId":   resource.ResourceID,
			}
			batch.Queue(query, args)
		}
	}

	if batch.Len() == 0 {
		return nil
	}

	return tx.SendBatch(ctx, batch).Close()
}

func (c *instancesContainerImpl) upsertLinkRelationsBatch(
	ctx context.Context,
	tx pgx.Tx,
	instances []state.InstanceState,
) error {
	query := upsertBlueprintLinkRelationsQuery()
	batch := &pgx.Batch{}
	for i := range instances {
		links := commoncore.MapToSlice(instances[i].Links)
		for _, link := range links {
			args := pgx.NamedArgs{
				"instanceId": instances[i].InstanceID,
				"linkName":   link.Name,
				"linkId":     link.LinkID,
			}
			batch.Queue(query, args)
		}
	}

	if batch.Len() == 0 {
		return nil
	}

	return tx.SendBatch(ctx, batch).Close()
}

func (c *instancesContainerImpl) upsertChildRelationsBatch(
	ctx context.Context,
	tx pgx.Tx,
	relations []childRelation,
) error {
	query := upsertBlueprintInstanceRelationsQuery()
	batch := &pgx.Batch{}
	for _, rel := range relations {
		args := pgx.NamedArgs{
			"parentInstanceId":  rel.parentInstanceID,
			"childInstanceName": rel.childName,
			"childInstanceId":   rel.childInstanceID,
		}
		batch.Queue(query, args)
	}

	return tx.SendBatch(ctx, batch).Close()
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
