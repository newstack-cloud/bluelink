CREATE TABLE IF NOT EXISTS cleanup_operations (
    id uuid PRIMARY KEY,
    cleanup_type varchar(64) NOT NULL,
    status varchar(32) NOT NULL,
    started_at timestamptz NOT NULL,
    ended_at timestamptz,
    items_deleted bigint DEFAULT 0,
    error_message text,
    threshold_date timestamptz NOT NULL
);

-- Composite index for getting latest operation by type (ordered by started_at desc)
CREATE INDEX IF NOT EXISTS idx_cleanup_operations_type_started
    ON cleanup_operations (cleanup_type, started_at DESC);
