-- events.timestamp - critical for time-range queries on event data
CREATE INDEX IF NOT EXISTS idx_events_timestamp
    ON events ("timestamp" DESC);

-- events composite index - time-ordered events by channel (common query pattern)
CREATE INDEX IF NOT EXISTS idx_events_channel_timestamp
    ON events (channel_type, channel_id, "timestamp" DESC);

-- events.type - filtering events by type
CREATE INDEX IF NOT EXISTS idx_events_type
    ON events ("type");

-- changesets.created - for time-range changeset queries
CREATE INDEX IF NOT EXISTS idx_changesets_created
    ON changesets (created DESC);

-- resource_drift.timestamp - for drift detection time queries
CREATE INDEX IF NOT EXISTS idx_resource_drift_timestamp
    ON resource_drift ("timestamp" DESC);

-- link_drift.timestamp - for drift detection time queries
CREATE INDEX IF NOT EXISTS idx_link_drift_timestamp
    ON link_drift ("timestamp" DESC);

-- resources.last_drift_detected_timestamp - for drift reporting
CREATE INDEX IF NOT EXISTS idx_resources_last_drift_detected
    ON resources (last_drift_detected_timestamp DESC)
    WHERE last_drift_detected_timestamp IS NOT NULL;

-- links.last_drift_detected_timestamp - for drift reporting
CREATE INDEX IF NOT EXISTS idx_links_last_drift_detected
    ON links (last_drift_detected_timestamp DESC)
    WHERE last_drift_detected_timestamp IS NOT NULL;

-- blueprint_instances.last_deployed_timestamp - for deployment history queries
CREATE INDEX IF NOT EXISTS idx_blueprint_instances_last_deployed
    ON blueprint_instances (last_deployed_timestamp DESC)
    WHERE last_deployed_timestamp IS NOT NULL;
