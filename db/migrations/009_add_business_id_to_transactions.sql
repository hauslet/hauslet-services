-- +goose Up
-- Add business_id to transactions table
-- This migration adds a business reference to payment transactions.

ALTER TABLE transactions
ADD COLUMN IF NOT EXISTS business_id UUID;

CREATE INDEX IF NOT EXISTS idx_transactions_business_id ON transactions(business_id);

COMMENT ON COLUMN transactions.business_id IS 'Associated business ID (if applicable)';

-- +goose Down
-- Remove business_id from transactions table
DROP INDEX IF EXISTS idx_transactions_business_id;

ALTER TABLE transactions
DROP COLUMN IF EXISTS business_id;
