package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
)

const (
	// Maximum number of cleanup operations to keep per cleanup type.
	cleanupOperationsRollingWindowSize = 50
)

type cleanupOperationsContainerImpl struct {
	connPool *pgxpool.Pool
	logger   core.Logger
}

func (c *cleanupOperationsContainerImpl) Get(
	ctx context.Context,
	id string,
) (*manage.CleanupOperation, error) {
	var operation manage.CleanupOperation
	err := c.connPool.QueryRow(
		ctx,
		cleanupOperationQuery(),
		&pgx.NamedArgs{
			"id": id,
		},
	).Scan(&operation)
	if err != nil {
		return nil, c.handleGetError(err, id, "")
	}

	if operation.ID == "" {
		return nil, manage.CleanupOperationNotFoundError(id)
	}

	return &operation, nil
}

func (c *cleanupOperationsContainerImpl) GetLatestByType(
	ctx context.Context,
	cleanupType manage.CleanupType,
) (*manage.CleanupOperation, error) {
	var operation manage.CleanupOperation
	err := c.connPool.QueryRow(
		ctx,
		cleanupOperationLatestByTypeQuery(),
		&pgx.NamedArgs{
			"cleanupType": cleanupType,
		},
	).Scan(&operation)
	if err != nil {
		return nil, c.handleGetError(err, "", cleanupType)
	}

	if operation.ID == "" {
		return nil, manage.CleanupOperationNotFoundForTypeError(cleanupType)
	}

	return &operation, nil
}

func (c *cleanupOperationsContainerImpl) handleGetError(
	err error,
	id string,
	cleanupType manage.CleanupType,
) error {
	var pgErr *pgconn.PgError
	if errors.Is(err, pgx.ErrNoRows) ||
		(errors.As(err, &pgErr) && isAltNotFoundPostgresErrorCode(pgErr.Code)) {
		if id != "" {
			return manage.CleanupOperationNotFoundError(id)
		}
		return manage.CleanupOperationNotFoundForTypeError(cleanupType)
	}
	return err
}

func (c *cleanupOperationsContainerImpl) Save(
	ctx context.Context,
	operation *manage.CleanupOperation,
) error {
	tx, err := c.connPool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(
		ctx,
		saveCleanupOperationQuery(),
		buildCleanupOperationArgs(operation),
	)
	if err != nil {
		return err
	}

	// Enforce rolling window by deleting old records
	_, err = tx.Exec(
		ctx,
		deleteOldCleanupOperationsQuery(),
		pgx.NamedArgs{
			"cleanupType": operation.CleanupType,
			"keepCount":   cleanupOperationsRollingWindowSize,
		},
	)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (c *cleanupOperationsContainerImpl) Update(
	ctx context.Context,
	operation *manage.CleanupOperation,
) error {
	_, err := c.connPool.Exec(
		ctx,
		updateCleanupOperationQuery(),
		&pgx.NamedArgs{
			"id":           operation.ID,
			"status":       operation.Status,
			"endedAt":      toNullableTimestamp(int(operation.EndedAt)),
			"itemsDeleted": operation.ItemsDeleted,
			"errorMessage": toNullableText(operation.ErrorMessage),
		},
	)
	return err
}

func buildCleanupOperationArgs(operation *manage.CleanupOperation) *pgx.NamedArgs {
	return &pgx.NamedArgs{
		"id":            operation.ID,
		"cleanupType":   operation.CleanupType,
		"status":        operation.Status,
		"startedAt":     toUnixTimestamp(int(operation.StartedAt)),
		"endedAt":       toNullableTimestamp(int(operation.EndedAt)),
		"itemsDeleted":  operation.ItemsDeleted,
		"errorMessage":  toNullableText(operation.ErrorMessage),
		"thresholdDate": toUnixTimestamp(int(operation.ThresholdDate)),
	}
}
