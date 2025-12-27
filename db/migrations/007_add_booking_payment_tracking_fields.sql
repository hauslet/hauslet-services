-- +goose Up
-- Add payment tracking fields to bookings table
-- This migration adds payment_reference and last_payment_id fields to support
-- payment-booking integration

-- Add payment reference field to track the associated payment reference
ALTER TABLE bookings
ADD COLUMN IF NOT EXISTS payment_reference VARCHAR(255);

-- Add last payment ID field to track the most recent payment attempt
ALTER TABLE bookings
ADD COLUMN IF NOT EXISTS last_payment_id UUID;

-- Add indexes for efficient queries
CREATE INDEX IF NOT EXISTS idx_bookings_payment_reference ON bookings(payment_reference);
CREATE INDEX IF NOT EXISTS idx_bookings_last_payment_id ON bookings(last_payment_id);

-- Add comments for documentation
COMMENT ON COLUMN bookings.payment_reference IS 'Reference ID of the payment (e.g., PAY-PMT-12345)';
COMMENT ON COLUMN bookings.last_payment_id IS 'UUID of the most recent payment attempt for this booking';

-- +goose Down
-- Remove payment tracking fields from bookings table
DROP INDEX IF EXISTS idx_bookings_last_payment_id;
DROP INDEX IF EXISTS idx_bookings_payment_reference;

ALTER TABLE bookings
DROP COLUMN IF EXISTS last_payment_id,
DROP COLUMN IF EXISTS payment_reference;
