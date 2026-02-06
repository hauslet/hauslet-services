-- +goose Up
-- Add moderation tracking fields to limit redundant processing and notifications

-- 1. Track when moderation outcome notifications were last sent
ALTER TABLE listings
    ADD COLUMN IF NOT EXISTS moderation_notified_at TIMESTAMPTZ;

-- 2. Track when media items were last successfully moderated
ALTER TABLE listing_media
    ADD COLUMN IF NOT EXISTS last_moderated_at TIMESTAMPTZ;

-- Add indexes for performance queries
CREATE INDEX IF NOT EXISTS idx_listing_media_last_moderated_at ON listing_media (last_moderated_at);

-- +goose Down
-- Remove moderation tracking fields

DROP INDEX IF EXISTS idx_listing_media_last_moderated_at;

ALTER TABLE listing_media
    DROP COLUMN IF EXISTS last_moderated_at;

ALTER TABLE listings
    DROP COLUMN IF EXISTS moderation_notified_at;
