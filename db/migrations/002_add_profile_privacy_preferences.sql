-- +goose Up
-- Add profile privacy and preference toggles
ALTER TABLE profiles
    ADD COLUMN IF NOT EXISTS bio_visible                  BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS allow_personalized_offers    BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS enable_performance_analytics BOOLEAN NOT NULL DEFAULT FALSE;

-- +goose Down
-- Remove profile privacy and preference toggles
ALTER TABLE profiles
    DROP COLUMN IF EXISTS enable_performance_analytics,
    DROP COLUMN IF EXISTS allow_personalized_offers,
    DROP COLUMN IF EXISTS bio_visible;
