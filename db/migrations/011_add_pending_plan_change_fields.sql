-- +goose Up
-- Add pending plan change fields to agent_subscriptions for Option A upgrade/downgrade flow

ALTER TABLE agent_subscriptions
ADD COLUMN IF NOT EXISTS pending_plan_type VARCHAR(50),
ADD COLUMN IF NOT EXISTS pending_plan_scheduled_at TIMESTAMPTZ;

COMMENT ON COLUMN agent_subscriptions.pending_plan_type IS 'Scheduled plan change to be applied at next billing cycle (Option A upgrade flow)';
COMMENT ON COLUMN agent_subscriptions.pending_plan_scheduled_at IS 'Timestamp when the plan change was scheduled';

-- +goose Down
-- Remove pending plan change fields

ALTER TABLE agent_subscriptions
DROP COLUMN IF EXISTS pending_plan_scheduled_at,
DROP COLUMN IF EXISTS pending_plan_type;
