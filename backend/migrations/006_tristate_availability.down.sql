-- Revert tri-state availability changes
DROP INDEX IF EXISTS idx_availability_status;
ALTER TABLE availability DROP COLUMN IF EXISTS available;
