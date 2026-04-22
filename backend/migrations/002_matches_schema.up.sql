-- Matches table for imported schedule
-- NOTE: All dates and times are stored in US Eastern Time (America/New_York)
-- Stack Team App exports are in Eastern Time, and all local matches are in Eastern Time
CREATE TABLE matches (
    id BIGSERIAL PRIMARY KEY,
    event_name VARCHAR(255) NOT NULL,
    team_name VARCHAR(255) NOT NULL,
    age_group VARCHAR(10),  -- U6, U8, U10, U12, U14, etc. or NULL if parsing failed
    match_date DATE NOT NULL,  -- Date in US Eastern Time
    start_time TIME NOT NULL,  -- Time in US Eastern Time
    end_time TIME NOT NULL,    -- Time in US Eastern Time
    location VARCHAR(500) NOT NULL,
    description TEXT,  -- Contains field info and meeting time
    reference_id VARCHAR(100),  -- From Stack Team App, may be duplicate
    status VARCHAR(20) NOT NULL DEFAULT 'active',  -- active, cancelled
    created_by BIGINT REFERENCES users(id),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Index for duplicate detection
CREATE INDEX idx_matches_reference_id ON matches(reference_id);
CREATE INDEX idx_matches_date_time_location ON matches(match_date, start_time, location);
CREATE INDEX idx_matches_status ON matches(status);
CREATE INDEX idx_matches_date ON matches(match_date);

-- Match role slots (Center Referee, Assistant Referee 1, Assistant Referee 2)
CREATE TABLE match_roles (
    id BIGSERIAL PRIMARY KEY,
    match_id BIGINT NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
    role_type VARCHAR(20) NOT NULL,  -- center, assistant_1, assistant_2
    assigned_referee_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(match_id, role_type)  -- Only one center, one assistant_1, etc. per match
);

CREATE INDEX idx_match_roles_match_id ON match_roles(match_id);
CREATE INDEX idx_match_roles_referee_id ON match_roles(assigned_referee_id);

-- Referee availability marking
CREATE TABLE availability (
    id BIGSERIAL PRIMARY KEY,
    match_id BIGINT NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
    referee_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    available BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(match_id, referee_id)  -- One availability record per match per referee
);

CREATE INDEX idx_availability_match_id ON availability(match_id);
CREATE INDEX idx_availability_referee_id ON availability(referee_id);

-- Assignment history for audit trail
CREATE TABLE assignment_history (
    id BIGSERIAL PRIMARY KEY,
    match_id BIGINT NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
    role_type VARCHAR(20) NOT NULL,  -- center, assistant_1, assistant_2
    referee_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    action VARCHAR(20) NOT NULL,  -- assigned, removed
    actor_id BIGINT NOT NULL REFERENCES users(id),  -- Assignor who made the change
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_assignment_history_match_id ON assignment_history(match_id);
CREATE INDEX idx_assignment_history_referee_id ON assignment_history(referee_id);
