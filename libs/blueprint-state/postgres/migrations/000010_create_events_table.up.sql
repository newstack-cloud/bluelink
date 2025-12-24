-- Events table with time-based partitioning for scale
-- Benefits: efficient time-range queries, parallel scans, easy data archival
--
-- PARTITION MANAGEMENT:
-- The default partition catches all inserts. For optimal performance at scale,
-- create monthly partitions proactively using the provided function:
--   SELECT create_events_partition('2025-01-01');
-- Or create multiple months at once:
--   SELECT create_events_partitions_range('2025-01-01', '2026-01-01');
--
-- Old partitions can be archived/dropped:
--   DROP TABLE events_2024_01;

-- Create the partitioned events table
CREATE TABLE IF NOT EXISTS events (
    id uuid NOT NULL,
    "type" varchar(255) NOT NULL,
    channel_type varchar(255) NOT NULL,
    channel_id uuid NOT NULL,
    data jsonb NOT NULL,
    "timestamp" timestamptz NOT NULL,
    "end" boolean NOT NULL,
    PRIMARY KEY (id, "timestamp")
) PARTITION BY RANGE ("timestamp");

-- Default partition catches all rows when no specific partition exists
-- This ensures inserts never fail due to missing partitions
CREATE TABLE IF NOT EXISTS events_default PARTITION OF events DEFAULT;

-- Helper function to create a partition for a specific month
-- Usage: SELECT create_events_partition('2025-01-01');
CREATE OR REPLACE FUNCTION create_events_partition(partition_date DATE)
RETURNS TEXT AS $$
DECLARE
    partition_name TEXT;
    partition_start DATE;
    partition_end DATE;
BEGIN
    partition_start := date_trunc('month', partition_date)::DATE;
    partition_end := (partition_start + INTERVAL '1 month')::DATE;
    partition_name := 'events_' || to_char(partition_start, 'YYYY_MM');

    -- Check if partition already exists
    IF EXISTS (
        SELECT 1 FROM pg_class c
        JOIN pg_namespace n ON n.oid = c.relnamespace
        WHERE c.relname = partition_name
        AND n.nspname = current_schema()
    ) THEN
        RETURN partition_name || ' already exists';
    END IF;

    -- Create the partition
    EXECUTE format(
        'CREATE TABLE %I PARTITION OF events FOR VALUES FROM (%L) TO (%L)',
        partition_name,
        partition_start,
        partition_end
    );

    RETURN partition_name || ' created';
END;
$$ LANGUAGE plpgsql;

-- Helper function to create partitions for a date range
-- Usage: SELECT create_events_partitions_range('2025-01-01', '2026-01-01');
CREATE OR REPLACE FUNCTION create_events_partitions_range(start_date DATE, end_date DATE)
RETURNS TABLE(partition_name TEXT, status TEXT) AS $$
DECLARE
    iter_date DATE := date_trunc('month', start_date)::DATE;
BEGIN
    WHILE iter_date < end_date LOOP
        partition_name := 'events_' || to_char(iter_date, 'YYYY_MM');
        status := create_events_partition(iter_date);
        RETURN NEXT;
        iter_date := iter_date + INTERVAL '1 month';
    END LOOP;
END;
$$ LANGUAGE plpgsql;
