-- Day-level unavailability tracking
-- Allows referees to mark entire days as unavailable
CREATE TABLE day_unavailability (
    id BIGSERIAL PRIMARY KEY,
    referee_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    unavailable_date DATE NOT NULL,
    reason TEXT,  -- Optional reason for unavailability
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(referee_id, unavailable_date)  -- One record per referee per date
);

CREATE INDEX idx_day_unavailability_referee_id ON day_unavailability(referee_id);
CREATE INDEX idx_day_unavailability_date ON day_unavailability(unavailable_date);
