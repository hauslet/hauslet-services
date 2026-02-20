package database

import (
	"context"
	"fmt"
	"time"

	"hauslet/config"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// NewPostgres creates a new PostgreSQL database connection using GORM
// It configures logging based on the environment (production = silent, dev = error only)
func NewPostgres(cfg *config.DBConfig, env string) (*gorm.DB, error) {
	if cfg == nil || cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	gormConfig := &gorm.Config{}
	if env == "production" {
		// In production, we disable SQL logging to prevent PII leakage
		gormConfig.Logger = gormlogger.Default.LogMode(gormlogger.Silent)
	} else {
		// In development, only log errors to avoid exposing sensitive data
		gormConfig.Logger = gormlogger.Default.LogMode(gormlogger.Error)
	}

	db, err := gorm.Open(postgres.Open(cfg.DatabaseURL), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	// Set connection pool settings
	// Keep these conservative to avoid exhausting Cloud SQL's max_connections
	// (shared across all service instances). A small Cloud SQL instance may
	// only have 50-100 total slots.
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)
	sqlDB.SetConnMaxIdleTime(5 * time.Minute)

	return db, nil
}

// NewPostgresWithContext creates a database connection with context for timeout control
func NewPostgresWithContext(ctx context.Context, cfg *config.DBConfig, env string) (*gorm.DB, error) {
	db, err := NewPostgres(cfg, env)
	if err != nil {
		return nil, err
	}

	// Test the connection with context
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

// Close closes the database connection
func Close(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %w", err)
	}
	return sqlDB.Close()
}
