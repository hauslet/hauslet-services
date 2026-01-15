package service

import (
	"context"

	"github.com/google/uuid"
)

// PropertyHooks defines the interface for interacting with the property module
// This allows the leads module to validate listings and get ownership information
type PropertyHooks interface {
	// ListingExists checks if a listing exists and is active
	ListingExists(ctx context.Context, listingID uuid.UUID) (bool, error)

	// GetListingOwner retrieves ownership information for a listing
	GetListingOwner(ctx context.Context, listingID uuid.UUID) (*ListingOwnerInfo, error)
}

// ListingOwnerInfo contains ownership information for a listing
type ListingOwnerInfo struct {
	OwnerID    uuid.UUID
	OwnerType  string     // "user" or "business"
	BusinessID *uuid.UUID // Populated if owner_type is "business"
}

// BusinessHooks defines the interface for interacting with the business module
// This allows the leads module to check business membership and permissions
type BusinessHooks interface {
	// HasPermission checks if a user has a specific permission in a business
	HasPermission(ctx context.Context, userID, businessID uuid.UUID, permission string) (bool, error)

	// IsMember checks if a user is a member of a business
	IsMember(ctx context.Context, userID, businessID uuid.UUID) (bool, error)

	// GetBusinessMembers retrieves all members of a business (for auto-assignment)
	GetBusinessMembers(ctx context.Context, businessID uuid.UUID) ([]uuid.UUID, error)
}

// AnalyticsHooks defines the interface for tracking lead-related analytics
// This is a no-op implementation for Phase 1, but ready for future integration
type AnalyticsHooks interface {
	// TrackLeadCreated tracks when a lead is created
	TrackLeadCreated(ctx context.Context, leadID uuid.UUID, listingID uuid.UUID, source string) error

	// TrackLeadConverted tracks when a lead converts
	TrackLeadConverted(ctx context.Context, leadID uuid.UUID, conversionValue *float64) error

	// TrackLeadResponseTime tracks response time for a lead
	TrackLeadResponseTime(ctx context.Context, leadID uuid.UUID, responseTime int64) error
}

// NullAnalyticsHooks is a no-op implementation of AnalyticsHooks for Phase 1
type NullAnalyticsHooks struct{}

// TrackLeadCreated is a no-op
func (n *NullAnalyticsHooks) TrackLeadCreated(ctx context.Context, leadID uuid.UUID, listingID uuid.UUID, source string) error {
	return nil
}

// TrackLeadConverted is a no-op
func (n *NullAnalyticsHooks) TrackLeadConverted(ctx context.Context, leadID uuid.UUID, conversionValue *float64) error {
	return nil
}

// TrackLeadResponseTime is a no-op
func (n *NullAnalyticsHooks) TrackLeadResponseTime(ctx context.Context, leadID uuid.UUID, responseTime int64) error {
	return nil
}
