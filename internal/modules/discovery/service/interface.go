package service

import (
	"context"
	"time"

	"hauslet/internal/modules/discovery/domain"
	propertydomain "hauslet/internal/modules/property/domain"

	"github.com/google/uuid"
)

// DiscoveryService handles all search and discovery operations
type DiscoveryService interface {
	// SearchListings performs semantic search with promotion-aware ranking
	SearchListings(ctx context.Context, filter SearchFilter, options SearchOptions) (*domain.SearchResult, error)

	// GetHomeFeed returns a curated home feed with multiple sections
	GetHomeFeed(ctx context.Context, userID *uuid.UUID, options FeedOptions) ([]domain.HomeFeedSection, error)

	// GetFeaturedListings returns currently featured (promoted) listings
	GetFeaturedListings(ctx context.Context, limit int) ([]domain.RankedListing, error)

	// FindSimilarListings finds listings similar to the given listing
	FindSimilarListings(ctx context.Context, listingID uuid.UUID, limit int) ([]domain.RankedListing, error)

	// Future: Recommendations, search history, user preferences
	// GetRecommendations(ctx context.Context, userID uuid.UUID, limit int) ([]domain.RankedListing, error)
	// SaveSearch(ctx context.Context, userID uuid.UUID, query string, filters map[string]interface{}, resultCount int) error
	// GetRecentSearches(ctx context.Context, userID uuid.UUID, limit int) ([]*domain.SearchHistory, error)
}

// SearchFilter contains all search parameters
type SearchFilter struct {
	Query              *string         // Search query for semantic search
	Location           *LocationFilter // Location-based filtering
	PriceRange         *PriceRangeFilter
	PropertyTypes      []string
	Bedrooms           *IntRangeFilter
	Bathrooms          *IntRangeFilter
	ListingTypes       []string
	City               *string
	State              *string
	Country            *string
	Amenities          []string
	GuestCount         *int
	CheckIn            *time.Time
	CheckOut           *time.Time
	Furnishing         []string
	AccommodationTypes []string
	// Additional filters can be added as needed
}

// LocationFilter for geospatial searches
type LocationFilter struct {
	Latitude  float64
	Longitude float64
	RadiusKm  float64 // Search radius in kilometers
}

// PriceRangeFilter for price-based filtering
type PriceRangeFilter struct {
	Min      *int64 // Minimum price in cents
	Max      *int64 // Maximum price in cents
	Currency string // Currency code (e.g., "NGN")
}

// IntRangeFilter for integer range filtering (bedrooms, bathrooms, etc.)
type IntRangeFilter struct {
	Min *int
	Max *int
}

// SearchOptions controls search behavior and ranking
type SearchOptions struct {
	Limit           int                   // Maximum number of results
	IncludePromoted bool                  // Whether to include promoted listings
	UserID          *uuid.UUID            // For personalization (future)
	RankingConfig   *domain.RankingConfig // Custom ranking configuration (optional)
}

// FeedOptions controls home feed composition
type FeedOptions struct {
	Location          *LocationFilter          // Location for "near you" section
	Limit             int                      // Default limit per section
	SectionsToInclude []domain.FeedSectionType // Specific sections to include (empty = all)
}

// ===== HOOK INTERFACES (What Discovery needs from other modules) =====

// PropertyDiscoveryHooks defines the minimal interface Discovery needs from Property module
type PropertyDiscoveryHooks interface {
	// GetListingByID retrieves a single listing by ID
	GetListingByID(ctx context.Context, id uuid.UUID) (*propertydomain.Listing, error)

	// GetListingsByIDs batch fetches listings by IDs
	GetListingsByIDs(ctx context.Context, ids []uuid.UUID) ([]propertydomain.Listing, error)

	// SearchListingsWithEmbedding performs semantic search using embeddings
	SearchListingsWithEmbedding(ctx context.Context, filter SearchFilter, limit int, excludedIDs []uuid.UUID) ([]propertydomain.ScoredListing, error)

	// GetRecentListings fetches recently published listings
	GetRecentListings(ctx context.Context, limit int) ([]propertydomain.Listing, error)

	// FindSimilarListings finds listings similar to the given listing
	FindSimilarListings(ctx context.Context, listingID uuid.UUID, limit int) ([]propertydomain.ScoredListing, error)
}

// PromotionDiscoveryHooks defines the minimal interface Discovery needs from Promotions module
type PromotionDiscoveryHooks interface {
	// GetActivePromotionForListing gets the active promotion for a single listing
	GetActivePromotionForListing(ctx context.Context, listingID uuid.UUID) (*PromotionInfo, error)

	// GetActivePromotionForListings batch fetches active promotions for multiple listings
	GetActivePromotionForListings(ctx context.Context, listingIDs []uuid.UUID) (map[uuid.UUID]*PromotionInfo, error)

	// GetFeaturedListings gets listing IDs for currently featured promotions
	GetFeaturedListings(ctx context.Context, limit int) ([]uuid.UUID, error)

	// GetPremiumListings gets listing IDs for currently premium promotions
	GetPremiumListings(ctx context.Context, limit int) ([]uuid.UUID, error)
}

// CalendarDiscoveryHooks defines what Discovery needs from Calendar
type CalendarDiscoveryHooks interface {
	// GetUnavailableListingIDs returns IDs of listings that are busy/booked in the given range
	GetUnavailableListingIDs(ctx context.Context, startTime, endTime time.Time) ([]uuid.UUID, error)
}

// PromotionInfo contains promotion details needed for ranking
type PromotionInfo struct {
	PromotionID     uuid.UUID
	PromotionType   string  // "featured" or "premium"
	BoostMultiplier float64 // Boost multiplier (10.0 for featured, 3.0 for premium)
	ExpiresAt       time.Time
}
