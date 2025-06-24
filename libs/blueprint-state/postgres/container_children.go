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

type childrenContainerImpl struct {
	connPool  *pgxpool.Pool
	instances *instancesContainerImpl
}

func (c *childrenContainerImpl) Get(
	ctx context.Context,
	instanceID string,
	childName string,
) (state.InstanceState, error) {
	itemID := idutils.ChildInBlueprintID(instanceID, childName)
	var childState state.InstanceState
	err := c.connPool.QueryRow(
		ctx,
		blueprintInstanceChildQuery(),
		&pgx.NamedArgs{
			"parentInstanceId": instanceID,
			"childName":        childName,
		},
	).Scan(&childState)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.Is(err, pgx.ErrNoRows) ||
			(errors.As(err, &pgErr) && isAltNotFoundPostgresErrorCode(pgErr.Code)) {
			return state.InstanceState{}, state.InstanceNotFoundError(itemID)
		}

		return state.InstanceState{}, err
	}

	return childState, nil
}

func (c *childrenContainerImpl) Attach(
	ctx context.Context,
	parentInstanceID string,
	childInstanceID string,
	childName string,
) error {
	_, err := c.instances.Get(ctx, childInstanceID)
	if err != nil {
		return err
	}

	_, err = c.instances.Get(ctx, parentInstanceID)
	if err != nil {
		return err
	}

	return c.attach(ctx, parentInstanceID, childInstanceID, childName)
}

func (c *childrenContainerImpl) attach(
	ctx context.Context,
	parentInstanceId string,
	childInstanceID string,
	childName string,
) error {
	_, err := c.connPool.Exec(
		ctx,
		attachChildQuery(),
		&pgx.NamedArgs{
			"parentInstanceId": parentInstanceId,
			"childInstanceId":  childInstanceID,
			"childName":        childName,
		},
	)
	return err
}

func (c *childrenContainerImpl) Detach(
	ctx context.Context,
	instanceID string,
	childName string,
) error {
	itemID := idutils.ChildInBlueprintID(instanceID, childName)
	cTag, err := c.connPool.Exec(
		ctx,
		detachChildQuery(),
		&pgx.NamedArgs{
			"instanceId": instanceID,
			"childName":  childName,
		},
	)
	if err != nil {
		return err
	}

	if cTag.RowsAffected() == 0 {
		return state.InstanceNotFoundError(itemID)
	}

	return nil
}

func (c *childrenContainerImpl) SaveDependencies(
	ctx context.Context,
	instanceId string,
	childName string,
	dependencies *state.DependencyInfo,
) error {
	_, err := c.instances.Get(ctx, instanceId)
	if err != nil {
		return err
	}

	return c.saveDependencies(ctx, instanceId, childName, dependencies)
}

func (c *childrenContainerImpl) saveDependencies(
	ctx context.Context,
	instanceId string,
	childName string,
	dependencies *state.DependencyInfo,
) error {
	_, err := c.connPool.Exec(
		ctx,
		saveDependenciesQuery(),
		&pgx.NamedArgs{
			"instanceId":   instanceId,
			"childName":    childName,
			"dependencies": dependencies,
		},
	)
	return err
}
