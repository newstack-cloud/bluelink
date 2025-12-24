ALTER TABLE links ADD COLUMN IF NOT EXISTS drifted boolean DEFAULT false;
ALTER TABLE links ADD COLUMN IF NOT EXISTS last_drift_detected_timestamp timestamptz;
