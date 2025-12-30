package hooks

import (
	"context"
	"fmt"
	"time"

	bookingRepo "hauslet/internal/modules/booking/repository"

	"github.com/google/uuid"
)

// ReviewHooksAdapter handles review-related callbacks for the booking module
// Implements the BookingHooks interface from the review module
type ReviewHooksAdapter struct {
	bookingRepo bookingRepo.BookingRepository
}

// NewReviewHooksAdapter creates a new adapter for handling review events
func NewReviewHooksAdapter(repo bookingRepo.BookingRepository) *ReviewHooksAdapter {
	return &ReviewHooksAdapter{
		bookingRepo: repo,
	}
}

// OnReviewPublished updates booking review timestamps when a review is published
// This is called by the review module after a review is published
func (a *ReviewHooksAdapter) OnReviewPublished(
	ctx context.Context,
	bookingID, reviewerID uuid.UUID,
	reviewerType string,
) error {
	// Get the booking
	booking, err := a.bookingRepo.GetBookingByID(ctx, bookingID)
	if err != nil {
		return fmt.Errorf("failed to get booking %s: %w", bookingID, err)
	}

	if booking == nil {
		return fmt.Errorf("booking %s not found", bookingID)
	}

	// Update the appropriate timestamp
	now := time.Now()

	switch reviewerType {
	case "guest":
		// Guest reviewed the listing/host
		if booking.GuestReviewedAt == nil {
			booking.GuestReviewedAt = &now
		}
	case "host":
		// Host reviewed the guest
		if booking.HostReviewedAt == nil {
			booking.HostReviewedAt = &now
		}
	default:
		// Unknown reviewer type, skip update but don't error
		return nil
	}

	// Save the updated booking
	if err := a.bookingRepo.UpdateBooking(ctx, booking); err != nil {
		return fmt.Errorf("failed to update booking %s review timestamp: %w", bookingID, err)
	}

	return nil
}
