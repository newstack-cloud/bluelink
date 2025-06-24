package postgres

import "github.com/newstack-cloud/bluelink/libs/blueprint/state"

func upsertInstanceQuery() string {
	return `
	INSERT INTO blueprint_instances (
		id,
		"name",
		status,
		last_status_update_timestamp,
		last_deployed_timestamp,
		last_deploy_attempt_timestamp,
		metadata,
		exports,
		child_dependencies,
		durations
	) VALUES (
		@id,
		@name,
		@status,
		@lastStatusUpdateTimestamp,
		@lastDeployedTimestamp,
		@lastDeployAttemptTimestamp,
		@metadata,
		@exports,
		@childDependencies,
		@durations
	) ON CONFLICT (id) DO UPDATE SET
	 	status = excluded.status,
		last_status_update_timestamp = excluded.last_status_update_timestamp,
		last_deployed_timestamp = excluded.last_deployed_timestamp,
		last_deploy_attempt_timestamp = excluded.last_deploy_attempt_timestamp,
		metadata = excluded.metadata,
		exports = excluded.exports,
		child_dependencies = excluded.child_dependencies,
		durations = excluded.durations
	`
}

func upsertBlueprintInstanceRelationsQuery() string {
	return `
	INSERT INTO blueprint_instance_children (
		parent_instance_id,
		child_instance_name,
		child_instance_id
	) VALUES (
	 	@parentInstanceId,
		@childInstanceName,
		@childInstanceId
	) ON CONFLICT (parent_instance_id, child_instance_name) DO UPDATE SET
		child_instance_id = excluded.child_instance_id
	`
}

func upsertBlueprintResourceRelationsQuery() string {
	return `
	INSERT INTO blueprint_instance_resources (
		instance_id,
		resource_name,
		resource_id
	) VALUES (
	 	@instanceId,
		@resourceName,
		@resourceId
	) ON CONFLICT (instance_id, resource_name) DO UPDATE SET
		resource_id = excluded.resource_id
	`
}

func upsertBlueprintLinkRelationsQuery() string {
	return `
	INSERT INTO blueprint_instance_links (
		instance_id,
		link_name,
		link_id
	) VALUES (
	 	@instanceId,
		@linkName,
		@linkId
	) ON CONFLICT (instance_id, link_name) DO UPDATE SET
		link_id = excluded.link_id
	`
}

func blueprintInstanceQuery() string {
	return `
	SELECT
		json_build_object(
			'id', bi.id,
			'name', bi.name,
			'status', bi.status,
			'lastStatusUpdateTimestamp', EXTRACT(EPOCH FROM bi.last_status_update_timestamp)::bigint,
			'lastDeployedTimestamp', EXTRACT(EPOCH FROM bi.last_deployed_timestamp)::bigint,
			'lastDeployAttemptTimestamp', EXTRACT(EPOCH FROM bi.last_deploy_attempt_timestamp)::bigint,
			'resourceIds', COALESCE(json_object_agg(r.name, r.id) FILTER (WHERE r.name IS NOT NULL), '{}'::json),
			'resources', COALESCE(json_object_agg(r.id, r.json) FILTER (WHERE r.id IS NOT NULL), '{}'::json),
			'links', COALESCE(json_object_agg(l.name, l.json) FILTER (WHERE l.name IS NOT NULL), '{}'::json),
			'metadata', bi.metadata,
			'exports', bi.exports,
			'childDependencies', bi.child_dependencies,
			'durations', bi.durations
		) As instance_json
	FROM
		blueprint_instances bi
	LEFT JOIN resources_json r ON bi.id = r.instance_id
	LEFT JOIN links_json l ON bi.id = l.instance_id
	WHERE bi.id = @blueprintInstanceId
	GROUP BY bi.id
	`
}

func blueprintInstanceIDLookupQuery() string {
	return `
	SELECT id FROM blueprint_instances WHERE "name" = @blueprintInstanceName
	`
}

func blueprintInstanceDescendantsQuery() string {
	return `
	WITH RECURSIVE descendants AS (
		SELECT
			bic.parent_instance_id,
			bic.child_instance_name,
			bic.child_instance_id
		FROM
			blueprint_instance_children bic
		INNER JOIN blueprint_instances bi ON bi.id = bic.child_instance_id
		WHERE
			parent_instance_id = @parentInstanceId
		UNION
		SELECT
			c.parent_instance_id,
			c.child_instance_name,
			c.child_instance_id
		FROM
			blueprint_instance_children c
		INNER JOIN descendants d ON d.child_instance_id = c.parent_instance_id
	)
	SELECT
		d.parent_instance_id,
		d.child_instance_name,
		d.child_instance_id,
		json_build_object(
			'id', bi.id,
			'name', bi.name,
			'status', bi.status,
			'lastStatusUpdateTimestamp', EXTRACT(EPOCH FROM bi.last_status_update_timestamp)::bigint,
			'lastDeployedTimestamp', EXTRACT(EPOCH FROM bi.last_deployed_timestamp)::bigint,
			'lastDeployAttemptTimestamp', EXTRACT(EPOCH FROM bi.last_deploy_attempt_timestamp)::bigint,
			'resourceIds', COALESCE(json_object_agg(r.name, r.id) FILTER (WHERE r.name IS NOT NULL), '{}'::json),
			'resources', COALESCE(json_object_agg(r.id, r.json) FILTER (WHERE r.id IS NOT NULL), '{}'::json),
			'links', COALESCE(json_object_agg(l.name, l.json) FILTER (WHERE l.name IS NOT NULL), '{}'::json),
			'metadata', bi.metadata,
			'exports', bi.exports,
			'childDependencies', bi.child_dependencies,
			'durations', bi.durations
		) AS instance_json
	FROM descendants d
	INNER JOIN blueprint_instances bi ON bi.id = d.child_instance_id
	LEFT JOIN resources_json r ON bi.id = r.instance_id
	LEFT JOIN links_json l ON bi.id = l.instance_id
	GROUP BY d.parent_instance_id, d.child_instance_name, d.child_instance_id, bi.id
	`
}

