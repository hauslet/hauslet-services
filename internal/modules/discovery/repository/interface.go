package repository

import (
	"context"

	"hauslet/internal/modules/discovery/repository/schema"

	"github.com/google/uuid"
)

// DiscoveryRepository handles data persistence for the discovery module
type DiscoveryRepository interface {
	// Search History
	SaveSearchHistory(ctx context.Context, history *schema.SearchHistory) error
	GetRecentSearches(ctx context.Context, userID uuid.UUID, limit int) ([]*schema.SearchHistory, error)
	GetPopularSearches(ctx context.Context, limit int) ([]string, error)

	// User Preferences
	SaveUserPreferences(ctx context.Context, prefs *schema.UserPreferences) error
	GetUserPreferences(ctx context.Context, userID uuid.UUID) (*schema.UserPreferences, error)
	UpdateUserPreferences(ctx context.Context, userID uuid.UUID, updates map[string]interface{}) error

	// Analytics (future)
	// TrackListingClick(ctx context.Context, searchID, listingID uuid.UUID) error
	// GetClickThroughRate(ctx context.Context, listingID uuid.UUID) (float64, error)
}
