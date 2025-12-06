-- +goose Up
-- Add auditing and reactivation metadata to users
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS deactivated_at      TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS deactivated_reason  TEXT,
    ADD COLUMN IF NOT EXISTS deactivated_by      UUID,
    ADD COLUMN IF NOT EXISTS reactivate_on_login BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS reactivated_at      TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS reactivated_by      UUID;

-- +goose Down
-- Remove auditing and reactivation metadata from users
ALTER TABLE users
    DROP COLUMN IF EXISTS reactivated_by,
    DROP COLUMN IF EXISTS reactivated_at,
    DROP COLUMN IF EXISTS reactivate_on_login,
    DROP COLUMN IF EXISTS deactivated_by,
    DROP COLUMN IF EXISTS deactivated_reason,
    DROP COLUMN IF EXISTS deactivated_at;
