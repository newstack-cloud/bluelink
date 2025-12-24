package postgres

import "github.com/newstack-cloud/bluelink/libs/blueprint/state"

func linkQuery() string {
	return `SELECT json FROM links_json WHERE id = @linkId`
}

func linkByNameQuery() string {
	return `
	SELECT json FROM links_json
	WHERE instance_id = @instanceId AND "name" = @linkName`
}

func linksInInstanceQuery() string {
	return `
	SELECT json FROM links_json
	WHERE instance_id = @instanceId`
}

func upsertLinksQuery() string {
	return `
	INSERT INTO links (
		id,
		status,
		precise_status,
		last_status_update_timestamp,
		last_deployed_timestamp,
		last_deploy_attempt_timestamp,
		intermediary_resources_state,
		data,
		resource_data_mappings,
		failure_reasons,
		durations
	) VALUES (
	 	@id,
		@status,
		@preciseStatus,
		@lastStatusUpdateTimestamp,
		@lastDeployedTimestamp,
		@lastDeployAttemptTimestamp,
		@intermediaryResourcesState,
		@data,
		@resourceDataMappings,
		@failureReasons,
		@durations
	) ON CONFLICT (id) DO UPDATE SET
		status = excluded.status,
		precise_status = excluded.precise_status,
		last_status_update_timestamp = excluded.last_status_update_timestamp,
		last_deployed_timestamp = excluded.last_deployed_timestamp,
		last_deploy_attempt_timestamp = excluded.last_deploy_attempt_timestamp,
		intermediary_resources_state = excluded.intermediary_resources_state,
		data = excluded.data,
		resource_data_mappings = excluded.resource_data_mappings,
		failure_reasons = excluded.failure_reasons,
		durations = excluded.durations
	`
}

func updateLinkStatusQuery(statusInfo *state.LinkStatusInfo) string {
	query := `
	UPDATE links
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
	WHERE id = @linkId`

	return query
}

func removeLinkQuery() string {
	return `DELETE FROM links WHERE id = @linkId`
}

func linkDriftQuery() string {
	return `
	SELECT
		json_build_object(
			'linkId', ld.link_id,
			'linkName', bil.link_name,
			'resourceADrift', ld.resource_a_drift,
			'resourceBDrift', ld.resource_b_drift,
			'intermediaryDrift', ld.intermediary_drift,
			'timestamp', EXTRACT(EPOCH FROM ld.timestamp)::bigint
		) as json
	FROM link_drift ld
	LEFT JOIN blueprint_instance_links bil ON bil.link_id = ld.link_id
	WHERE ld.link_id = @linkId`
}

func upsertLinkDriftQuery() string {
	return `
	INSERT INTO link_drift (
		link_id,
		resource_a_drift,
		resource_b_drift,
		intermediary_drift,
		"timestamp"
	) VALUES (
		@linkId,
		@resourceADrift,
		@resourceBDrift,
		@intermediaryDrift,
		@timestamp
	) ON CONFLICT (link_id) DO UPDATE SET
		resource_a_drift = excluded.resource_a_drift,
		resource_b_drift = excluded.resource_b_drift,
		intermediary_drift = excluded.intermediary_drift,
		"timestamp" = excluded."timestamp"
	`
}

func removeLinkDriftQuery() string {
	return `
	DELETE FROM link_drift
	WHERE link_id = @linkId
	`
}

func updateLinkDriftedFieldsQuery(driftState state.LinkDriftState, drifted bool) string {
	query := `
	UPDATE links
	SET
		drifted = @drifted`

	if drifted && driftState.Timestamp != nil {
		query += `,
		last_drift_detected_timestamp = @lastDriftDetectedTimestamp`
	}

	query += `
	WHERE id = @linkId`

	return query
}
