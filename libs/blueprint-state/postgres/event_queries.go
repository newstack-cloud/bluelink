package postgres

import (
	"fmt"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
)

func eventQuery() string {
	return `
	SELECT
		json_build_object(
			'id', e.id,
			'type', e.type,
			'channelType', e.channel_type,
			'channelId', e.channel_id,
			'data', e.data::text,
			'timestamp', EXTRACT(EPOCH FROM e.timestamp)::bigint,
			'end', e.end
		) As event_json
	FROM events e
	WHERE id = @id
	`
}

func saveEventQuery() string {
	return `
		INSERT INTO events (
			id,
			"type",
			channel_type,
			channel_id,
			data,
			"timestamp",
			"end"
		) VALUES (
			@id,
			@eventType,
			@channelType,
			@channelId,
			@data,
			@timestamp,
			@end
		)
		ON CONFLICT (id, "timestamp") DO NOTHING
	`
}

func channelEventsQuery(
	params *manage.EventStreamParams,
	includeStartingEventID bool,
) string {
	query := `
		SELECT
			e.id,
			e.type,
			e.channel_type as channelType,
			e.channel_id as channelId,
			e.data::text as data,
			EXTRACT(EPOCH FROM e.timestamp)::bigint as timestamp,
			e.end
		FROM events e
		WHERE channel_type = @channelType
			AND channel_id = @channelId
	`

	if params.StartingEventID != "" {
		query += fmt.Sprintf(`
			AND e.id %s @afterEventId
		`, comparisonOperator(includeStartingEventID))
	}

	if params.StartingEventID == "" {
		query += `
			AND e.timestamp > @afterTimestamp
		`
	}

	query += `
		ORDER BY e.timestamp ASC, e.id ASC
	`

	return query
}

func lastChannelEventQuery() string {
	return `
		SELECT
			e.id,
			e.type,
			e.channel_type as channelType,
			e.channel_id as channelId,
			e.data::text as data,
			EXTRACT(EPOCH FROM e.timestamp)::bigint as timestamp,
			e.end
		FROM events e
		WHERE channel_type = @channelType
			AND channel_id = @channelId
		ORDER BY e.timestamp DESC
		LIMIT 1
	`
}

func cleanupEventsQuery() string {
	return `
		DELETE FROM events
		WHERE "timestamp" < @cleanupBefore
	`
}

func eventsByIDsQuery() string {
	return `
		SELECT
			e.id,
			e.type,
			e.channel_type as channelType,
			e.channel_id as channelId,
			e.data::text as data,
			EXTRACT(EPOCH FROM e.timestamp)::bigint as timestamp,
			e.end
		FROM events e
		WHERE id = ANY(@ids)
	`
}

func comparisonOperator(inclusive bool) string {
	if inclusive {
		return ">="
	}
	return ">"
}
