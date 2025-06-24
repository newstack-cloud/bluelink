package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

type metadataContainerImpl struct {
	connPool *pgxpool.Pool
}

func (c *metadataContainerImpl) Get(
	ctx context.Context,
	instanceID string,
) (map[string]*core.MappingNode, error) {
	metadata := make(map[string]*core.MappingNode)
	err := c.connPool.QueryRow(
		ctx,
		blueprintMetadataQuery(),
		&pgx.NamedArgs{
			"instanceId": instanceID,
		},
	).Scan(&metadata)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.Is(err, pgx.ErrNoRows) ||
			(errors.As(err, &pgErr) && isAltNotFoundPostgresErrorCode(pgErr.Code)) {
			return nil, state.InstanceNotFoundError(instanceID)
		}

		return nil, err
	}

	return metadata, nil
}

func (c *metadataContainerImpl) Save(
	ctx context.Context,
	instanceID string,
	metadata map[string]*core.MappingNode,
) error {
	cTag, err := c.connPool.Exec(
		ctx,
		saveBlueprintMetadataQuery(),
		&pgx.NamedArgs{
			"instanceId": instanceID,
			"metadata":   metadata,
		},
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.Is(err, pgx.ErrNoRows) ||
			(errors.As(err, &pgErr) && isAltNotFoundPostgresErrorCode(pgErr.Code)) {
			return state.InstanceNotFoundError(instanceID)
		}

		return err
	}

	if cTag.RowsAffected() == 0 {
		return state.InstanceNotFoundError(instanceID)
	}

	return nil
}

func (c *metadataContainerImpl) Remove(
	ctx context.Context,
	instanceID string,
) (map[string]*core.MappingNode, error) {
	metadata, err := c.Get(ctx, instanceID)
	if err != nil {
		return nil, err
	}

	cTag, err := c.connPool.Exec(
		ctx,
		removeBlueprintMetadataQuery(),
		&pgx.NamedArgs{
			"instanceId": instanceID,
		},
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.Is(err, pgx.ErrNoRows) ||
			(errors.As(err, &pgErr) && isAltNotFoundPostgresErrorCode(pgErr.Code)) {
			return nil, state.InstanceNotFoundError(instanceID)
		}

		return nil, err
	}

	if cTag.RowsAffected() == 0 {
		return nil, state.InstanceNotFoundError(instanceID)
	}

	return metadata, nil
}
