package service

import (
	"context"
	"hauslet/internal/modules/interactions/domain"

	"github.com/google/uuid"
)

// TrackerService handles tracking user interactions (writes to Redis buffer)
type TrackerService interface {
	// Track records a new interaction (fire-and-forget to Redis)
	Track(ctx context.Context, input TrackInput) error

	// TrackBatch records multiple interactions
	TrackBatch(ctx context.Context, inputs []TrackInput) error
}

// ReaderService handles reading interaction analytics (reads from DB aggregates)
type ReaderService interface {
	// GetListingAnalytics retrieves analytics for a listing
	GetListingAnalytics(ctx context.Context, listingID uuid.UUID, days int) (*domain.ListingAnalytics, error)

	// GetUserHistory retrieves a user's recent interaction history
	GetUserHistory(ctx context.Context, userID uuid.UUID, limit int) (*domain.UserActivityHistory, error)
}

// TrackInput represents the input for tracking an interaction
type TrackInput struct {
	UserID     *uuid.UUID
	SessionID  string
	Type       domain.InteractionType
	EntityType domain.EntityType
	EntityID   *uuid.UUID
	Context    map[string]any

	// Client-provided metadata
	DeviceType domain.DeviceType
	Platform   domain.Platform
	IPAddress  string // Will be hashed
	UserAgent  string
	Referrer   string
}
