-- Blueprint instance records
CREATE TABLE IF NOT EXISTS temp_blueprint_instances (data jsonb);
\COPY temp_blueprint_instances (data) FROM 'postgres/__testdata/seed/tmp/blueprint-instances.nd.json';

INSERT INTO blueprint_instances (
    id,
    "name",
    "status",
    last_status_update_timestamp,
    last_deployed_timestamp,
    last_deploy_attempt_timestamp,
    metadata,
    exports,
    child_dependencies,
    durations
)
SELECT
    (data->>'id')::uuid,
    data->>'name',
    (data->>'status')::smallint,
    TO_TIMESTAMP((data->>'lastStatusUpdateTimestamp')::bigint),
    TO_TIMESTAMP((data->>'lastDeployedTimestamp')::bigint),
    TO_TIMESTAMP((data->>'lastDeployAttemptTimestamp')::bigint),
    (data->>'metadata')::jsonb,
    (data->>'exports')::jsonb,
    (data->>'childDependencies')::jsonb,
    (data->>'durations')::jsonb
FROM temp_blueprint_instances;

DROP TABLE IF EXISTS temp_blueprint_instances;


-- Blueprint instance children records
CREATE TABLE IF NOT EXISTS temp_blueprint_instance_children (data jsonb);
\COPY temp_blueprint_instance_children (data) FROM 'postgres/__testdata/seed/tmp/blueprint-instance-children.nd.json';

INSERT INTO blueprint_instance_children (
    parent_instance_id,
    child_instance_name,
    child_instance_id
)
SELECT
    (data->>'parentInstanceId')::uuid,
    data->>'childInstanceName',
    (data->>'childInstanceId')::uuid
FROM temp_blueprint_instance_children;

DROP TABLE IF EXISTS temp_blueprint_instance_children;


-- Resource records
CREATE TABLE IF NOT EXISTS temp_resources (data jsonb);
\COPY temp_resources (data) FROM 'postgres/__testdata/seed/tmp/resources.nd.json';

INSERT INTO resources (
    id,
    "type",
    template_name,
    "status",
    precise_status,
    last_status_update_timestamp,
    last_deployed_timestamp,
    last_deploy_attempt_timestamp,
    spec_data,
    "description",
    metadata,
    depends_on_resources,
    depends_on_children,
    failure_reasons,
    drifted,
    last_drift_detected_timestamp,
    durations
)
SELECT
    (data->>'id')::uuid,
    data->>'type',
    data->>'templateName',
    (data->>'status')::smallint,
    (data->>'preciseStatus')::smallint,
    TO_TIMESTAMP((data->>'lastStatusUpdateTimestamp')::bigint),
    TO_TIMESTAMP((data->>'lastDeployedTimestamp')::bigint),
    TO_TIMESTAMP((data->>'lastDeployAttemptTimestamp')::bigint),
    (data->>'specData')::jsonb,
    data->>'description',
    (data->>'metadata')::jsonb,
    (data->>'dependsOnResources')::jsonb,
    (data->>'dependsOnChildren')::jsonb,
    (data->>'failureReasons')::jsonb,
    (data->>'drifted')::boolean,
    TO_TIMESTAMP((data->>'lastDriftDetectedTimestamp')::bigint),
    (data->>'durations')::jsonb
FROM temp_resources;

DROP TABLE IF EXISTS temp_resources;


-- Blueprint instance resources records
CREATE TABLE IF NOT EXISTS temp_blueprint_instance_resources (data jsonb);
\COPY temp_blueprint_instance_resources (data) FROM 'postgres/__testdata/seed/tmp/blueprint-instance-resources.nd.json';

INSERT INTO blueprint_instance_resources (
    instance_id,
    resource_name,
    resource_id
)
SELECT
    (data->>'instanceId')::uuid,
    data->>'resourceName',
    (data->>'resourceId')::uuid
FROM temp_blueprint_instance_resources;

DROP TABLE IF EXISTS temp_blueprint_instance_resources;


-- Resource drift records
CREATE TABLE IF NOT EXISTS temp_resource_drift (data jsonb);
\COPY temp_resource_drift (data) FROM 'postgres/__testdata/seed/tmp/resource-drift.nd.json';

INSERT INTO resource_drift (
    resource_id,
    instance_id,
    drifted_spec_data,
    difference,
    "timestamp"
)
SELECT
    (data->>'resourceId')::uuid,
    (data->>'instanceId')::uuid,
    (data->>'specData')::jsonb,
    (data->>'difference')::jsonb,
    TO_TIMESTAMP((data->>'timestamp')::bigint)
FROM temp_resource_drift;

DROP TABLE IF EXISTS temp_resource_drift;


-- Link records
CREATE TABLE IF NOT EXISTS temp_links (data jsonb);
\COPY temp_links (data) FROM 'postgres/__testdata/seed/tmp/links.nd.json';

