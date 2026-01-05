-- +goose Up
-- Add payment_method_id column to agent_subscriptions table for recurring billing
ALTER TABLE agent_subscriptions
ADD COLUMN payment_method_id UUID;

-- Add index for faster lookups by payment method
CREATE INDEX idx_subscription_payment_method ON agent_subscriptions(payment_method_id) WHERE payment_method_id IS NOT NULL AND deleted_at IS NULL;

-- Add comment to document the column purpose
COMMENT ON COLUMN agent_subscriptions.payment_method_id IS 'Reference to saved payment method for automatic recurring billing';

-- +goose Down
DROP INDEX IF EXISTS idx_subscription_payment_method;
ALTER TABLE agent_subscriptions
DROP COLUMN IF EXISTS payment_method_id;
