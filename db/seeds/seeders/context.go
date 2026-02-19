package seeders

import (
	"gorm.io/gorm"
)

// SeedContext holds the database connection and configuration for seeders
type SeedContext struct {
	DB     *gorm.DB
	Config *SeedConfig
}

// SeedConfig holds configuration for seeding
type SeedConfig struct {
	UserCount       int
	BusinessCount   int
	PropertyCount   int
	ListingCount    int
	BookingCount    int
	ReviewCount     int
	AdminEmails     []string
	SeedMedia       bool
	SeedReviews     bool
	SeedBookings    bool
	SeedFinance     bool
	SeedPastData    bool
	SeedFutureData  bool
	ListingOwnerIDs string
	ShortletRatio   float64
	RentRatio       float64
	SaleRatio       float64
	DatabaseURL     string
}
