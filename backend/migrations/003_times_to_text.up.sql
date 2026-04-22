-- Convert time columns from TIME to TEXT for simpler handling
-- TIME type can have unexpected timezone conversion behavior
-- Since we're dealing with simple wall-clock times (e.g., "9:00 AM"), TEXT is more appropriate

ALTER TABLE matches
    ALTER COLUMN start_time TYPE TEXT,
    ALTER COLUMN end_time TYPE TEXT;
