package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

type exportContainerImpl struct {
	connPool *pgxpool.Pool
}

func (c *exportContainerImpl) GetAll(
	ctx context.Context,
	instanceID string,
) (map[string]*state.ExportState, error) {
	exports := make(map[string]*state.ExportState)
	err := c.connPool.QueryRow(
		ctx,
		allExportsQuery(),
		&pgx.NamedArgs{
			"instanceId": instanceID,
		},
	).Scan(&exports)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.Is(err, pgx.ErrNoRows) ||
			(errors.As(err, &pgErr) && isAltNotFoundPostgresErrorCode(pgErr.Code)) {
			return nil, state.InstanceNotFoundError(instanceID)
		}

		return nil, err
	}

	return exports, nil
}

func (c *exportContainerImpl) Get(
	ctx context.Context,
	instanceID string,
	exportName string,
) (state.ExportState, error) {
	var export state.ExportState
	err := c.connPool.QueryRow(
		ctx,
		singleExportQuery(),
		&pgx.NamedArgs{
			"instanceId": instanceID,
			"exportName": exportName,
		},
	).Scan(&export)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.Is(err, pgx.ErrNoRows) ||
			(errors.As(err, &pgErr) && isAltNotFoundPostgresErrorCode(pgErr.Code)) {
			return state.ExportState{}, state.InstanceNotFoundError(instanceID)
		}

		return state.ExportState{}, err
	}

	if isEmptyExport(export) {
		return state.ExportState{}, state.ExportNotFoundError(instanceID, exportName)
	}

	return export, nil
}

func (c *exportContainerImpl) SaveAll(
	ctx context.Context,
	instanceID string,
	exports map[string]*state.ExportState,
) error {
	cTag, err := c.connPool.Exec(
		ctx,
		saveAllExportsQuery(),
		&pgx.NamedArgs{
			"instanceId": instanceID,
			"exports":    exports,
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

func (c *exportContainerImpl) Save(
	ctx context.Context,
	instanceID string,
	exportName string,
	export state.ExportState,
) error {
	cTag, err := c.connPool.Exec(
		ctx,
		saveSingleExportQuery(),
		&pgx.NamedArgs{
			"instanceId": instanceID,
			"exportName": exportName,
			"export":     export,
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

func (c *exportContainerImpl) RemoveAll(
	ctx context.Context,
	instanceID string,
) (map[string]*state.ExportState, error) {
	allExports, err := c.GetAll(ctx, instanceID)
	if err != nil {
		return nil, err
	}

	cTag, err := c.connPool.Exec(
		ctx,
		removeAllExportsQuery(),
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

	return allExports, nil
}

func (c *exportContainerImpl) Remove(
	ctx context.Context,
	instanceID string,
	exportName string,
) (state.ExportState, error) {
	export, err := c.Get(ctx, instanceID, exportName)
	if err != nil {
		return state.ExportState{}, err
	}

	cTag, err := c.connPool.Exec(
		ctx,
		removeSingleExportQuery(),
		&pgx.NamedArgs{
			"instanceId": instanceID,
			"exportName": exportName,
		},
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.Is(err, pgx.ErrNoRows) ||
			(errors.As(err, &pgErr) && isAltNotFoundPostgresErrorCode(pgErr.Code)) {
			return state.ExportState{}, state.InstanceNotFoundError(instanceID)
		}

		return state.ExportState{}, err
	}

	if cTag.RowsAffected() == 0 {
		return state.ExportState{}, state.InstanceNotFoundError(instanceID)
	}

	return export, nil
}

func isEmptyExport(export state.ExportState) bool {
	return export.Value == nil &&
		export.Field == "" &&
		export.Type == "" &&
		export.Description == ""
}
