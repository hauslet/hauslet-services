package http

import (
	"context"

	businessmiddleware "hauslet/internal/modules/business/middleware"
	"hauslet/internal/modules/property/domain"

	"github.com/google/uuid"
)

// verifyListingOwnership checks if the user owns the listing
func (h *HTTPHandler) verifyListingOwnership(ctx context.Context, listingID uuid.UUID, userID uuid.UUID) error {
	// Get the listing by ID
	listing, err := h.propertyService.GetListingByID(ctx, listingID, false)
	if err != nil {
		h.log.Error("Failed to get listing %s: %v", listingID, err)
		return err
	}

	if listing == nil {
		return domain.ErrListingNotFound
	}

	// Business-owned listing: allow members from business context (set by tenant slug middleware).
	if listing.OwnerType == domain.OwnerBusiness {
		if bc, ok := businessmiddleware.GetBusinessContext(ctx); ok && bc.BusinessID == listing.OwnerID && bc.Membership != nil {
			return nil
		}
	}

	// Check if the user owns the listing
	if listing.OwnerID != userID {
		h.log.Warn("User %s attempted to access listing %s owned by %s", userID, listingID, listing.OwnerID)
		return domain.ErrForbidden
	}

	return nil
}
