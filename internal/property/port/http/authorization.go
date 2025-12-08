package http

import (
	"context"
	"hauslet/internal/property/domain"

	"github.com/google/uuid"
)

// verifyListingOwnership checks if the user owns the listing
func (h *HTTPHandler) verifyListingOwnership(ctx context.Context, listingID uuid.UUID, userID uuid.UUID) error {
	// Get the listing by ID
	listing, err := h.propertyService.GetListingByID(ctx, listingID, false)
	if err != nil {
		h.log.Logf("ERROR Failed to get listing %s: %v", listingID, err)
		return err
	}

	if listing == nil {
		return domain.ErrListingNotFound
	}

	// Check if the user owns the listing
	if listing.OwnerID != userID {
		h.log.Logf("WARN User %s attempted to access listing %s owned by %s", userID, listingID, listing.OwnerID)
		return domain.ErrForbidden
	}

	return nil
}
