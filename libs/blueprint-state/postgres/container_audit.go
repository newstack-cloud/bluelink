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

type auditContainerImpl struct {
	connPool *pgxpool.Pool
	logger   core.Logger
}

func (a *auditContainerImpl) Get(
	ctx context.Context,
	id string,
) (*manage.AuditRecord, error) {
	var record manage.AuditRecord
	err := a.connPool.QueryRow(
		ctx,
		auditRecordQuery(),
		&pgx.NamedArgs{
			"id": id,
		},
	).Scan(&record)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.Is(err, pgx.ErrNoRows) ||
			(errors.As(err, &pgErr) && isAltNotFoundPostgresErrorCode(pgErr.Code)) {
			return nil, manage.AuditRecordNotFoundError(id)
		}

		return nil, err
	}

	if record.ID == "" {
		return nil, manage.AuditRecordNotFoundError(id)
	}

	return &record, nil
}

func (a *auditContainerImpl) Save(
	ctx context.Context,
	record *manage.AuditRecord,
) error {
	_, err := a.connPool.Exec(
		ctx,
		saveAuditRecordQuery(),
		buildAuditRecordArgs(record),
	)
	return err
}

func (a *auditContainerImpl) Query(
	ctx context.Context,
	query *manage.AuditRecordQuery,
) ([]*manage.AuditRecord, error) {
	rows, err := a.connPool.Query(
		ctx,
		auditRecordsQuery(),
		buildAuditRecordQueryArgs(query),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	records := []*manage.AuditRecord{}
	for rows.Next() {
		var record manage.AuditRecord
		if err := rows.Scan(&record); err != nil {
			return nil, err
		}
		records = append(records, &record)
	}
	return records, rows.Err()
}

func (a *auditContainerImpl) Cleanup(
	ctx context.Context,
	thresholdDate time.Time,
) (int64, error) {
	result, err := a.connPool.Exec(
		ctx,
		cleanupAuditRecordsQuery(),
		pgx.NamedArgs{
			"cleanupBefore": thresholdDate,
		},
	)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

func buildAuditRecordArgs(record *manage.AuditRecord) *pgx.NamedArgs {
	return &pgx.NamedArgs{
		"id":               record.ID,
		"timestamp":        toUnixTimestamp(int(record.Timestamp)),
		"action":           record.Action,
		"target":           emptyToNil(record.Target),
		"outcome":          record.Outcome,
		"statusCode":       record.StatusCode,
		"method":           record.Method,
		"path":             record.Path,
		"principalMethod":  record.PrincipalMethod,
		"principalSubject": record.PrincipalSubject,
		"principalName":    emptyToNil(record.PrincipalName),
		"onBehalfOf":       emptyToNil(record.OnBehalfOf),
		"sourceIp":         emptyToNil(record.SourceIP),
		"userAgent":        emptyToNil(record.UserAgent),
		"requestId":        emptyToNil(record.RequestID),
	}
}

func buildAuditRecordQueryArgs(query *manage.AuditRecordQuery) *pgx.NamedArgs {
	limit := query.Limit
	if limit <= 0 {
		limit = manage.DefaultAuditQueryLimit
	}

	return &pgx.NamedArgs{
		"fromTimestamp":    zeroTimestampToNil(query.From),
		"toTimestamp":      zeroTimestampToNil(query.To),
		"afterTimestamp":   zeroTimestampToNil(query.AfterTimestamp),
		"afterId":          query.AfterID,
		"action":           emptyToNil(query.Action),
		"principalSubject": emptyToNil(query.PrincipalSubject),
		"outcome":          emptyToNil(query.Outcome),
		"queryLimit":       limit,
	}
}

func zeroTimestampToNil(value int64) *time.Time {
	if value == 0 {
		return nil
	}
	timestamp := toUnixTimestamp(int(value))
	return &timestamp
}

func emptyToNil(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
