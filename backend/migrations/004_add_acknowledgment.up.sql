-- Add acknowledgment tracking to match_roles
ALTER TABLE match_roles
    ADD COLUMN acknowledged BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN acknowledged_at TIMESTAMP;

-- Add index for finding unacknowledged assignments
CREATE INDEX idx_match_roles_acknowledged ON match_roles(acknowledged, acknowledged_at);
