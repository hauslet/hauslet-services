package hooks

import (
	"context"
	propertyservice "hauslet/internal/modules/property/service"

	"github.com/google/uuid"
)

// PromotionPropertyAdapter exposes minimal property data to the promotions module without coupling it to the full service.
type PromotionPropertyAdapter struct {
	svc propertyservice.PropertyService
}

// NewPromotionPropertyAdapter creates the adapter for promotions module.
func NewPromotionPropertyAdapter(svc propertyservice.PropertyService) *PromotionPropertyAdapter {
	return &PromotionPropertyAdapter{svc: svc}
}

// CountUserListings returns the total number of listings owned by a user.
// Can optionally filter by published status.
func (a *PromotionPropertyAdapter) CountUserListings(ctx context.Context, userID uuid.UUID, publishedOnly bool) (int, error) {
	// Create filter for user's listings
	filter := propertyservice.ListingFilter{
		OwnerID:        &userID,
		IncludeDeleted: false, // Never count deleted listings
	}

	// Filter by published status if requested
	if publishedOnly {
		published := true
		filter.Published = &published
	}

	// Get listing count (we only need the count, so limit to 1 for efficiency)
	pagination := propertyservice.Pagination{
		Limit:  1,
		Offset: 0,
	}

	_, totalCount, err := a.svc.ListListings(ctx, filter, pagination)
	if err != nil {
		return 0, err
	}

	return int(totalCount), nil
}
