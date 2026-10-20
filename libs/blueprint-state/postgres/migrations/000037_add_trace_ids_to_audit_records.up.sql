-- Correlates an audited action with the distributed trace it was carried out under,
-- so that a record of who did what leads to the trace of the work it did.
--
-- Nullable, since a deployment that exports no traces records no trace, and a trace
-- a sampler declined has identifiers that resolve to nothing and is recorded as absent.
ALTER TABLE audit_records
    ADD COLUMN IF NOT EXISTS trace_id text,
    ADD COLUMN IF NOT EXISTS span_id text;

-- Looking a trail up by trace is the reverse of the usual read, used to answer what was
-- audited for a trace already being investigated. Partial, because the column is empty
-- for every record written while tracing was off.
CREATE INDEX IF NOT EXISTS audit_records_trace_id_idx
    ON audit_records (trace_id)
    WHERE trace_id IS NOT NULL;
