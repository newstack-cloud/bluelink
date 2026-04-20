DROP VIEW IF EXISTS resources_json;

CREATE VIEW resources_json AS (
  SELECT
    resources.id,
  	bir.instance_id,
  	bir.resource_name AS name,
    json_build_object(
      'id', resources.id,
      'name', bir.resource_name,
      'type', resources.type,
      'templateName', resources.template_name,
      'instanceId', bir.instance_id,
      'status', resources.status,
      'preciseStatus', resources.precise_status,
      'lastStatusUpdateTimestamp', EXTRACT(EPOCH FROM resources.last_status_update_timestamp)::bigint,
      'lastDeployedTimestamp', EXTRACT(EPOCH FROM resources.last_deployed_timestamp)::bigint,
      'lastDeployAttemptTimestamp', EXTRACT(EPOCH FROM resources.last_deploy_attempt_timestamp)::bigint,
      'specData', resources.spec_data,
      'description', resources.description,
      'metadata', resources.metadata,
      'systemMetadata', resources.system_metadata,
      'computedFields', resources.computed_fields,
      'dependsOnResources', resources.depends_on_resources,
      'dependsOnChildren', resources.depends_on_children,
      'failureReasons', resources.failure_reasons,
      'drifted', resources.drifted,
      'lastDriftDetectedTimestamp', EXTRACT(EPOCH FROM resources.last_drift_detected_timestamp)::bigint,
      'durations', resources.durations
    ) AS json
  FROM
    blueprint_instance_resources bir
  INNER JOIN resources ON bir.resource_id = resources.id
);
