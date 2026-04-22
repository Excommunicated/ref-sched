-- Remove acknowledgment tracking from match_roles
DROP INDEX IF EXISTS idx_match_roles_acknowledged;

ALTER TABLE match_roles
    DROP COLUMN acknowledged,
    DROP COLUMN acknowledged_at;
