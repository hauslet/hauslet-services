package hooks

import (
	"context"
	"fmt"
	"hauslet/internal/modules/booking/repository"
	"hauslet/internal/modules/booking/service"
	"hauslet/internal/modules/finance/domain"

	"github.com/google/uuid"
)

// BookingPartyQuerier exposes minimal booking data to finance for authorization checks
type BookingPartyQuerier interface {
	// GetBookingParty determines if a user is the guest or host of a booking
	// Returns DisputePartyGuest, DisputePartyHost, or error if user not involved
	GetBookingParty(ctx context.Context, bookingID, userID uuid.UUID) (domain.DisputeParty, error)

	// GetBookingPaymentID retrieves the last payment ID for a booking
	GetBookingPaymentID(ctx context.Context, bookingID uuid.UUID) (uuid.UUID, error)
}

// FinanceBookingAdapter exposes minimal booking data to the finance module without full coupling
type FinanceBookingAdapter struct {
	bookingRepo  repository.BookingRepository
	listingHooks service.ListingHooks
}

// NewFinanceBookingAdapter creates the adapter for finance authorization
func NewFinanceBookingAdapter(
	bookingRepo repository.BookingRepository,
	listingHooks service.ListingHooks,
) *FinanceBookingAdapter {
	return &FinanceBookingAdapter{
		bookingRepo:  bookingRepo,
		listingHooks: listingHooks,
	}
}

// GetBookingParty determines if user is guest or host of a booking
func (a *FinanceBookingAdapter) GetBookingParty(
	ctx context.Context,
	bookingID, userID uuid.UUID,
) (domain.DisputeParty, error) {
	// Get booking directly from repository
	booking, err := a.bookingRepo.GetBookingByID(ctx, bookingID)
	if err != nil {
		return "", fmt.Errorf("failed to get booking: %w", err)
	}
	if booking == nil {
		return "", fmt.Errorf("booking not found")
	}

	// Check if user is the guest
	if booking.GuestID == userID {
		return domain.DisputePartyGuest, nil
	}

	// Check if user is the host (via listing ownership)
	// This uses the same pattern as booking/service/completion.go:109
	hostID, err := a.listingHooks.GetListingOwner(ctx, booking.ListingID)
	if err != nil {
		return "", fmt.Errorf("failed to get listing owner: %w", err)
	}

	if hostID == userID {
		return domain.DisputePartyHost, nil
	}

	// User is neither guest nor host
	return "", fmt.Errorf("user not involved in this booking")
}

// GetBookingPaymentID retrieves the last payment ID for a booking
func (a *FinanceBookingAdapter) GetBookingPaymentID(
	ctx context.Context,
	bookingID uuid.UUID,
) (uuid.UUID, error) {
	// Get booking directly from repository
	booking, err := a.bookingRepo.GetBookingByID(ctx, bookingID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to get booking: %w", err)
	}
	if booking == nil {
		return uuid.Nil, fmt.Errorf("booking not found")
	}

	// Return last payment ID (or error if not set)
	if booking.LastPaymentID == nil {
		return uuid.Nil, fmt.Errorf("booking has no payment associated")
	}

	return *booking.LastPaymentID, nil
}
