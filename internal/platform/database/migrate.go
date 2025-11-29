package database

import (
	"fmt"

	"github.com/go-pkgz/lgr"
	"gorm.io/gorm"
)

// RunMigrations runs all database migrations
// This should be called once during application startup
func RunMigrations(db *gorm.DB, log *lgr.Logger, models ...interface{}) error {
	// Step 1: Setup PostgreSQL extensions (if needed)
	// Note: These may require superuser privileges
	// In production, these should ideally be run manually by a DBA
	if err := setupExtensions(db); err != nil {
		// Extensions are optional - just log and continue
		log.Logf("INFO: Skipping extensions setup (may require superuser): %v", err)
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

	return nil
}
