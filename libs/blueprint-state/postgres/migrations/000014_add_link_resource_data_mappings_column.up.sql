ALTER TABLE IF EXISTS links
  ADD COLUMN IF NOT EXISTS resource_data_mappings jsonb;
