-- +goose Up
-- Add is_moderated field to profiles table to track moderation status
ALTER TABLE profiles
    ADD COLUMN IF NOT EXISTS is_moderated boolean NOT NULL DEFAULT false;

-- Create index for faster queries filtering by moderation status
CREATE INDEX IF NOT EXISTS idx_profiles_is_moderated ON profiles(is_moderated);

-- +goose Down
-- Remove moderation status index
DROP INDEX IF EXISTS idx_profiles_is_moderated;

-- Remove is_moderated column
ALTER TABLE profiles
    DROP COLUMN IF EXISTS is_moderated;