package main

import (
	"os"
	"strconv"
)

// SeedConfig holds configuration for seeding the database
type SeedConfig struct {
	// Entity counts
	UserCount     int
	BusinessCount int
	PropertyCount int
	ListingCount  int
	BookingCount  int
	ReviewCount   int

	// Special accounts that should always exist
	AdminEmails []string

	// Feature flags
	SeedMedia       bool   // Whether to generate media records
	SeedReviews     bool   // Whether to generate reviews
	SeedBookings    bool   // Whether to generate bookings
	SeedFinance     bool   // Whether to seed finance/wallets
	SeedPastData    bool   // Whether to include historical data
	SeedFutureData  bool   // Whether to include future bookings
	ListingOwnerIDs string // Optional CSV list of owner UUIDs to target for property/listing seeds

	// Ratios and distributions
	ShortletRatio float64 // Percentage of listings that are shortlets
	RentRatio     float64 // Percentage of listings that are rentals
	SaleRatio     float64 // Percentage of listings that are for sale

	// Database connection
	DatabaseURL string
}

// DefaultConfig returns the default seed configuration
func DefaultConfig() *SeedConfig {
	return &SeedConfig{
		UserCount:     getEnvInt("SEED_USERS", 30),
		BusinessCount: getEnvInt("SEED_BUSINESSES", 5),
		PropertyCount: getEnvInt("SEED_PROPERTIES", 50),
		ListingCount:  getEnvInt("SEED_LISTINGS", 100),
		BookingCount:  getEnvInt("SEED_BOOKINGS", 100),
		ReviewCount:   getEnvInt("SEED_REVIEWS", 60),

		AdminEmails: []string{
			"admin@hauslet.com",
			"support@hauslet.com",
		},

		SeedMedia:       getEnvBool("SEED_MEDIA", true),
		SeedReviews:     getEnvBool("SEED_REVIEWS", true),
		SeedBookings:    getEnvBool("SEED_BOOKINGS", true),
		SeedFinance:     getEnvBool("SEED_FINANCE", true),
		SeedPastData:    getEnvBool("SEED_PAST_DATA", true),
		SeedFutureData:  getEnvBool("SEED_FUTURE_DATA", true),
		ListingOwnerIDs: getEnv("SEED_LISTING_OWNER_IDS", ""),

		ShortletRatio: 0.40, // 40% shortlets
		RentRatio:     0.35, // 35% rentals
		SaleRatio:     0.25, // 25% sales

		DatabaseURL: getEnv("DATABASE_URL", ""),
	}
}

// TestConfig returns a minimal configuration for fast testing
func TestConfig() *SeedConfig {
	return &SeedConfig{
		UserCount:     5,
		BusinessCount: 1,
		PropertyCount: 10,
		ListingCount:  15,
		BookingCount:  20,
		ReviewCount:   10,

		AdminEmails: []string{"admin@hauslet.com"},

		SeedMedia:       true,
		SeedReviews:     true,
		SeedBookings:    true,
		SeedFinance:     true,
		SeedPastData:    false,
		SeedFutureData:  true,
		ListingOwnerIDs: getEnv("SEED_LISTING_OWNER_IDS", ""),

		ShortletRatio: 0.50,
		RentRatio:     0.30,
		SaleRatio:     0.20,

		DatabaseURL: getEnv("DATABASE_URL", ""),
	}
}

// Helper functions to read environment variables
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}
