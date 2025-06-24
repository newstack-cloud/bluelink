package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/idutils"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

type resourcesContainerImpl struct {
	connPool *pgxpool.Pool
}

func (c *resourcesContainerImpl) Get(
	ctx context.Context,
	resourceID string,
) (state.ResourceState, error) {
	var resource state.ResourceState
	err := c.connPool.QueryRow(
		ctx,
		resourceQuery(),
		&pgx.NamedArgs{
			"resourceId": resourceID,
		},
	).Scan(&resource)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.Is(err, pgx.ErrNoRows) ||
			(errors.As(err, &pgErr) && isAltNotFoundPostgresErrorCode(pgErr.Code)) {
			return state.ResourceState{}, state.ResourceNotFoundError(resourceID)
		}

		return state.ResourceState{}, err
	}

	if resource.ResourceID == "" {
		return state.ResourceState{}, state.ResourceNotFoundError(resourceID)
	}

	return resource, nil
}

func (c *resourcesContainerImpl) GetByName(
	ctx context.Context,
	instanceID string,
	resourceName string,
) (state.ResourceState, error) {
	var resource state.ResourceState
	itemID := idutils.ReourceInBlueprintID(instanceID, resourceName)
	err := c.connPool.QueryRow(
		ctx,
		resourceByNameQuery(),
		&pgx.NamedArgs{
			"instanceId":   instanceID,
			"resourceName": resourceName,
		},
	).Scan(&resource)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.Is(err, pgx.ErrNoRows) ||
			(errors.As(err, &pgErr) && isAltNotFoundPostgresErrorCode(pgErr.Code)) {
			return state.ResourceState{}, state.ResourceNotFoundError(itemID)
		}

		return state.ResourceState{}, err
	}

	if resource.ResourceID == "" {
		return state.ResourceState{}, state.ResourceNotFoundError(itemID)
	}

	return resource, nil
}

func (c *resourcesContainerImpl) Save(
	ctx context.Context,
	resourceState state.ResourceState,
) error {
	tx, err := c.connPool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	resourceStateSlice := []*state.ResourceState{&resourceState}
	err = upsertResources(ctx, tx, resourceStateSlice)
	if err != nil {
		return err
	}

	err = upsertBlueprintResourceRelations(ctx, tx, resourceState.InstanceID, resourceStateSlice)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && isAltNotFoundPostgresErrorCode(pgErr.Code) {
			return state.InstanceNotFoundError(resourceState.InstanceID)
		}

		return err
	}

	return tx.Commit(ctx)
}

func (c *resourcesContainerImpl) UpdateStatus(
	ctx context.Context,
	resourceID string,
	statusInfo state.ResourceStatusInfo,
) error {
	qInfo := prepareUpdateResourceStatusQuery(resourceID, &statusInfo)
	cTag, err := c.connPool.Exec(
		ctx,
		qInfo.sql,
		qInfo.params,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && isAltNotFoundPostgresErrorCode(pgErr.Code) {
			return state.ResourceNotFoundError(resourceID)
		}

		return err
	}

	if cTag.RowsAffected() == 0 {
		return state.ResourceNotFoundError(resourceID)
	}

	return nil
}

func (c *resourcesContainerImpl) Remove(
	ctx context.Context,
	resourceID string,
) (state.ResourceState, error) {
	resourceToRemove, err := c.Get(ctx, resourceID)
	if err != nil {
		return state.ResourceState{}, err
	}

	_, err = c.connPool.Exec(
		ctx,
		removeResourceQuery(),
		&pgx.NamedArgs{
			"resourceId": resourceID,
		},
	)
	if err != nil {
		return state.ResourceState{}, err
	}

	return resourceToRemove, nil
}

func (c *resourcesContainerImpl) GetDrift(
	ctx context.Context,
	resourceID string,
) (state.ResourceDriftState, error) {
	// Ensure that the resource the drift is for exists
	// to differentiate between a resource not being found
	// and a resource drift entry not being present for a given resource ID.
	_, err := c.Get(ctx, resourceID)
	if err != nil {
		return state.ResourceDriftState{}, err
	}

	var resourceDrift state.ResourceDriftState
	err = c.connPool.QueryRow(
		ctx,
		resourceDriftQuery(),
		&pgx.NamedArgs{
			"resourceId": resourceID,
		},
	).Scan(&resourceDrift)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Am empty drift state is valid if the requested resource exists.
			return state.ResourceDriftState{}, nil
		}

		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && isAltNotFoundPostgresErrorCode(pgErr.Code) {
			return state.ResourceDriftState{}, state.ResourceNotFoundError(resourceID)
		}

		return state.ResourceDriftState{}, err
	}

	return resourceDrift, nil
}

