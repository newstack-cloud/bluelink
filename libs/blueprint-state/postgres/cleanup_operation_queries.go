package postgres

func cleanupOperationQuery() string {
	return `
	SELECT
		json_build_object(
			'id', c.id,
			'cleanupType', c.cleanup_type,
			'status', c.status,
			'startedAt', EXTRACT(EPOCH FROM c.started_at)::bigint,
			'endedAt', COALESCE(EXTRACT(EPOCH FROM c.ended_at)::bigint, 0),
			'itemsDeleted', c.items_deleted,
			'errorMessage', COALESCE(c.error_message, ''),
			'thresholdDate', EXTRACT(EPOCH FROM c.threshold_date)::bigint
		) As cleanup_operation_json
	FROM cleanup_operations c
	WHERE id = @id`
}

func cleanupOperationLatestByTypeQuery() string {
	return `
	SELECT
		json_build_object(
			'id', c.id,
			'cleanupType', c.cleanup_type,
			'status', c.status,
			'startedAt', EXTRACT(EPOCH FROM c.started_at)::bigint,
			'endedAt', COALESCE(EXTRACT(EPOCH FROM c.ended_at)::bigint, 0),
			'itemsDeleted', c.items_deleted,
			'errorMessage', COALESCE(c.error_message, ''),
			'thresholdDate', EXTRACT(EPOCH FROM c.threshold_date)::bigint
		) As cleanup_operation_json
	FROM cleanup_operations c
	WHERE cleanup_type = @cleanupType
	ORDER BY started_at DESC
	LIMIT 1`
}

func saveCleanupOperationQuery() string {
	return `
	INSERT INTO cleanup_operations (
		id,
		cleanup_type,
		status,
		started_at,
		ended_at,
		items_deleted,
		error_message,
		threshold_date
	) VALUES (
		@id,
		@cleanupType,
		@status,
		@startedAt,
		@endedAt,
		@itemsDeleted,
		@errorMessage,
		@thresholdDate
	)`
}

func updateCleanupOperationQuery() string {
	return `
	UPDATE cleanup_operations SET
		status = @status,
		ended_at = @endedAt,
		items_deleted = @itemsDeleted,
		error_message = @errorMessage
	WHERE id = @id`
}

// Deletes the oldest cleanup operations for a given type, keeping only the most recent N.
func deleteOldCleanupOperationsQuery() string {
	return `
	DELETE FROM cleanup_operations
	WHERE id IN (
		SELECT id FROM cleanup_operations
		WHERE cleanup_type = @cleanupType
		ORDER BY started_at DESC
		OFFSET @keepCount
	)`
}
