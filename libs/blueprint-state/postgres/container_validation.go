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

type validationContainerImpl struct {
	connPool *pgxpool.Pool
	logger   core.Logger
}

func (v *validationContainerImpl) Get(
	ctx context.Context,
	id string,
) (*manage.BlueprintValidation, error) {
	var blueprintValidation manage.BlueprintValidation
	err := v.connPool.QueryRow(
		ctx,
		blueprintValidationQuery(),
		&pgx.NamedArgs{
			"id": id,
		},
	).Scan(&blueprintValidation)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.Is(err, pgx.ErrNoRows) ||
			(errors.As(err, &pgErr) && isAltNotFoundPostgresErrorCode(pgErr.Code)) {
			return nil, manage.BlueprintValidationNotFoundError(id)
		}

		return nil, err
	}

	if blueprintValidation.ID == "" {
		return nil, manage.BlueprintValidationNotFoundError(id)
	}

	return &blueprintValidation, nil
}

func (v *validationContainerImpl) Save(
	ctx context.Context,
	validation *manage.BlueprintValidation,
) error {
	qInfo := prepareSaveBlueprintValidationQuery(validation)
	_, err := v.connPool.Exec(
		ctx,
		qInfo.sql,
		qInfo.params,
	)
	return err
}

func (v *validationContainerImpl) Cleanup(
	ctx context.Context,
	thresholdDate time.Time,
) error {
	query := cleanupBlueprintValidationsQuery()
	_, err := v.connPool.Exec(
		ctx,
		query,
		pgx.NamedArgs{
			"cleanupBefore": thresholdDate,
		},
	)
	return err
}

func prepareSaveBlueprintValidationQuery(validation *manage.BlueprintValidation) *queryInfo {
	sql := saveBlueprintValidationQuery()

	params := buildBlueprintValidationArgs(validation)

	return &queryInfo{
		sql:    sql,
		params: params,
	}
}

func buildBlueprintValidationArgs(validation *manage.BlueprintValidation) *pgx.NamedArgs {
	return &pgx.NamedArgs{
		"id":                validation.ID,
		"status":            validation.Status,
		"blueprintLocation": validation.BlueprintLocation,
		"created":           toUnixTimestamp(int(validation.Created)),
	}
}
