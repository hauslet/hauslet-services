-- Migration: Remove Redundant Listing Fields
-- Date: 2026-01-03
-- Reason: ViewCount/LastViewedAt are replaced by Interactions Module
--         FeaturedUntil/BoostLevel are replaced by Promotions Module
--
-- These fields are redundant with the interactions and promotions modules:
-- - ViewCount & LastViewedAt: Replaced by listing_interactions table
-- - FeaturedUntil & BoostLevel: Replaced by listing_promotions table
--
-- This enforces single source of truth:
--   * Interactions Module: All view tracking & analytics
--   * Promotions Module: All featured/boost logic
--   * Listings: Core property/listing data only

-- Remove redundant fields from listings table
ALTER TABLE listings 
DROP COLUMN IF EXISTS view_count,
DROP COLUMN IF EXISTS last_viewed_at,
DROP COLUMN IF EXISTS featured_until,
DROP COLUMN IF EXISTS boost_level;

-- Drop the index on featured_until if it exists
DROP INDEX IF EXISTS idx_listings_featured_until;
