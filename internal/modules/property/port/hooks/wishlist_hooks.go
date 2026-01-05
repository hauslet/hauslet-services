package hooks

import (
	"context"
	"hauslet/internal/modules/property/domain"
	propertyService "hauslet/internal/modules/property/service"

	"github.com/google/uuid"
)

type WishlistHooksAdapter struct {
	svc propertyService.PropertyService
}

func NewWishlistHooksAdapter(svc propertyService.PropertyService) *WishlistHooksAdapter {
	return &WishlistHooksAdapter{svc: svc}
}

func (a *WishlistHooksAdapter) CanAddListingToWishlist(ctx context.Context, listingID uuid.UUID) (bool, error) {
	listing, err := a.svc.GetListingByID(ctx, listingID, false)
	if err != nil {
		return false, err
	}
	return listing != nil &&
			listing.Status == domain.StatusActive &&
			listing.Published,
		nil
}
