CREATE TABLE IF NOT EXISTS audit_records (
    id uuid PRIMARY KEY,
    "timestamp" timestamptz NOT NULL,
    "action" varchar(255) NOT NULL,
    target text,
    outcome varchar(32) NOT NULL,
    status_code integer NOT NULL,
    "method" varchar(16) NOT NULL,
    "path" text NOT NULL,
    principal_method varchar(32) NOT NULL,
    principal_subject text NOT NULL,
    principal_name text,
    on_behalf_of text,
    source_ip text,
    user_agent text,
    request_id text
);

-- The audit trail is read as a time range, optionally narrowed to one action or
-- one principal, and is ordered by (timestamp, id) so that extraction can
-- resume from a timestamp without re-shipping or skipping part of a second.
CREATE INDEX IF NOT EXISTS audit_records_timestamp_id_idx
    ON audit_records ("timestamp", id);
CREATE INDEX IF NOT EXISTS audit_records_action_timestamp_idx
    ON audit_records ("action", "timestamp");
CREATE INDEX IF NOT EXISTS audit_records_principal_subject_timestamp_idx
    ON audit_records (principal_subject, "timestamp");
