-- +goose Up
-- Migration: Add indexes for listing type-specific search filters
-- Purpose: Optimize JSONB queries for shortlet, rental, and sale detail fields
-- Created: 2025-12-20

-- ====================
-- Shortlet Indexes
-- ====================

-- GIN index for general shortlet_details queries
CREATE INDEX IF NOT EXISTS idx_listings_shortlet_details_gin
ON listings USING gin (shortlet_details)
WHERE listing_type = 'shortlet';

-- B-tree indexes for frequently filtered numeric fields
CREATE INDEX IF NOT EXISTS idx_shortlet_min_nights
ON listings (((shortlet_details->>'min_nights')::int))
WHERE listing_type = 'shortlet' AND shortlet_details->>'min_nights' IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_shortlet_max_nights
ON listings (((shortlet_details->>'max_nights')::int))
WHERE listing_type = 'shortlet' AND shortlet_details->>'max_nights' IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_shortlet_max_guests
ON listings (((shortlet_details->>'max_guests')::int))
WHERE listing_type = 'shortlet' AND shortlet_details->>'max_guests' IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_shortlet_extra_guest_fee
ON listings (((shortlet_details->>'extra_guest_fee')::float))
WHERE listing_type = 'shortlet' AND shortlet_details->>'extra_guest_fee' IS NOT NULL;

-- Text index for accommodation_type
CREATE INDEX IF NOT EXISTS idx_shortlet_accommodation_type
ON listings ((shortlet_details->>'accommodation_type'))
WHERE listing_type = 'shortlet' AND shortlet_details->>'accommodation_type' IS NOT NULL;

-- ====================
-- Rental Indexes
-- ====================

-- GIN index for general rental_details queries
CREATE INDEX IF NOT EXISTS idx_listings_rental_details_gin
ON listings USING gin (rental_details)
WHERE listing_type = 'rent';

-- B-tree indexes for rental period fields
CREATE INDEX IF NOT EXISTS idx_rental_min_period
ON listings (((rental_details->>'min_rental_period')::int))
WHERE listing_type = 'rent' AND rental_details->>'min_rental_period' IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_rental_max_period
ON listings (((rental_details->>'max_rental_period')::int))
WHERE listing_type = 'rent' AND rental_details->>'max_rental_period' IS NOT NULL;

-- Text index for rental_price_period
CREATE INDEX IF NOT EXISTS idx_rental_price_period
ON listings ((rental_details->>'rental_price_period'))
WHERE listing_type = 'rent' AND rental_details->>'rental_price_period' IS NOT NULL;

-- Note: Timestamp index for rental_availability_from is omitted because text->timestamp casting
-- is not IMMUTABLE in PostgreSQL. The GIN index on rental_details will still help with these queries.

-- ====================
-- Sale Indexes
-- ====================

-- GIN index for general sale_details queries
CREATE INDEX IF NOT EXISTS idx_listings_sale_details_gin
ON listings USING gin (sale_details)
WHERE listing_type = 'sale';

-- Text index for ownership_title
CREATE INDEX IF NOT EXISTS idx_sale_ownership_title
ON listings ((sale_details->>'ownership_title'))
WHERE listing_type = 'sale' AND sale_details->>'ownership_title' IS NOT NULL;

-- Boolean index for payment_plan
CREATE INDEX IF NOT EXISTS idx_sale_payment_plan
ON listings (((sale_details->>'payment_plan')::boolean))
WHERE listing_type = 'sale' AND sale_details->>'payment_plan' IS NOT NULL;

-- Note: Timestamp index for sale_availability_from is omitted because text->timestamp casting
-- is not IMMUTABLE in PostgreSQL. The GIN index on sale_details will still help with these queries.

-- ====================
-- Property Indexes (if not already exist)
-- ====================

-- Index for property_class
CREATE INDEX IF NOT EXISTS idx_properties_class ON properties (property_class);

-- Index for property_condition
CREATE INDEX IF NOT EXISTS idx_properties_condition ON properties (property_condition);

-- GIN index for amenities JSONB queries (ALL amenities AND logic)
CREATE INDEX IF NOT EXISTS idx_properties_amenities_gin ON properties USING gin (amenities);

-- ====================
-- Composite Indexes for Common Queries
-- ====================

-- Composite index for published + listing_type (most common filter combination)
CREATE INDEX IF NOT EXISTS idx_listings_published_type
ON listings (published, listing_type, status)
WHERE deleted_at IS NULL;

-- ====================
-- Index Comments for Documentation
-- ====================

COMMENT ON INDEX idx_listings_shortlet_details_gin IS 'GIN index for shortlet JSONB searches - supports general JSONB queries';
COMMENT ON INDEX idx_listings_rental_details_gin IS 'GIN index for rental JSONB searches - supports general JSONB queries';
COMMENT ON INDEX idx_listings_sale_details_gin IS 'GIN index for sale JSONB searches - supports general JSONB queries';
COMMENT ON INDEX idx_properties_amenities_gin IS 'GIN index for amenities AND-logic filtering - supports nested JSONB queries';
COMMENT ON INDEX idx_listings_published_type IS 'Composite index for most common search filter combination';


-- +goose Down
-- Remove all indexes created in this migration in reverse order

-- Remove index comments (optional, but clean)
COMMENT ON INDEX idx_listings_published_type IS NULL;
COMMENT ON INDEX idx_properties_amenities_gin IS NULL;
COMMENT ON INDEX idx_listings_sale_details_gin IS NULL;
COMMENT ON INDEX idx_listings_rental_details_gin IS NULL;
COMMENT ON INDEX idx_listings_shortlet_details_gin IS NULL;

-- Drop composite indexes
DROP INDEX IF EXISTS idx_listings_published_type;

-- Drop property indexes (only if they were created in this migration - be careful!)
-- DROP INDEX IF EXISTS idx_properties_amenities_gin;
-- DROP INDEX IF EXISTS idx_properties_condition;
-- DROP INDEX IF EXISTS idx_properties_class;

-- Drop sale indexes
DROP INDEX IF EXISTS idx_sale_payment_plan;
DROP INDEX IF EXISTS idx_sale_ownership_title;
DROP INDEX IF EXISTS idx_listings_sale_details_gin;

-- Drop rental indexes
DROP INDEX IF EXISTS idx_rental_price_period;
DROP INDEX IF EXISTS idx_rental_max_period;
DROP INDEX IF EXISTS idx_rental_min_period;
DROP INDEX IF EXISTS idx_listings_rental_details_gin;

-- Drop shortlet indexes
DROP INDEX IF EXISTS idx_shortlet_accommodation_type;
DROP INDEX IF EXISTS idx_shortlet_extra_guest_fee;
DROP INDEX IF EXISTS idx_shortlet_max_guests;
DROP INDEX IF EXISTS idx_shortlet_max_nights;
DROP INDEX IF EXISTS idx_shortlet_min_nights;
DROP INDEX IF EXISTS idx_listings_shortlet_details_gin;