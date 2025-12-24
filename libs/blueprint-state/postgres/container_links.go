package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/idutils"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

type linksContainerImpl struct {
	connPool *pgxpool.Pool
}

func (c *linksContainerImpl) Get(
	ctx context.Context,
	linkID string,
) (state.LinkState, error) {
	var link state.LinkState
	err := c.connPool.QueryRow(
		ctx,
		linkQuery(),
		&pgx.NamedArgs{
			"linkId": linkID,
		},
	).Scan(&link)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.Is(err, pgx.ErrNoRows) ||
			(errors.As(err, &pgErr) && isAltNotFoundPostgresErrorCode(pgErr.Code)) {
			return state.LinkState{}, state.LinkNotFoundError(linkID)
		}

		return state.LinkState{}, err
	}

	if link.LinkID == "" {
		return state.LinkState{}, state.LinkNotFoundError(linkID)
	}

	return link, nil
}

func (c *linksContainerImpl) GetByName(
	ctx context.Context,
	instanceID string,
	linkName string,
) (state.LinkState, error) {
	var link state.LinkState
	itemID := idutils.LinkInBlueprintID(instanceID, linkName)
	err := c.connPool.QueryRow(
		ctx,
		linkByNameQuery(),
		&pgx.NamedArgs{
			"instanceId": instanceID,
			"linkName":   linkName,
		},
	).Scan(&link)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.Is(err, pgx.ErrNoRows) ||
			(errors.As(err, &pgErr) && isAltNotFoundPostgresErrorCode(pgErr.Code)) {
			return state.LinkState{}, state.LinkNotFoundError(itemID)
		}

		return state.LinkState{}, err
	}

	if link.LinkID == "" {
		return state.LinkState{}, state.LinkNotFoundError(itemID)
	}

	return link, nil
}

func (c *linksContainerImpl) ListWithResourceDataMappings(
	ctx context.Context,
	instanceID string,
	resourceName string,
) ([]state.LinkState, error) {
	rows, err := c.connPool.Query(
		ctx,
		linksInInstanceQuery(),
		&pgx.NamedArgs{
			"instanceId": instanceID,
		},
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []*state.LinkState
	for rows.Next() {
		var linkState state.LinkState
		// This query is against the a json column, so we can scan
		// the entire value into the struct directly.
		err = rows.Scan(
			&linkState,
		)
		if err != nil {
			return nil, err
		}

		links = append(links, &linkState)
	}

	return filterLinksForResourceDataMappings(resourceName, links), nil
}

func filterLinksForResourceDataMappings(
	resourceName string,
	links []*state.LinkState,
) []state.LinkState {
	var filteredLinks []state.LinkState

	for _, link := range links {
		if _, ok := link.ResourceDataMappings[resourceName]; ok {
			filteredLinks = append(filteredLinks, *link)
		}

		for resourceFieldPath := range link.ResourceDataMappings {
			resourceNamePrefix := fmt.Sprintf("%s::", resourceName)
			if strings.HasPrefix(resourceFieldPath, resourceNamePrefix) {
				filteredLinks = append(filteredLinks, *link)
				break
			}
		}
	}

	return filteredLinks
}

func (c *linksContainerImpl) Save(
	ctx context.Context,
	linkState state.LinkState,
) error {
	tx, err := c.connPool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	linkStateSlice := []*state.LinkState{&linkState}
	err = upsertLinks(ctx, tx, linkStateSlice)
	if err != nil {
		return err
	}

	err = upsertBlueprintLinkRelations(ctx, tx, linkState.InstanceID, linkStateSlice)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && isAltNotFoundPostgresErrorCode(pgErr.Code) {
			return state.InstanceNotFoundError(linkState.InstanceID)
		}

		return err
	}

	return tx.Commit(ctx)
}

func (c *linksContainerImpl) UpdateStatus(
	ctx context.Context,
	linkID string,
	statusInfo state.LinkStatusInfo,
) error {
	qInfo := prepareUpdateLinkStatusQuery(linkID, &statusInfo)
	cTag, err := c.connPool.Exec(
		ctx,
		qInfo.sql,
		qInfo.params,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && isAltNotFoundPostgresErrorCode(pgErr.Code) {
			return state.LinkNotFoundError(linkID)
		}

		return err
	}

	if cTag.RowsAffected() == 0 {
		return state.LinkNotFoundError(linkID)
	}

	return nil
}

func (c *linksContainerImpl) Remove(
	ctx context.Context,
	linkID string,
) (state.LinkState, error) {
	linkToRemove, err := c.Get(ctx, linkID)
	if err != nil {
		return state.LinkState{}, err
	}

	_, err = c.connPool.Exec(
		ctx,
		removeLinkQuery(),
		&pgx.NamedArgs{
			"linkId": linkID,
		},
	)
	if err != nil {
		return state.LinkState{}, err
	}

	return linkToRemove, nil
}

func prepareUpdateLinkStatusQuery(
	linkID string,
	statusInfo *state.LinkStatusInfo,
) *queryInfo {
	sql := updateLinkStatusQuery(statusInfo)

	params := buildUpdateLinkStatusArgs(linkID, statusInfo)

	return &queryInfo{
		sql:    sql,
		params: params,
	}
}

