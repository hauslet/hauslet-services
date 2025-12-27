-- +goose Up
-- Add booking_id to transactions table
-- This migration adds a booking reference to payment transactions.

ALTER TABLE transactions
ADD COLUMN IF NOT EXISTS booking_id UUID;

CREATE INDEX IF NOT EXISTS idx_transactions_booking_id ON transactions(booking_id);

COMMENT ON COLUMN transactions.booking_id IS 'Associated booking ID (if applicable)';

-- +goose Down
-- Remove booking_id from transactions table
DROP INDEX IF EXISTS idx_transactions_booking_id;

ALTER TABLE transactions
DROP COLUMN IF EXISTS booking_id;
