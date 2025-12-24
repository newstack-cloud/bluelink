-- changesets.instance_id - heavily used for filtering changesets by blueprint instance
CREATE INDEX IF NOT EXISTS idx_changesets_instance_id
    ON changesets (instance_id);

-- resource_drift.instance_id - required for efficient joins when querying drift by instance
CREATE INDEX IF NOT EXISTS idx_resource_drift_instance_id
    ON resource_drift (instance_id);

-- link_drift.instance_id - required for efficient joins when querying drift by instance
CREATE INDEX IF NOT EXISTS idx_link_drift_instance_id
    ON link_drift (instance_id);
