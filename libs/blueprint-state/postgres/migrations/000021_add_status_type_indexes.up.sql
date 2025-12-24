-- resources.type - frequently used for filtering resources by type
CREATE INDEX IF NOT EXISTS idx_resources_type
    ON resources (type);

-- resources.template_name - used for template-based filtering
CREATE INDEX IF NOT EXISTS idx_resources_template_name
    ON resources (template_name);

-- resources.status - used for finding resources in specific states
CREATE INDEX IF NOT EXISTS idx_resources_status
    ON resources (status);

-- links.status - used for finding links in specific states
CREATE INDEX IF NOT EXISTS idx_links_status
    ON links (status);

-- changesets.status - used for finding changesets in specific states (pending, running, etc.)
CREATE INDEX IF NOT EXISTS idx_changesets_status
    ON changesets (status);

-- Composite index for common query pattern: find changesets by instance and status
CREATE INDEX IF NOT EXISTS idx_changesets_instance_status
    ON changesets (instance_id, status);

-- blueprint_instances.status - used for finding instances in specific states
CREATE INDEX IF NOT EXISTS idx_blueprint_instances_status
    ON blueprint_instances (status);

-- Partial indexes for boolean flags - smaller index size, faster scans
-- Only index rows where drifted=true since that's the common query pattern
CREATE INDEX IF NOT EXISTS idx_resources_drifted
    ON resources (id) WHERE drifted = true;

CREATE INDEX IF NOT EXISTS idx_links_drifted
    ON links (id) WHERE drifted = true;
