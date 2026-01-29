package database

import (
	"fmt"
	"log/slog"

	"gorm.io/gorm"
)

// RunMigrations runs all database migrations
// This should be called once during application startup
func RunMigrations(db *gorm.DB, log *slog.Logger, models ...interface{}) error {
	// Step 1: Setup PostgreSQL extensions (if needed)
	// Note: These may require superuser privileges
	// In production, these should ideally be run manually by a DBA
	if err := setupExtensions(db); err != nil {
		// Extensions are optional - just log and continue
		log.Info(": Skipping extensions setup (may require superuser)", "error", err)
	}

	// Step 2: Auto-migrate all models
	if err := db.AutoMigrate(models...); err != nil {
		return fmt.Errorf("failed to auto-migrate models: %w", err)
	}

	return nil
}

// setupExtensions creates PostgreSQL extensions
// Only includes extensions actually needed for the application
func setupExtensions(db *gorm.DB) error {
	// postgis: For geospatial data types (geography, geometry)
	// Only needed if using location-based features
	if err := db.Exec(`CREATE EXTENSION IF NOT EXISTS "postgis"`).Error; err != nil {
		return fmt.Errorf("failed to create postgis extension: %w", err)
	}

	if err := db.Exec(`CREATE EXTENSION IF NOT EXISTS "uuid-ossp"`).Error; err != nil {
		return fmt.Errorf("failed to create uuid-ossp extension: %w", err)
	}

	if err := db.Exec(`CREATE EXTENSION IF NOT EXISTS "vector"`).Error; err != nil {
		return fmt.Errorf("failed to create vector extension: %w", err)
	}

	// Add search_vector column for Full Text Search
	if err := db.Exec(`
		ALTER TABLE listings
		ADD COLUMN IF NOT EXISTS search_vector tsvector
		GENERATED ALWAYS AS (to_tsvector('english', coalesce(title, '') || ' ' || coalesce(description, '') || ' ' || coalesce(extra_description, ''))) STORED;
	`).Error; err != nil {
		return fmt.Errorf("failed to add search_vector column: %w", err)
	}

	// Add GIN index for search_vector
	if err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_listings_search_vector ON listings USING GIN(search_vector)`).Error; err != nil {
		return fmt.Errorf("failed to create search_vector index: %w", err)
	}

	return nil
}
