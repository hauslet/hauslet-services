-- Enable pg_trgm extension for fuzzy matching
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- Add GIN indexes for fast fuzzy search on title and description
CREATE INDEX IF NOT EXISTS idx_listings_title_trgm ON listings USING gin (title gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_listings_description_trgm ON listings USING gin (description gin_trgm_ops);
