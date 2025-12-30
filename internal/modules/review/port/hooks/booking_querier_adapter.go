package hooks

import (
	"context"
	"fmt"

	bookingRepo "hauslet/internal/modules/booking/repository"
	bookingSchema "hauslet/internal/modules/booking/repository/schema"
	propertyRepo "hauslet/internal/modules/property/repository"

	"github.com/google/uuid"
)

// BookingQuerierAdapter adapts BookingRepository to ReviewService's BookingQuerier interface
type BookingQuerierAdapter struct {
	bookingRepo  bookingRepo.BookingRepository
	propertyRepo propertyRepo.Repository
}

// NewBookingQuerierAdapter creates a new adapter for querying booking data
func NewBookingQuerierAdapter(
	bookingRepo bookingRepo.BookingRepository,
	propertyRepo propertyRepo.Repository,
) *BookingQuerierAdapter {
	return &BookingQuerierAdapter{
		bookingRepo:  bookingRepo,
		propertyRepo: propertyRepo,
	}
}

// GetBookingParties returns the guest and host IDs for a booking
func (a *BookingQuerierAdapter) GetBookingParties(ctx context.Context, bookingID uuid.UUID) (guestID, hostID uuid.UUID, err error) {
	booking, err := a.bookingRepo.GetBookingByID(ctx, bookingID)
	if err != nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("failed to get booking: %w", err)
	}
	if booking == nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("booking not found")
	}

	// Get listing to find host/owner ID
	listing, err := a.propertyRepo.GetListingByID(ctx, booking.ListingID, false)
	if err != nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("failed to get listing: %w", err)
	}
	if listing == nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("listing not found")
	}

	return booking.GuestID, listing.OwnerID, nil
}

// GetBookingListing returns the listing ID for a booking
func (a *BookingQuerierAdapter) GetBookingListing(ctx context.Context, bookingID uuid.UUID) (uuid.UUID, error) {
	booking, err := a.bookingRepo.GetBookingByID(ctx, bookingID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to get booking: %w", err)
	}
	if booking == nil {
		return uuid.Nil, fmt.Errorf("booking not found")
	}

	return booking.ListingID, nil
}

// IsBookingCompleted checks if a booking is in completed status
func (a *BookingQuerierAdapter) IsBookingCompleted(ctx context.Context, bookingID uuid.UUID) (bool, error) {
	booking, err := a.bookingRepo.GetBookingByID(ctx, bookingID)
	if err != nil {
		return false, fmt.Errorf("failed to get booking: %w", err)
	}
	if booking == nil {
		return false, fmt.Errorf("booking not found")
	}

	return booking.Status == bookingSchema.BookingStatusCompleted, nil
}
