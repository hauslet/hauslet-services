-- +goose Up
-- Enforce that every listing references a real property and is unique per property.
-- ALTER TABLE listings
--     ADD CONSTRAINT listings_property_fk
--         FOREIGN KEY (property_id) REFERENCES properties(id)
--         ON UPDATE CASCADE
--         ON DELETE CASCADE;

-- CREATE UNIQUE INDEX IF NOT EXISTS ux_listings_property_id
--     ON listings(property_id);

-- -- +goose Down
-- DROP INDEX IF EXISTS ux_listings_property_id;
-- ALTER TABLE listings
--     DROP CONSTRAINT IF EXISTS listings_property_fk;
