CREATE OR REPLACE VIEW links_json AS (
  SELECT
    links.id,
  	bil.instance_id,
  	bil.link_name AS name,
    json_build_object(
      'id', links.id,
      'name', bil.link_name,
      'instanceId', bil.instance_id,
      'status', links.status,
      'preciseStatus', links.precise_status,
      'lastStatusUpdateTimestamp', EXTRACT(EPOCH FROM links.last_status_update_timestamp)::bigint,
      'lastDeployedTimestamp', EXTRACT(EPOCH FROM links.last_deployed_timestamp)::bigint,
      'lastDeployAttemptTimestamp', EXTRACT(EPOCH FROM links.last_deploy_attempt_timestamp)::bigint,
      'data', links.data,
      'failureReasons', links.failure_reasons,
      'durations', links.durations,
      'resourceDataMappings', links.resource_data_mappings
    ) AS json
  FROM
    blueprint_instance_links bil
  INNER JOIN links ON bil.link_id = links.id
);
