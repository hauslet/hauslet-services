-- +goose Up
-- Update booking check-in/out fields to support scheduled and actual timestamps

-- Allow actual check-in/out timestamps to be nullable
ALTER TABLE bookings
ALTER COLUMN check_in DROP NOT NULL,
ALTER COLUMN check_out DROP NOT NULL;

-- Create new scheduled timestamp columns
ALTER TABLE bookings
ADD COLUMN IF NOT EXISTS check_in_time_tmp TIMESTAMPTZ,
ADD COLUMN IF NOT EXISTS check_out_time_tmp TIMESTAMPTZ;

-- Backfill scheduled timestamps from legacy string values
UPDATE bookings
SET
    check_in_time_tmp = CASE
        WHEN check_in_time IS NULL THEN NULL
        WHEN check_in_time::text ~ '^\d{4}-\d{2}-\d{2}' THEN check_in_time::timestamptz
        WHEN check_in IS NOT NULL THEN date_trunc('day', check_in) + (check_in_time::time)
        ELSE NULL
    END,
    check_out_time_tmp = CASE
        WHEN check_out_time IS NULL THEN NULL
        WHEN check_out_time::text ~ '^\d{4}-\d{2}-\d{2}' THEN check_out_time::timestamptz
        WHEN check_out IS NOT NULL THEN date_trunc('day', check_out) + (check_out_time::time)
        ELSE NULL
    END;

-- Replace legacy columns with the scheduled timestamp columns
ALTER TABLE bookings
DROP COLUMN IF EXISTS check_in_time,
DROP COLUMN IF EXISTS check_out_time;

ALTER TABLE bookings
RENAME COLUMN check_in_time_tmp TO check_in_time;

ALTER TABLE bookings
RENAME COLUMN check_out_time_tmp TO check_out_time;

-- Index scheduled timestamps for range queries
CREATE INDEX IF NOT EXISTS idx_bookings_check_in_time ON bookings (check_in_time);
CREATE INDEX IF NOT EXISTS idx_bookings_check_out_time ON bookings (check_out_time);

-- +goose Down
-- Revert scheduled timestamps to text columns (non-lossless)
DROP INDEX IF EXISTS idx_bookings_check_out_time;
DROP INDEX IF EXISTS idx_bookings_check_in_time;

ALTER TABLE bookings
ALTER COLUMN check_in_time TYPE VARCHAR(32) USING check_in_time::text,
ALTER COLUMN check_out_time TYPE VARCHAR(32) USING check_out_time::text;
