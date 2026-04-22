-- Add explicit available/unavailable tracking to availability table
-- Previously: record exists = available, no record = unavailable/no preference
-- Now: record with available=true = available, available=false = unavailable, no record = no preference

-- Step 1: Add column as nullable with default (avoids table rewrite/lock)
ALTER TABLE availability ADD COLUMN IF NOT EXISTS available BOOLEAN DEFAULT true;

-- Step 2: Update any NULL values to true (existing records were implicitly available)
UPDATE availability SET available = true WHERE available IS NULL;

-- Step 3: Make column NOT NULL now that all values are set
ALTER TABLE availability ALTER COLUMN available SET NOT NULL;

-- Create index for filtering by availability status
CREATE INDEX IF NOT EXISTS idx_availability_status ON availability(referee_id, available);
