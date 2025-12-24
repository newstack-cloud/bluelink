CREATE TABLE IF NOT EXISTS link_drift (
    id SERIAL PRIMARY KEY,
    link_id uuid NOT NULL UNIQUE,
    instance_id uuid,
    resource_a_drift jsonb,
    resource_b_drift jsonb,
    intermediary_drift jsonb,
    "timestamp" timestamptz,
    FOREIGN KEY (link_id) REFERENCES links (id) ON DELETE CASCADE,
    FOREIGN KEY (instance_id) REFERENCES blueprint_instances (id)
);
