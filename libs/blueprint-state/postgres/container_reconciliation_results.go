package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
)

type reconciliationResultsContainerImpl struct {
	connPool *pgxpool.Pool
	logger   core.Logger
}

func (r *reconciliationResultsContainerImpl) Get(
	ctx context.Context,
	id string,
) (*manage.ReconciliationResult, error) {
	var result manage.ReconciliationResult
	err := r.connPool.QueryRow(
		ctx,
		reconciliationResultQuery(),
		&pgx.NamedArgs{
			"id": id,
		},
	).Scan(&result)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.Is(err, pgx.ErrNoRows) ||
			(errors.As(err, &pgErr) && isAltNotFoundPostgresErrorCode(pgErr.Code)) {
			return nil, manage.ReconciliationResultNotFoundError(id)
		}

		return nil, err
	}

	if result.ID == "" {
		return nil, manage.ReconciliationResultNotFoundError(id)
	}

	return &result, nil
}

func (r *reconciliationResultsContainerImpl) GetLatestByChangesetID(
	ctx context.Context,
	changesetID string,
) (*manage.ReconciliationResult, error) {
	var result manage.ReconciliationResult
	err := r.connPool.QueryRow(
		ctx,
		reconciliationResultLatestByChangesetQuery(),
		&pgx.NamedArgs{
			"changesetId": changesetID,
		},
	).Scan(&result)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.Is(err, pgx.ErrNoRows) ||
			(errors.As(err, &pgErr) && isAltNotFoundPostgresErrorCode(pgErr.Code)) {
			return nil, manage.ReconciliationResultNotFoundForChangesetError(changesetID)
		}

		return nil, err
	}

	if result.ID == "" {
		return nil, manage.ReconciliationResultNotFoundForChangesetError(changesetID)
	}

	return &result, nil
}

func (r *reconciliationResultsContainerImpl) GetAllByChangesetID(
	ctx context.Context,
	changesetID string,
) ([]*manage.ReconciliationResult, error) {
	rows, err := r.connPool.Query(
		ctx,
		reconciliationResultAllByChangesetQuery(),
		&pgx.NamedArgs{
			"changesetId": changesetID,
		},
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return collectReconciliationResults(rows)
}

func (r *reconciliationResultsContainerImpl) GetLatestByInstanceID(
	ctx context.Context,
	instanceID string,
) (*manage.ReconciliationResult, error) {
	var result manage.ReconciliationResult
	err := r.connPool.QueryRow(
		ctx,
		reconciliationResultLatestByInstanceQuery(),
		&pgx.NamedArgs{
			"instanceId": instanceID,
		},
	).Scan(&result)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.Is(err, pgx.ErrNoRows) ||
			(errors.As(err, &pgErr) && isAltNotFoundPostgresErrorCode(pgErr.Code)) {
			return nil, manage.ReconciliationResultNotFoundForInstanceError(instanceID)
		}

		return nil, err
	}

	if result.ID == "" {
		return nil, manage.ReconciliationResultNotFoundForInstanceError(instanceID)
	}

	return &result, nil
}

func (r *reconciliationResultsContainerImpl) GetAllByInstanceID(
	ctx context.Context,
	instanceID string,
) ([]*manage.ReconciliationResult, error) {
	rows, err := r.connPool.Query(
		ctx,
		reconciliationResultAllByInstanceQuery(),
		&pgx.NamedArgs{
			"instanceId": instanceID,
		},
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return collectReconciliationResults(rows)
}

func (r *reconciliationResultsContainerImpl) Save(
	ctx context.Context,
	result *manage.ReconciliationResult,
) error {
	qInfo := prepareSaveReconciliationResultQuery(result)
	_, err := r.connPool.Exec(
		ctx,
		qInfo.sql,
		qInfo.params,
	)
	return err
}

func (r *reconciliationResultsContainerImpl) Cleanup(
	ctx context.Context,
	thresholdDate time.Time,
) (int64, error) {
	query := cleanupReconciliationResultsQuery()
	result, err := r.connPool.Exec(
		ctx,
		query,
		pgx.NamedArgs{
			"cleanupBefore": thresholdDate,
		},
	)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

func prepareSaveReconciliationResultQuery(result *manage.ReconciliationResult) *queryInfo {
	sql := saveReconciliationResultQuery()

	params := buildReconciliationResultArgs(result)

	return &queryInfo{
		sql:    sql,
		params: params,
	}
}

func buildReconciliationResultArgs(result *manage.ReconciliationResult) *pgx.NamedArgs {
	return &pgx.NamedArgs{
		"id":          result.ID,
		"changesetId": result.ChangesetID,
		"instanceId":  result.InstanceID,
		"result":      result.Result,
		"created":     toUnixTimestamp(int(result.Created)),
	}
}

func collectReconciliationResults(rows pgx.Rows) ([]*manage.ReconciliationResult, error) {
	results := []*manage.ReconciliationResult{}

	for rows.Next() {
		var result manage.ReconciliationResult
		err := rows.Scan(&result)
		if err != nil {
			return nil, err
		}
		results = append(results, &result)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}
