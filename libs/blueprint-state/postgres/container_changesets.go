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

type changesetsContainerImpl struct {
	connPool *pgxpool.Pool
	logger   core.Logger
}

func (c *changesetsContainerImpl) Get(
	ctx context.Context,
	id string,
) (*manage.Changeset, error) {
	var changeset manage.Changeset
	err := c.connPool.QueryRow(
		ctx,
		changesetQuery(),
		&pgx.NamedArgs{
			"id": id,
		},
	).Scan(&changeset)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.Is(err, pgx.ErrNoRows) ||
			(errors.As(err, &pgErr) && isAltNotFoundPostgresErrorCode(pgErr.Code)) {
			return nil, manage.ChangesetNotFoundError(id)
		}

		return nil, err
	}

	if changeset.ID == "" {
		return nil, manage.ChangesetNotFoundError(id)
	}

	return &changeset, nil
}

func (c *changesetsContainerImpl) Save(
	ctx context.Context,
	changeset *manage.Changeset,
) error {
	qInfo := prepareSaveChangesetQuery(changeset)
	_, err := c.connPool.Exec(
		ctx,
		qInfo.sql,
		qInfo.params,
	)
	return err
}

func (c *changesetsContainerImpl) Cleanup(
	ctx context.Context,
	thresholdDate time.Time,
) error {
	query := cleanupChangesetsQuery()
	_, err := c.connPool.Exec(
		ctx,
		query,
		pgx.NamedArgs{
			"cleanupBefore": thresholdDate,
		},
	)
	return err
}

func prepareSaveChangesetQuery(changeset *manage.Changeset) *queryInfo {
	sql := saveChangesetQuery()

	params := buildChangesetArgs(changeset)

	return &queryInfo{
		sql:    sql,
		params: params,
	}
}

func buildChangesetArgs(changeset *manage.Changeset) *pgx.NamedArgs {
	return &pgx.NamedArgs{
		"id":                changeset.ID,
		"instanceId":        changeset.InstanceID,
		"destroy":           changeset.Destroy,
		"status":            changeset.Status,
		"blueprintLocation": changeset.BlueprintLocation,
		"changes":           changeset.Changes,
		"created":           toUnixTimestamp(int(changeset.Created)),
	}
}
