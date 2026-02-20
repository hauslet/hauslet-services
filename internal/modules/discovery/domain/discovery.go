package domain

import (
	"time"

	propertydomain "hauslet/internal/modules/property/domain"

	"github.com/google/uuid"
)

// RankedListing represents a listing with comprehensive ranking information
type RankedListing struct {
	Listing        propertydomain.Listing // The actual listing
	Score          RankingScore           // Detailed scoring breakdown
	Ranking        int                    // Position in results (1-indexed)
	PromotionBoost *PromotionBoostInfo    // Promotion details if promoted
}

// PromotionBoostInfo provides details about the promotion affecting this listing
type PromotionBoostInfo struct {
	PromotionID     uuid.UUID // ID of the promotion
	PromotionType   string    // "featured" or "premium"
	BoostMultiplier float64   // Boost multiplier (10.0 for featured, 3.0 for premium)
	ExpiresAt       time.Time // When the promotion expires
}

// HomeFeedSection represents a section in the home feed
type HomeFeedSection struct {
	SectionType FeedSectionType // Type of section
	Title       string          // Display title for the section
	Listings    []RankedListing // Listings in this section
	TotalCount  int             // Total count of listings in this section
	SearchData  map[string]any  // Discover payload to replicate section listing data
}

// SearchHistory tracks user search queries for analytics and personalization
type SearchHistory struct {
	ID              uuid.UUID              // Search history ID
	UserID          uuid.UUID              // User who performed the search
	Query           string                 // Search query string
	Filters         map[string]interface{} // Applied filters (JSONB)
	ResultCount     int                    // Number of results returned
	ClickedListings []uuid.UUID            // Listing IDs clicked from results
	CreatedAt       time.Time              // When the search was performed
}

// UserPreferences stores user preferences for personalization (future)
type UserPreferences struct {
	ID                 uuid.UUID     // Preferences ID
	UserID             uuid.UUID     // User ID
	PreferredLocations []string      // Preferred cities/areas
	PreferredTypes     []string      // Preferred property types
	PriceRange         *PriceRange   // Preferred price range
	BedroomRange       *IntRange     // Preferred bedroom count range
	SavedFilters       []SavedFilter // Saved search filters
	UpdatedAt          time.Time     // Last update time
}

// PriceRange represents a price range preference
type PriceRange struct {
	Min      *float64 // Minimum price (nil = no minimum)
	Max      *float64 // Maximum price (nil = no maximum)
	Currency string   // Currency code (e.g., "NGN", "USD")
}

// IntRange represents an integer range (e.g., bedrooms)
type IntRange struct {
	Min *int // Minimum value (nil = no minimum)
	Max *int // Maximum value (nil = no maximum)
}

// SavedFilter represents a saved search filter
type SavedFilter struct {
	Name      string                 // User-defined filter name
	Filters   map[string]interface{} // Filter parameters (JSONB)
	CreatedAt time.Time              // When the filter was saved
}

// SearchResult contains search results with metadata
type SearchResult struct {
	Listings       []RankedListing // Ranked listings
	TotalCount     int             // Total matching results (before limit)
	SearchID       uuid.UUID       // Unique search ID for tracking
	ProcessingTime int64           // Processing time in milliseconds
}

// IsPromoted checks if a listing is currently promoted
func (r RankedListing) IsPromoted() bool {
	return r.PromotionBoost != nil
}

// IsFeatured checks if a listing is featured (10x boost)
func (r RankedListing) IsFeatured() bool {
	return r.IsPromoted() && r.PromotionBoost.PromotionType == "featured"
}

// IsPremium checks if a listing is premium (3x boost)
func (r RankedListing) IsPremium() bool {
	return r.IsPromoted() && r.PromotionBoost.PromotionType == "premium"
}
