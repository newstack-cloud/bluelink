CREATE TABLE IF NOT EXISTS reconciliation_results (
    id uuid PRIMARY KEY,
    changeset_id uuid NOT NULL,
    instance_id uuid NOT NULL,
    result jsonb NOT NULL,
    created timestamptz NOT NULL,
    FOREIGN KEY (changeset_id) REFERENCES changesets (id)
        ON DELETE CASCADE,
    FOREIGN KEY (instance_id) REFERENCES blueprint_instances (id)
        ON DELETE CASCADE
);

-- Index for looking up results by changeset ID (most common query)
CREATE INDEX IF NOT EXISTS idx_reconciliation_results_changeset_id
    ON reconciliation_results (changeset_id);

-- Index for looking up results by instance ID
CREATE INDEX IF NOT EXISTS idx_reconciliation_results_instance_id
    ON reconciliation_results (instance_id);

-- Composite index for getting latest by changeset (ordered by created desc)
CREATE INDEX IF NOT EXISTS idx_reconciliation_results_changeset_created
    ON reconciliation_results (changeset_id, created DESC);

-- Composite index for getting latest by instance (ordered by created desc)
CREATE INDEX IF NOT EXISTS idx_reconciliation_results_instance_created
    ON reconciliation_results (instance_id, created DESC);
