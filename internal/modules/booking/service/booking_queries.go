package service

import (
	"context"
	"fmt"
	"hauslet/internal/modules/auth/authorization"
	"hauslet/internal/modules/booking/domain"
	"hauslet/internal/modules/booking/repository/schema"

	"github.com/google/uuid"
)

func (s *BookingServiceImpl) GetBooking(ctx context.Context, bookingID uuid.UUID, requestorID uuid.UUID) (*domain.Booking, error) {
	schemaBooking, err := s.repo.GetBookingByID(ctx, bookingID)
	if err != nil {
		return nil, err
	}

	booking := domain.MapBookingFromSchema(schemaBooking)
	if err := s.authorizeBookingAccess(ctx, booking, requestorID); err != nil {
		return nil, err
	}

	return booking, nil
}

func (s *BookingServiceImpl) GetBookingsByIDs(ctx context.Context, bookingIDs []uuid.UUID) ([]*domain.Booking, error) {
	schemaBookings, err := s.repo.GetBookingsByIDs(ctx, bookingIDs)
	if err != nil {
		return nil, err
	}
	bookings := make([]*domain.Booking, len(schemaBookings))
	for i, sb := range schemaBookings {
		bookings[i] = domain.MapBookingFromSchema(sb)
	}
	return bookings, nil
}

func (s *BookingServiceImpl) GetBookingByReference(ctx context.Context, reference string, requestorID uuid.UUID) (*domain.Booking, error) {
	schemaBooking, err := s.repo.GetBookingByReference(ctx, reference)
	if err != nil {
		return nil, err
	}

	booking := domain.MapBookingFromSchema(schemaBooking)
	if err := s.authorizeBookingAccess(ctx, booking, requestorID); err != nil {
		return nil, err
	}

	return booking, nil
}

func (s *BookingServiceImpl) authorizeBookingAccess(ctx context.Context, booking *domain.Booking, requestorID uuid.UUID) error {
	if booking.GuestID == requestorID {
		return nil
	}

	ownerID, err := s.listingHooks.GetListingOwner(ctx, booking.ListingID)
	if err != nil {
		return err
	}
	if ownerID != requestorID {
		return domain.ErrUnauthorized
	}

	return nil
}

func (s *BookingServiceImpl) ListBookingsForGuest(ctx context.Context, guestID uuid.UUID, limit, offset int) ([]*domain.Booking, error) {
	schemaBookings, err := s.repo.ListBookingsForGuest(ctx, guestID, limit, offset)
	if err != nil {
		return nil, err
	}

	bookings := make([]*domain.Booking, 0, len(schemaBookings))
	for _, sb := range schemaBookings {
		bookings = append(bookings, domain.MapBookingFromSchema(sb))
	}

	return bookings, nil
}

func (s *BookingServiceImpl) ListBookingsForListing(ctx context.Context, listingID uuid.UUID, requestorID uuid.UUID, status *domain.BookingStatus, limit, offset int) ([]*domain.Booking, error) {
	// Verify requestor is the listing owner
	ownerID, err := s.listingHooks.GetListingOwner(ctx, listingID)
	if err != nil {
		return nil, err
	}
	if ownerID != requestorID {
		return nil, domain.ErrUnauthorized
	}

	schemaBookings, err := s.repo.ListBookingsForListing(ctx, listingID, limit, offset)
	if err != nil {
		return nil, err
	}

	bookings := make([]*domain.Booking, 0, len(schemaBookings))
	for _, sb := range schemaBookings {
		booking := domain.MapBookingFromSchema(sb)
		// Filter by status if provided
		if status != nil && booking.Status != *status {
			continue
		}
		bookings = append(bookings, booking)
	}

	return bookings, nil
}

func (s *BookingServiceImpl) ListBookingsForHost(ctx context.Context, hostID uuid.UUID, status *domain.BookingStatus, limit, offset int) ([]*domain.Booking, error) {
	if s.supplyGate == nil {
		return nil, domain.ErrUnauthorized // or internal error, but gate is required
	}

	// Use Supply Gate to verify host eligibility
	if err := s.supplyGate.Authorize(ctx, hostID, authorization.SupplyActionManageBookings, nil); err != nil {
		return nil, fmt.Errorf("unauthorized to manage bookings: %w", err)
	}

	// Map domain status to schema status if provided
	var schemaStatus *schema.BookingStatus
	if status != nil {
		st := schema.BookingStatus(*status)
		schemaStatus = &st
	}

	schemaBookings, err := s.repo.ListBookingsForHost(ctx, hostID, schemaStatus, limit, offset)
	if err != nil {
		return nil, err
	}

	bookings := make([]*domain.Booking, 0, len(schemaBookings))
	for _, sb := range schemaBookings {
		bookings = append(bookings, domain.MapBookingFromSchema(sb))
	}

	return bookings, nil
}
