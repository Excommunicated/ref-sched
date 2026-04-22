-- Create users table
CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    google_id VARCHAR(255) UNIQUE NOT NULL,
    email VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'pending_referee',
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    date_of_birth DATE,
    certified BOOLEAN DEFAULT FALSE,
    cert_expiry DATE,
    grade VARCHAR(20),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Create index on google_id for fast lookups
CREATE INDEX idx_users_google_id ON users(google_id);

-- Create index on email
CREATE INDEX idx_users_email ON users(email);

-- Create index on role and status for filtering
CREATE INDEX idx_users_role_status ON users(role, status);

-- Add comments
COMMENT ON TABLE users IS 'Users table for referees and assignors';
COMMENT ON COLUMN users.role IS 'User role: pending_referee, referee, or assignor';
COMMENT ON COLUMN users.status IS 'User status: pending, active, inactive, or removed';
COMMENT ON COLUMN users.grade IS 'Assignor-managed referee grade: Junior, Mid, or Senior';
