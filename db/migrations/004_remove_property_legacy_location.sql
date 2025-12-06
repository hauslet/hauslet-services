-- +goose Up
-- Remove legacy location_lat/location_lng columns now that we use PostGIS geography.
ALTER TABLE properties
    DROP COLUMN IF EXISTS location_lat,
    DROP COLUMN IF EXISTS location_lng;

-- +goose Down
-- Reintroduce legacy columns without data backfill.
ALTER TABLE properties
    ADD COLUMN IF NOT EXISTS location_lat decimal(10,8),
    ADD COLUMN IF NOT EXISTS location_lng decimal(11,8);
