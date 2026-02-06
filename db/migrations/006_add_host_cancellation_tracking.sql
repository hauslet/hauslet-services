-- Migration: Add host cancellation tracking and listing suspension fields

CREATE TABLE host_cancellation_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    host_id UUID NOT NULL REFERENCES users(id),
    booking_id UUID NOT NULL UNIQUE REFERENCES bookings(id),
    cancelled_at TIMESTAMPTZ NOT NULL,
    penalty_amount BIGINT NOT NULL DEFAULT 0,
    penalty_paid BOOLEAN NOT NULL DEFAULT false,
    warning_sent BOOLEAN NOT NULL DEFAULT false,
    reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_host_cancellations_host_id ON host_cancellation_records(host_id);
CREATE INDEX idx_host_cancellations_cancelled_at ON host_cancellation_records(cancelled_at);

ALTER TABLE listings
    ADD COLUMN suspended_until TIMESTAMPTZ,
    ADD COLUMN suspension_reason TEXT;

CREATE INDEX idx_listings_suspended_until ON listings(suspended_until);

-- Rollback: DROP TABLE host_cancellation_records; ALTER TABLE listings DROP COLUMN suspended_until; ALTER TABLE listings DROP COLUMN suspension_reason;