INSERT INTO links (
    id,
    "status",
    precise_status,
    last_status_update_timestamp,
    last_deployed_timestamp,
    last_deploy_attempt_timestamp,
    intermediary_resources_state,
    "data",
    resource_data_mappings,
    failure_reasons,
    durations
)
SELECT
    (data->>'id')::uuid,
    (data->>'status')::smallint,
    (data->>'preciseStatus')::smallint,
    TO_TIMESTAMP((data->>'lastStatusUpdateTimestamp')::bigint),
    TO_TIMESTAMP((data->>'lastDeployedTimestamp')::bigint),
    TO_TIMESTAMP((data->>'lastDeployAttemptTimestamp')::bigint),
    (data->>'intermediaryResourcesState')::jsonb,
    (data->>'data')::jsonb,
    (data->>'resourceDataMappings')::jsonb,
    (data->>'failureReasons')::jsonb,
    (data->>'durations')::jsonb
FROM temp_links;

DROP TABLE IF EXISTS temp_links;


-- Blueprint instance links records
CREATE TABLE IF NOT EXISTS temp_blueprint_instance_links (data jsonb);
\COPY temp_blueprint_instance_links (data) FROM 'postgres/__testdata/seed/tmp/blueprint-instance-links.nd.json';

INSERT INTO blueprint_instance_links (
    instance_id,
    link_name,
    link_id
)
SELECT
    (data->>'instanceId')::uuid,
    data->>'linkName',
    (data->>'linkId')::uuid
FROM temp_blueprint_instance_links;

DROP TABLE IF EXISTS temp_blueprint_instance_links;


-- Link drift records
CREATE TABLE IF NOT EXISTS temp_link_drift (data jsonb);
\COPY temp_link_drift (data) FROM 'postgres/__testdata/seed/tmp/link-drift.nd.json';

INSERT INTO link_drift (
    link_id,
    instance_id,
    resource_a_drift,
    resource_b_drift,
    intermediary_drift,
    "timestamp"
)
SELECT
    (data->>'linkId')::uuid,
    (data->>'instanceId')::uuid,
    (data->>'resourceADrift')::jsonb,
    (data->>'resourceBDrift')::jsonb,
    (data->>'intermediaryDrift')::jsonb,
    TO_TIMESTAMP((data->>'timestamp')::bigint)
FROM temp_link_drift;

DROP TABLE IF EXISTS temp_link_drift;


-- Event records
CREATE TABLE IF NOT EXISTS temp_events (data jsonb);
\COPY temp_events (data) FROM 'postgres/__testdata/seed/tmp/events.nd.json';
INSERT INTO events (
    id,
    "type",
    "channel_type",
    "channel_id",
    "data",
    "timestamp",
    "end"
)
SELECT
    (data->>'id')::uuid,
    data->>'type',
    data->>'channelType',
    (data->>'channelId')::uuid,
    (data->>'data')::jsonb,
    TO_TIMESTAMP((data->>'timestamp')::bigint),
    (data->>'end')::boolean
FROM temp_events;

DROP TABLE IF EXISTS temp_events;

-- Change set records
CREATE TABLE IF NOT EXISTS temp_changesets (data jsonb);
\COPY temp_changesets (data) FROM 'postgres/__testdata/seed/tmp/changesets.nd.json';
INSERT INTO changesets (
    id,
    instance_id,
    destroy,
    "status",
    blueprint_location,
    "changes",
    created
)
SELECT
    (data->>'id')::uuid,
    (data->>'instanceId')::uuid,
    (data->>'destroy')::boolean,
    data->>'status',
    data->>'blueprintLocation',
    (data->>'changes')::jsonb,
    TO_TIMESTAMP((data->>'created')::bigint)
FROM temp_changesets;

DROP TABLE IF EXISTS temp_changesets;

-- Blueprint validation records
CREATE TABLE IF NOT EXISTS temp_blueprint_validations (data jsonb);
\COPY temp_blueprint_validations (data) FROM 'postgres/__testdata/seed/tmp/blueprint-validations.nd.json';
INSERT INTO blueprint_validations (
    id,
    "status",
    blueprint_location,
    created
)
SELECT
    (data->>'id')::uuid,
    data->>'status',
    data->>'blueprintLocation',
    TO_TIMESTAMP((data->>'created')::bigint)
FROM temp_blueprint_validations;

DROP TABLE IF EXISTS temp_blueprint_validations;

-- Reconciliation result records
CREATE TABLE IF NOT EXISTS temp_reconciliation_results (data jsonb);
\COPY temp_reconciliation_results (data) FROM 'postgres/__testdata/seed/tmp/reconciliation-results.nd.json';
INSERT INTO reconciliation_results (
    id,
    changeset_id,
    instance_id,
    result,
    created
)
SELECT
    (data->>'id')::uuid,
    (data->>'changesetId')::uuid,
    (data->>'instanceId')::uuid,
    (data->>'result')::jsonb,
    TO_TIMESTAMP((data->>'created')::bigint)
FROM temp_reconciliation_results;

DROP TABLE IF EXISTS temp_reconciliation_results;
