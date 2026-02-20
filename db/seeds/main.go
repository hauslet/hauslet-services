package main

import (
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	"hauslet/db/seeds/seeders"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	// Parse command-line flags
	configMode := flag.String("config", "default", "Configuration mode: default, test")
	modules := flag.String("modules", "all", "Comma-separated list of modules to seed (e.g., auth,property) or 'all'")
	clearFirst := flag.Bool("clear", false, "Clear existing data before seeding")
	flag.Parse()

	// Get seed configuration
	var seedConfig *seeders.SeedConfig
	switch *configMode {
	case "test":
		cfg := TestConfig()
		seedConfig = &seeders.SeedConfig{
			UserCount:       cfg.UserCount,
			BusinessCount:   cfg.BusinessCount,
			PropertyCount:   cfg.PropertyCount,
			ListingCount:    cfg.ListingCount,
			BookingCount:    cfg.BookingCount,
			ReviewCount:     cfg.ReviewCount,
			AdminEmails:     cfg.AdminEmails,
			SeedMedia:       cfg.SeedMedia,
			SeedReviews:     cfg.SeedReviews,
			SeedBookings:    cfg.SeedBookings,
			SeedFinance:     cfg.SeedFinance,
			SeedPastData:    cfg.SeedPastData,
			SeedFutureData:  cfg.SeedFutureData,
			ListingOwnerIDs: cfg.ListingOwnerIDs,
			ShortletRatio:   cfg.ShortletRatio,
			RentRatio:       cfg.RentRatio,
			SaleRatio:       cfg.SaleRatio,
			DatabaseURL:     cfg.DatabaseURL,
		}
	default:
		cfg := DefaultConfig()
		seedConfig = &seeders.SeedConfig{
			UserCount:       cfg.UserCount,
			BusinessCount:   cfg.BusinessCount,
			PropertyCount:   cfg.PropertyCount,
			ListingCount:    cfg.ListingCount,
			BookingCount:    cfg.BookingCount,
			ReviewCount:     cfg.ReviewCount,
			AdminEmails:     cfg.AdminEmails,
			SeedMedia:       cfg.SeedMedia,
			SeedReviews:     cfg.SeedReviews,
			SeedBookings:    cfg.SeedBookings,
			SeedFinance:     cfg.SeedFinance,
			SeedPastData:    cfg.SeedPastData,
			SeedFutureData:  cfg.SeedFutureData,
			ListingOwnerIDs: cfg.ListingOwnerIDs,
			ShortletRatio:   cfg.ShortletRatio,
			RentRatio:       cfg.RentRatio,
			SaleRatio:       cfg.SaleRatio,
			DatabaseURL:     cfg.DatabaseURL,
		}
	}

	if seedConfig.DatabaseURL == "" {
		log.Fatal("DATABASE_URL is required. Set it via environment variable.")
	}

	// Initialize database connection
	fmt.Println("🔌 Connecting to database...")
	db, err := gorm.Open(postgres.Open(seedConfig.DatabaseURL), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Failed to get database instance: %v", err)
	}
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)
	defer sqlDB.Close()

	fmt.Println("✅ Database connected successfully")

	// Optionally clear existing data
	if *clearFirst {
		fmt.Println("🗑️  Clearing existing data...")
		if err := clearData(db); err != nil {
			log.Fatalf("Failed to clear data: %v", err)
		}
		fmt.Println("✅ Data cleared")
	}

	// Parse modules to seed
	moduleList := parseModules(*modules)

	// Run seeders
	startTime := time.Now()
	fmt.Println("\n🌱 Starting database seeding...")
	fmt.Printf("Configuration: %s\n", *configMode)
	fmt.Printf("Modules: %v\n", moduleList)
	fmt.Println(strings.Repeat("-", 50))

	context := &seeders.SeedContext{
		DB:     db,
		Config: seedConfig,
	}

	// Run seeders in order
	if shouldSeed("auth", moduleList) {
		runSeeder("Auth", func() error {
			return seeders.SeedAuth(context)
		})
	}

	if shouldSeed("profile", moduleList) {
		runSeeder("Profile", func() error {
			return seeders.SeedProfile(context)
		})
	}

	if shouldSeed("business", moduleList) {
		runSeeder("Business", func() error {
			return seeders.SeedBusiness(context)
		})
	}

	if shouldSeed("property", moduleList) {
		runSeeder("Property", func() error {
			return seeders.SeedProperty(context)
		})
	}

	if shouldSeed("listing", moduleList) {
		runSeeder("Listing", func() error {
			return seeders.SeedListing(context)
		})
	}

	if shouldSeed("promotions", moduleList) {
		runSeeder("Promotions", func() error {
			return seeders.SeedPromotions(context)
		})
	}

	if seedConfig.SeedBookings && shouldSeed("booking", moduleList) {
		runSeeder("Booking", func() error {
			return seeders.SeedBooking(context)
		})
	}

	if seedConfig.SeedReviews && shouldSeed("review", moduleList) {
		runSeeder("Review", func() error {
			return seeders.SeedReview(context)
		})
	}

	if seedConfig.SeedFinance && shouldSeed("finance", moduleList) {
		runSeeder("Finance", func() error {
			return seeders.SeedFinance(context)
		})
	}

	elapsed := time.Since(startTime)
	fmt.Println(strings.Repeat("-", 50))
	fmt.Printf("\n✅ Seeding completed successfully in %v\n", elapsed)
	fmt.Println("\n📊 Summary:")
	printSummary(db)
}

