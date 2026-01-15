package hooks

import (
	"context"
	"errors"
	"fmt"
	bookingdomain "hauslet/internal/modules/booking/domain"
	bookingrepository "hauslet/internal/modules/booking/repository"
	messagingdomain "hauslet/internal/modules/messaging/domain"
	propertydomain "hauslet/internal/modules/property/domain"
	propertyrepository "hauslet/internal/modules/property/repository"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// BookingHooks defines the contract for interacting with the Booking module
type BookingHooks interface {
	// GetBookingParticipants returns the guest and host IDs
	GetBookingParticipants(ctx context.Context, bookingID uuid.UUID) (guestID, hostID uuid.UUID, err error)

	// GetBookingState returns status to help AI decide if it can answer questions
	GetBookingState(ctx context.Context, bookingID uuid.UUID) (status string, err error)
}

// bookingHooksAdapter provides booking integration for messaging.
type bookingHooksAdapter struct {
	bookingRepo  bookingrepository.BookingRepository
	propertyRepo propertyrepository.Repository
}

// NewBookingHooksAdapter wires booking and property repositories into the messaging hooks.
func NewBookingHooksAdapter(
	bookingRepo bookingrepository.BookingRepository,
	propertyRepo propertyrepository.Repository,
) messagingdomain.BookingHooks {
	return &bookingHooksAdapter{
		bookingRepo:  bookingRepo,
		propertyRepo: propertyRepo,
	}
}

var _ messagingdomain.BookingHooks = (*bookingHooksAdapter)(nil)

func (a *bookingHooksAdapter) GetBookingParticipants(ctx context.Context, bookingID uuid.UUID) (guestID, hostID uuid.UUID, err error) {
	booking, err := a.bookingRepo.GetBookingByID(ctx, bookingID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return uuid.Nil, uuid.Nil, fmt.Errorf("booking not found: %w", bookingdomain.ErrBookingNotFound)
		}
		return uuid.Nil, uuid.Nil, fmt.Errorf("failed to load booking %s: %w", bookingID, err)
	}
	if booking == nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("booking not found: %w", bookingdomain.ErrBookingNotFound)
	}

	listing, err := a.propertyRepo.GetListingByID(ctx, booking.ListingID, false)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return uuid.Nil, uuid.Nil, fmt.Errorf("listing %s not found: %w", booking.ListingID, propertydomain.ErrListingNotFound)
		}
		return uuid.Nil, uuid.Nil, fmt.Errorf("failed to resolve listing %s for booking %s: %w", booking.ListingID, bookingID, err)
	}
	if listing == nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("listing %s not found for booking %s", booking.ListingID, bookingID)
	}
	if listing.OwnerID == uuid.Nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("listing %s missing owner", listing.ID)
	}

	return booking.GuestID, listing.OwnerID, nil
}

func (a *bookingHooksAdapter) GetBookingState(ctx context.Context, bookingID uuid.UUID) (string, error) {
	booking, err := a.bookingRepo.GetBookingByID(ctx, bookingID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", fmt.Errorf("booking not found: %w", bookingdomain.ErrBookingNotFound)
		}
		return "", fmt.Errorf("failed to load booking %s: %w", bookingID, err)
	}
	if booking == nil {
		return "", fmt.Errorf("booking not found: %w", bookingdomain.ErrBookingNotFound)
	}

	return string(booking.Status), nil
}