func (c *resourcesContainerImpl) SaveDrift(
	ctx context.Context,
	driftState state.ResourceDriftState,
) error {
	tx, err := c.connPool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	qInfo := prepareUpsertResourceDriftQuery(&driftState)
	_, err = tx.Exec(
		ctx,
		qInfo.sql,
		qInfo.params,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && isAltNotFoundPostgresErrorCode(pgErr.Code) {
			return state.ResourceNotFoundError(driftState.ResourceID)
		}

		return err
	}

	err = c.updateResourceDriftedFields(ctx, tx, driftState, true)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (c *resourcesContainerImpl) RemoveDrift(
	ctx context.Context,
	resourceID string,
) (state.ResourceDriftState, error) {
	driftState, err := c.GetDrift(ctx, resourceID)
	if err != nil {
		return state.ResourceDriftState{}, err
	}

	if driftState.ResourceID == "" {
		// Do nothing if the resource exists but no drift entry is present.
		return state.ResourceDriftState{}, nil
	}

	tx, err := c.connPool.Begin(ctx)
	if err != nil {
		return state.ResourceDriftState{}, err
	}
	defer tx.Rollback(ctx)

	err = c.removeDrift(ctx, tx, resourceID)
	if err != nil {
		return state.ResourceDriftState{}, err
	}

	err = c.updateResourceDriftedFields(ctx, tx, driftState, false)
	if err != nil {
		return state.ResourceDriftState{}, err
	}

	return driftState, tx.Commit(ctx)
}

func (c *resourcesContainerImpl) removeDrift(
	ctx context.Context,
	tx pgx.Tx,
	resourceID string,
) error {
	query := removeResourceDriftQuery()
	_, err := tx.Exec(
		ctx,
		query,
		&pgx.NamedArgs{
			"resourceId": resourceID,
		},
	)
	return err
}

func (c *resourcesContainerImpl) updateResourceDriftedFields(
	ctx context.Context,
	tx pgx.Tx,
	driftState state.ResourceDriftState,
	drifted bool,
) error {
	qInfo := prepareUpdateResourceDriftedFieldsQuery(driftState, drifted)
	_, err := tx.Exec(
		ctx,
		qInfo.sql,
		qInfo.params,
	)
	return err
}

func prepareUpsertResourceDriftQuery(resourceDriftState *state.ResourceDriftState) *queryInfo {
	sql := upsertResourceDriftQuery()

	params := buildResourceDriftArgs(resourceDriftState)

	return &queryInfo{
		sql:    sql,
		params: params,
	}
}

func buildResourceDriftArgs(resourceDriftState *state.ResourceDriftState) *pgx.NamedArgs {
	return &pgx.NamedArgs{
		"resourceId": resourceDriftState.ResourceID,
		"specData":   resourceDriftState.SpecData,
		"difference": resourceDriftState.Difference,
		"timestamp":  ptrToNullableTimestamp(resourceDriftState.Timestamp),
	}
}

func prepareUpdateResourceStatusQuery(
	resourceID string,
	statusInfo *state.ResourceStatusInfo,
) *queryInfo {
	sql := updateResourceStatusQuery(statusInfo)

	params := buildUpdateResourceStatusArgs(resourceID, statusInfo)

	return &queryInfo{
		sql:    sql,
		params: params,
	}
}

func buildUpdateResourceStatusArgs(
	resourceID string,
	statusInfo *state.ResourceStatusInfo,
) *pgx.NamedArgs {
	namedArgs := pgx.NamedArgs{
		"resourceId":    resourceID,
		"status":        statusInfo.Status,
		"preciseStatus": statusInfo.PreciseStatus,
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

	if statusInfo.FailureReasons != nil {
		namedArgs["failureReasons"] = statusInfo.FailureReasons
	}

	return &namedArgs
}

func prepareUpdateResourceDriftedFieldsQuery(
	driftState state.ResourceDriftState,
	drifted bool,
) *queryInfo {
	sql := updateResourceDriftedFieldsQuery(driftState, drifted)

	params := buildUpdateResourceDriftedFieldsArgs(driftState, drifted)

	return &queryInfo{
		sql:    sql,
		params: params,
	}
}

func buildUpdateResourceDriftedFieldsArgs(
	driftState state.ResourceDriftState,
	drifted bool,
) *pgx.NamedArgs {
	namedArgs := pgx.NamedArgs{
		"resourceId": driftState.ResourceID,
		"drifted":    drifted,
	}

	if drifted && driftState.Timestamp != nil {
		namedArgs["lastDriftDetectedTimestamp"] = toNullableTimestamp(
			*driftState.Timestamp,
		)
	}

	return &namedArgs
}
