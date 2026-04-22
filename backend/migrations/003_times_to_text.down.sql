-- Revert time columns back to TIME type
ALTER TABLE matches
    ALTER COLUMN start_time TYPE TIME USING start_time::TIME,
    ALTER COLUMN end_time TYPE TIME USING end_time::TIME;