func blueprintInstanceChildQuery() string {
	return `
	SELECT
		json_build_object(
			'id', bi.id,
			'name', bi.name,
			'status', bi.status,
			'lastStatusUpdateTimestamp', EXTRACT(EPOCH FROM bi.last_status_update_timestamp)::bigint,
			'lastDeployedTimestamp', EXTRACT(EPOCH FROM bi.last_deployed_timestamp)::bigint,
			'lastDeployAttemptTimestamp', EXTRACT(EPOCH FROM bi.last_deploy_attempt_timestamp)::bigint,
			'resourceIds', COALESCE(json_object_agg(r.name, r.id) FILTER (WHERE r.name IS NOT NULL), '{}'::json),
			'resources', COALESCE(json_object_agg(r.id, r.json) FILTER (WHERE r.id IS NOT NULL), '{}'::json),
			'links', COALESCE(json_object_agg(l.name, l.json) FILTER (WHERE l.name IS NOT NULL), '{}'::json),
			'metadata', bi.metadata,
			'exports', bi.exports,
			'childDependencies', bi.child_dependencies,
			'durations', bi.durations
		) As instance_json
	FROM
		blueprint_instance_children bic
	INNER JOIN blueprint_instances bi ON bi.id = bic.child_instance_id
	LEFT JOIN resources_json r ON bi.id = r.instance_id
	LEFT JOIN links_json l ON bi.id = l.instance_id
	WHERE bic.parent_instance_id = @parentInstanceId AND bic.child_instance_name = @childName
	GROUP BY bi.id
	`
}

func attachChildQuery() string {
	return `
	INSERT INTO blueprint_instance_children (
		parent_instance_id,
		child_instance_name,
		child_instance_id
	) VALUES (
		@parentInstanceId,
		@childName,
		@childInstanceId
	)
	ON CONFLICT DO NOTHING
	`
}

func detachChildQuery() string {
	return `
	DELETE FROM blueprint_instance_children
	WHERE parent_instance_id = @instanceId AND child_instance_name = @childName
	`
}

func saveDependenciesQuery() string {
	return `
	UPDATE blueprint_instances
	SET child_dependencies = jsonb_set(
		COALESCE(child_dependencies, '{}'),
		('{' || @childName || '}')::text[],
		@dependencies,
		true
	)
	WHERE id = @instanceId
	`
}

func allExportsQuery() string {
	return `
	SELECT exports FROM blueprint_instances WHERE id = @instanceId
	`
}

func singleExportQuery() string {
	return `
	SELECT COALESCE(exports->@exportName, '{}') FROM blueprint_instances WHERE id = @instanceId
	`
}

func saveAllExportsQuery() string {
	return `
	UPDATE blueprint_instances
	SET exports = @exports
	WHERE id = @instanceId
	`
}

func saveSingleExportQuery() string {
	return `
	UPDATE blueprint_instances
	SET exports = jsonb_set(
		COALESCE(exports, '{}'),
		('{' || @exportName || '}')::text[],
		@export,
		true
	)
	WHERE id = @instanceId
	`
}

func removeAllExportsQuery() string {
	return `
	UPDATE blueprint_instances
	SET exports = '{}'
	WHERE id = @instanceId
	`
}

func removeSingleExportQuery() string {
	return `
	UPDATE blueprint_instances
	SET exports = exports - @exportName
	WHERE id = @instanceId
	`
}

func blueprintMetadataQuery() string {
	return `
	SELECT metadata FROM blueprint_instances WHERE id = @instanceId
	`
}

func saveBlueprintMetadataQuery() string {
	return `
	UPDATE blueprint_instances
	SET metadata = @metadata
	WHERE id = @instanceId
	`
}

func removeBlueprintMetadataQuery() string {
	return `
	UPDATE blueprint_instances
	SET metadata = '{}'
	WHERE id = @instanceId
	`
}

func updateInstanceStatusQuery(statusInfo *state.InstanceStatusInfo) string {
	query := `
	UPDATE blueprint_instances
	SET
		status = @status`

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

	query += `
	WHERE id = @instanceId`

	return query
}

func removeInstanceQuery() string {
	return `
	DELETE FROM blueprint_instances
	WHERE id = @instanceId
	`
}
