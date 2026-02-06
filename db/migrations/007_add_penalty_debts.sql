-- Add penalty debts table for unpaid host cancellation penalties
CREATE TABLE IF NOT EXISTS penalty_debts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    host_id UUID NOT NULL REFERENCES users(id),
    booking_id UUID NOT NULL UNIQUE REFERENCES bookings(id),
    amount BIGINT NOT NULL,
    outstanding_amount BIGINT NOT NULL,
    currency VARCHAR(3) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    settled_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_penalty_debts_host_status ON penalty_debts(host_id, status);
CREATE INDEX IF NOT EXISTS idx_penalty_debts_status ON penalty_debts(status);