func buildUpdateLinkStatusArgs(
	linkID string,
	statusInfo *state.LinkStatusInfo,
) *pgx.NamedArgs {
	namedArgs := pgx.NamedArgs{
		"linkId":        linkID,
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

func (c *linksContainerImpl) GetDrift(
	ctx context.Context,
	linkID string,
) (state.LinkDriftState, error) {
	// Ensure that the link the drift is for exists
	// to differentiate between a link not being found
	// and a link drift entry not being present for a given link ID.
	_, err := c.Get(ctx, linkID)
	if err != nil {
		return state.LinkDriftState{}, err
	}

	var linkDrift state.LinkDriftState
	err = c.connPool.QueryRow(
		ctx,
		linkDriftQuery(),
		&pgx.NamedArgs{
			"linkId": linkID,
		},
	).Scan(&linkDrift)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// An empty drift state is valid if the requested link exists.
			return state.LinkDriftState{}, nil
		}

		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && isAltNotFoundPostgresErrorCode(pgErr.Code) {
			return state.LinkDriftState{}, state.LinkNotFoundError(linkID)
		}

		return state.LinkDriftState{}, err
	}

	return linkDrift, nil
}

func (c *linksContainerImpl) SaveDrift(
	ctx context.Context,
	driftState state.LinkDriftState,
) error {
	tx, err := c.connPool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	qInfo := prepareUpsertLinkDriftQuery(&driftState)
	_, err = tx.Exec(
		ctx,
		qInfo.sql,
		qInfo.params,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && isAltNotFoundPostgresErrorCode(pgErr.Code) {
			return state.LinkNotFoundError(driftState.LinkID)
		}

		return err
	}

	err = c.updateLinkDriftedFields(ctx, tx, driftState, true)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (c *linksContainerImpl) RemoveDrift(
	ctx context.Context,
	linkID string,
) (state.LinkDriftState, error) {
	driftState, err := c.GetDrift(ctx, linkID)
	if err != nil {
		return state.LinkDriftState{}, err
	}

	if driftState.LinkID == "" {
		// Do nothing if the link exists but no drift entry is present.
		return state.LinkDriftState{}, nil
	}

	tx, err := c.connPool.Begin(ctx)
	if err != nil {
		return state.LinkDriftState{}, err
	}
	defer tx.Rollback(ctx)

	err = c.removeDrift(ctx, tx, linkID)
	if err != nil {
		return state.LinkDriftState{}, err
	}

	err = c.updateLinkDriftedFields(ctx, tx, driftState, false)
	if err != nil {
		return state.LinkDriftState{}, err
	}

	return driftState, tx.Commit(ctx)
}

func (c *linksContainerImpl) removeDrift(
	ctx context.Context,
	tx pgx.Tx,
	linkID string,
) error {
	query := removeLinkDriftQuery()
	_, err := tx.Exec(
		ctx,
		query,
		&pgx.NamedArgs{
			"linkId": linkID,
		},
	)
	return err
}

func (c *linksContainerImpl) updateLinkDriftedFields(
	ctx context.Context,
	tx pgx.Tx,
	driftState state.LinkDriftState,
	drifted bool,
) error {
	qInfo := prepareUpdateLinkDriftedFieldsQuery(driftState, drifted)
	_, err := tx.Exec(
		ctx,
		qInfo.sql,
		qInfo.params,
	)
	return err
}

func prepareUpsertLinkDriftQuery(linkDriftState *state.LinkDriftState) *queryInfo {
	sql := upsertLinkDriftQuery()
	params := buildLinkDriftArgs(linkDriftState)

	return &queryInfo{
		sql:    sql,
		params: params,
	}
}

func buildLinkDriftArgs(linkDriftState *state.LinkDriftState) *pgx.NamedArgs {
	return &pgx.NamedArgs{
		"linkId":            linkDriftState.LinkID,
		"resourceADrift":    linkDriftState.ResourceADrift,
		"resourceBDrift":    linkDriftState.ResourceBDrift,
		"intermediaryDrift": linkDriftState.IntermediaryDrift,
		"timestamp":         ptrToNullableTimestamp(linkDriftState.Timestamp),
	}
}

func prepareUpdateLinkDriftedFieldsQuery(
	driftState state.LinkDriftState,
	drifted bool,
) *queryInfo {
	sql := updateLinkDriftedFieldsQuery(driftState, drifted)
	params := buildUpdateLinkDriftedFieldsArgs(driftState, drifted)

	return &queryInfo{
		sql:    sql,
		params: params,
	}
}

func buildUpdateLinkDriftedFieldsArgs(
	driftState state.LinkDriftState,
	drifted bool,
) *pgx.NamedArgs {
	namedArgs := pgx.NamedArgs{
		"linkId":  driftState.LinkID,
		"drifted": drifted,
	}

	if drifted && driftState.Timestamp != nil {
		namedArgs["lastDriftDetectedTimestamp"] = toNullableTimestamp(
			*driftState.Timestamp,
		)
	}

	return &namedArgs
}
