package postgres

func auditRecordJSONFields() string {
	return `
		json_build_object(
			'id', a.id,
			'timestamp', EXTRACT(EPOCH FROM a."timestamp")::bigint,
			'action', a."action",
			'target', a.target,
			'outcome', a.outcome,
			'statusCode', a.status_code,
			'method', a."method",
			'path', a."path",
			'principalMethod', a.principal_method,
			'principalSubject', a.principal_subject,
			'principalName', a.principal_name,
			'onBehalfOf', a.on_behalf_of,
			'sourceIp', a.source_ip,
			'userAgent', a.user_agent,
			'requestId', a.request_id,
			'traceId', a.trace_id,
			'spanId', a.span_id
		)`
}

func auditRecordQuery() string {
	return `
	SELECT` + auditRecordJSONFields() + ` AS audit_record_json
	FROM audit_records a
	WHERE a.id = @id`
}

// Every filter is optional, expressed as a nullable parameter rather than by
// assembling the SQL per call, so the trail is read through one statement.
func auditRecordsQuery() string {
	return `
	SELECT` + auditRecordJSONFields() + ` AS audit_record_json
	FROM audit_records a
	WHERE (@fromTimestamp::timestamptz IS NULL OR a."timestamp" >= @fromTimestamp)
		AND (@toTimestamp::timestamptz IS NULL OR a."timestamp" < @toTimestamp)
		AND (
			@afterTimestamp::timestamptz IS NULL
			OR (a."timestamp", a.id::text) > (@afterTimestamp, @afterId::text)
		)
		AND (@action::text IS NULL OR a."action" = @action)
		AND (@principalSubject::text IS NULL OR a.principal_subject = @principalSubject)
		AND (@outcome::text IS NULL OR a.outcome = @outcome)
	ORDER BY a."timestamp" ASC, a.id ASC
	LIMIT @queryLimit`
}

func cleanupAuditRecordsQuery() string {
	return `
	DELETE FROM audit_records
	WHERE "timestamp" <= @cleanupBefore`
}

// Records are append-only, so a conflicting ID is a caller that reused one
// rather than an update to apply, so the stored record is left as it is.
func saveAuditRecordQuery() string {
	return `
	INSERT INTO audit_records (
		id,
		"timestamp",
		"action",
		target,
		outcome,
		status_code,
		"method",
		"path",
		principal_method,
		principal_subject,
		principal_name,
		on_behalf_of,
		source_ip,
		user_agent,
		request_id,
		trace_id,
		span_id
	) VALUES (
		@id,
		@timestamp,
		@action,
		@target,
		@outcome,
		@statusCode,
		@method,
		@path,
		@principalMethod,
		@principalSubject,
		@principalName,
		@onBehalfOf,
		@sourceIp,
		@userAgent,
		@requestId,
		@traceId,
		@spanId
	)
	ON CONFLICT (id) DO NOTHING`
}
