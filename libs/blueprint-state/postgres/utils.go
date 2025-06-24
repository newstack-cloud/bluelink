package postgres

import (
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
)

var (
	emptyObjectMappingNode = &core.MappingNode{
		Fields: map[string]*core.MappingNode{},
	}
)

type queryInfo struct {
	sql    string
	params *pgx.NamedArgs
}

func toNullableText(value string) pgtype.Text {
	if value == "" {
		return pgtype.Text{
			Valid: false,
		}
	}

	return pgtype.Text{
		String: value,
		Valid:  true,
	}
}

func toNullableTimestamp(value int) pgtype.Timestamp {
	if value == 0 {
		return pgtype.Timestamp{
			Valid: false,
		}
	}

	return pgtype.Timestamp{
		Time:  time.Unix(int64(value), 0),
		Valid: true,
	}
}

func toUnixTimestamp(value int) time.Time {
	return time.Unix(int64(value), 0)
}

func ptrToNullableTimestamp(value *int) pgtype.Timestamp {
	if value == nil {
		return pgtype.Timestamp{
			Valid: false,
		}
	}

	return toNullableTimestamp(*value)
}

func mappingNodeOrNilFallback(
	node *core.MappingNode,
	fallback *core.MappingNode,
) *core.MappingNode {
	if core.IsNilMappingNode(node) {
		return fallback
	}

	return node
}

func sliceOrEmpty[Item any](
	slice []Item,
) []Item {
	if slice == nil {
		// Use an empty array instead of nil for columns that
		// can not be null in the database.
		return []Item{}
	}

	return slice
}

func mapOrEmpty[Item any](
	m map[string]Item,
) map[string]Item {
	if m == nil {
		// Use an empty map instead of nil for columns that
		// can not be null in the database.
		return map[string]Item{}
	}

	return m
}

func isAltNotFoundPostgresErrorCode(errCode string) bool {
	// 22P02 is the error code for invalid text representation,
	// this will occur when an ID is provided that is not for a valid UUID.
	// Instead of revealing specifics about ID formats for queries,
	// we will return a not found error.
	return errCode == "22P02" ||
		// 23503 is the error code for foreign key violation,
		// this will occur when a blueprint instance does not exist
		// when saving a relation between a blueprint instance and
		// a resource, link or child blueprint.
		// This will also occur when a resource does not exist when saving
		// a drift entry.
		errCode == "23503"
}
