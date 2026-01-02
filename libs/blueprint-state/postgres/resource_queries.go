package postgres

import "github.com/newstack-cloud/bluelink/libs/blueprint/state"

func resourceQuery() string {
	return `SELECT json FROM resources_json WHERE id = @resourceId`
}

func resourceByNameQuery() string {
	return `
	SELECT json FROM resources_json
	WHERE instance_id = @instanceId AND "name" = @resourceName`
}

func upsertResourcesQuery() string {
	return `
	INSERT INTO resources (
		id,
		type,
		template_name,
		status,
		precise_status,
		last_status_update_timestamp,
		last_deployed_timestamp,
		last_deploy_attempt_timestamp,
		spec_data,
		description,
		metadata,
		system_metadata,
		computed_fields,
		depends_on_resources,
		depends_on_children,
		failure_reasons,
		drifted,
		last_drift_detected_timestamp,
		durations
	) VALUES (
	 	@id,
		@type,
		@templateName,
		@status,
		@preciseStatus,
		@lastStatusUpdateTimestamp,
		@lastDeployedTimestamp,
		@lastDeployAttemptTimestamp,
		@specData,
		@description,
		@metadata,
		@systemMetadata,
		@computedFields,
		@dependsOnResources,
		@dependsOnChildren,
		@failureReasons,
		@drifted,
		@lastDriftDetectedTimestamp,
		@durations
	) ON CONFLICT (id) DO UPDATE SET
		type = excluded.type,
		template_name = excluded.template_name,
		status = excluded.status,
		precise_status = excluded.precise_status,
		last_status_update_timestamp = excluded.last_status_update_timestamp,
		last_deployed_timestamp = excluded.last_deployed_timestamp,
		last_deploy_attempt_timestamp = excluded.last_deploy_attempt_timestamp,
		spec_data = excluded.spec_data,
		description = excluded.description,
		metadata = excluded.metadata,
		system_metadata = excluded.system_metadata,
		computed_fields = excluded.computed_fields,
		depends_on_resources = excluded.depends_on_resources,
		depends_on_children = excluded.depends_on_children,
		failure_reasons = excluded.failure_reasons,
		drifted = excluded.drifted,
		last_drift_detected_timestamp = excluded.last_drift_detected_timestamp,
		durations = excluded.durations
	`
}

func resourceDriftQuery() string {
	return `
	SELECT 
		json_build_object(
		'resourceId', rd.resource_id,
		'resourceName', bir.resource_name,
		'specData', rd.drifted_spec_data,
		'difference', rd.difference,
		'timestamp', EXTRACT(EPOCH FROM rd.timestamp)::bigint
		) as json 
	FROM resource_drift rd
	LEFT JOIN blueprint_instance_resources bir ON bir.resource_id = rd.resource_id
	WHERE rd.resource_id = @resourceId`
}

func removeResourceDriftQuery() string {
	return `
	DELETE FROM resource_drift
	WHERE resource_id = @resourceId
	`
}

func upsertResourceDriftQuery() string {
	return `
	INSERT INTO resource_drift (
		resource_id,
		drifted_spec_data,
		difference,
		"timestamp"
	) VALUES (
		@resourceId,
		@specData,
		@difference,
		@timestamp
	) ON CONFLICT (resource_id) DO UPDATE SET
	 	drifted_spec_data = excluded.drifted_spec_data,
		difference = excluded.difference,
		"timestamp" = excluded."timestamp"
	`
}

func updateResourceDriftedFieldsQuery(driftState state.ResourceDriftState, drifted bool) string {
	query := `
	UPDATE resources
	SET
		drifted = @drifted`

	if drifted && driftState.Timestamp != nil {
		query += `,
		last_drift_detected_timestamp = @lastDriftDetectedTimestamp`
	}

	query += `
	WHERE id = @resourceId`

	return query
}

func removeResourceQuery() string {
	return `DELETE FROM resources WHERE id = @resourceId`
}

func updateResourceStatusQuery(statusInfo *state.ResourceStatusInfo) string {
	query := `
	UPDATE resources
	SET
		status = @status,
		precise_status = @preciseStatus`

	if statusInfo.LastStatusUpdateTimestamp != nil {
		query += `,
		last_status_update_timestamp = @lastStatusUpdateTimestamp`
	}

	if statusInfo.LastDeployedTimestamp != nil {
		query += `,
		last_deployed_timestamp = @lastDeployedTimestamp`
	}

	if statusInfo.LastDeployAttemptTimestamp != nil {
		query += `,
		last_deploy_attempt_timestamp = @lastDeployAttemptTimestamp`
	}

	if statusInfo.Durations != nil {
		query += `,
		durations = @durations`
	}

	if statusInfo.FailureReasons != nil {
		query += `,
		failure_reasons = @failureReasons`
	}

	query += `
	WHERE id = @resourceId`

	return query
}
