ALTER TABLE links
  ADD COLUMN IF NOT EXISTS contribution_records jsonb;
