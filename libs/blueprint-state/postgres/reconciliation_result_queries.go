package postgres

func reconciliationResultQuery() string {
	return `
	SELECT
		json_build_object(
			'id', r.id,
			'changesetId', r.changeset_id,
			'instanceId', r.instance_id,
			'result', r.result,
			'created', EXTRACT(EPOCH FROM r.created)::bigint
		) As reconciliation_result_json
	FROM reconciliation_results r
	WHERE id = @id`
}

func reconciliationResultLatestByChangesetQuery() string {
	return `
	SELECT
		json_build_object(
			'id', r.id,
			'changesetId', r.changeset_id,
			'instanceId', r.instance_id,
			'result', r.result,
			'created', EXTRACT(EPOCH FROM r.created)::bigint
		) As reconciliation_result_json
	FROM reconciliation_results r
	WHERE changeset_id = @changesetId
	ORDER BY created DESC
	LIMIT 1`
}

func reconciliationResultAllByChangesetQuery() string {
	return `
	SELECT
		json_build_object(
			'id', r.id,
			'changesetId', r.changeset_id,
			'instanceId', r.instance_id,
			'result', r.result,
			'created', EXTRACT(EPOCH FROM r.created)::bigint
		) As reconciliation_result_json
	FROM reconciliation_results r
	WHERE changeset_id = @changesetId
	ORDER BY created DESC`
}

func reconciliationResultLatestByInstanceQuery() string {
	return `
	SELECT
		json_build_object(
			'id', r.id,
			'changesetId', r.changeset_id,
			'instanceId', r.instance_id,
			'result', r.result,
			'created', EXTRACT(EPOCH FROM r.created)::bigint
		) As reconciliation_result_json
	FROM reconciliation_results r
	WHERE instance_id = @instanceId
	ORDER BY created DESC
	LIMIT 1`
}

func reconciliationResultAllByInstanceQuery() string {
	return `
	SELECT
		json_build_object(
			'id', r.id,
			'changesetId', r.changeset_id,
			'instanceId', r.instance_id,
			'result', r.result,
			'created', EXTRACT(EPOCH FROM r.created)::bigint
		) As reconciliation_result_json
	FROM reconciliation_results r
	WHERE instance_id = @instanceId
	ORDER BY created DESC`
}

func saveReconciliationResultQuery() string {
	return `
	INSERT INTO reconciliation_results (
		id,
		changeset_id,
		instance_id,
		result,
		created
	) VALUES (
		@id,
		@changesetId,
		@instanceId,
		@result,
		@created
	)`
}

func cleanupReconciliationResultsQuery() string {
	return `
	DELETE FROM reconciliation_results
	WHERE created < @cleanupBefore`
}
