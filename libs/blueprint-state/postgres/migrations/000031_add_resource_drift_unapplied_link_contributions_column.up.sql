ALTER TABLE IF EXISTS resource_drift
  ADD COLUMN IF NOT EXISTS unapplied_link_contributions jsonb;
