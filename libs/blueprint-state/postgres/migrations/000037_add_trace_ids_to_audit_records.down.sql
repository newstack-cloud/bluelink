DROP INDEX IF EXISTS audit_records_trace_id_idx;

ALTER TABLE audit_records
    DROP COLUMN IF EXISTS trace_id,
    DROP COLUMN IF EXISTS span_id;