// parseModules parses the comma-separated module list
func parseModules(input string) []string {
	if input == "all" || input == "" {
		return []string{"all"}
	}
	return strings.Split(input, ",")
}

// shouldSeed checks if a module should be seeded
func shouldSeed(module string, list []string) bool {
	if len(list) == 1 && list[0] == "all" {
		return true
	}
	for _, m := range list {
		if strings.TrimSpace(m) == module {
			return true
		}
	}
	return false
}

// runSeeder runs a seeder function with error handling
func runSeeder(name string, seeder func() error) {
	fmt.Printf("  📦 Seeding %s...", name)
	start := time.Now()

	if err := seeder(); err != nil {
		fmt.Printf(" ❌ Failed\n")
		log.Fatalf("Error seeding %s: %v", name, err)
	}

	elapsed := time.Since(start)
	fmt.Printf(" ✅ Done (%v)\n", elapsed)
}

// clearData removes all existing data (use with caution!)
func clearData(db *gorm.DB) error {
	// Order matters due to foreign keys - delete children first
	tables := []string{
		"listing_promotions",
		"listing_media",
		"reviews",
		"bookings",
		"calendar_events",
		"listings",
		"properties",
		"travel_companions",
		"profiles",
		"business_members",
		"business_invitations",
		"businesses",
		"transactions",
		"wallets",
		"search_histories",
		"user_preferences",
		"usage_trackings",
		"agent_subscriptions",
		"user_identities",
		"users",
	}

	for _, table := range tables {
		if err := db.Exec("DELETE FROM " + table).Error; err != nil {
			// Ignore errors for tables that might not exist
			fmt.Printf("Warning: Could not clear table %s: %v\n", table, err)
		}
	}

	return nil
}

// printSummary displays record counts for all tables
func printSummary(db *gorm.DB) {
	type countPair struct {
		name  string
		table string
	}

	counts := []countPair{
		{"Users", "users"},
		{"Profiles", "profiles"},
		{"Businesses", "businesses"},
		{"Properties", "properties"},
		{"Listings", "listings"},
		{"Listing Media", "listing_media"},
		{"Promotions", "listing_promotions"},
		{"Bookings", "bookings"},
		{"Reviews", "reviews"},
		{"Transactions", "transactions"},
	}

	for _, cp := range counts {
		var count int64
		db.Table(cp.table).Count(&count)
		fmt.Printf("  - %s: %d\n", cp.name, count)
	}
}
