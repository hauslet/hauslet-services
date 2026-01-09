package service

import (
	"context"
	"fmt"
	"time"

	"hauslet/internal/modules/booking/domain"

	"github.com/google/uuid"
)

const autoCheckInOutDelay = 60 * time.Minute
const autoCheckInOutLimit = 100

// CheckInBooking records an actual check-in time (host-only).
func (s *BookingServiceImpl) CheckInBooking(ctx context.Context, bookingID uuid.UUID, actorID uuid.UUID) (*domain.Booking, error) {
	booking, ownerID, err := s.getBookingWithOwner(ctx, bookingID, actorID)
	if err != nil {
		return nil, err
	}
	if actorID != ownerID {
		return nil, domain.ErrUnauthorized
	}
	if booking.Status != domain.BookingStatusConfirmed && booking.Status != domain.BookingStatusActive {
		return nil, domain.ErrCannotCheckIn
	}

	scheduledCheckIn := booking.ScheduledCheckIn()
	if scheduledCheckIn == nil {
		return nil, domain.ErrInvalidDateRange
	}

	if booking.CheckIn != nil {
		return booking, nil
	}

	checkInAt := time.Now()
	if checkInAt.Before(*scheduledCheckIn) {
		checkInAt = *scheduledCheckIn
	}

	booking.CheckIn = &checkInAt
	if booking.Status != domain.BookingStatusActive {
		booking.MarkActive(checkInAt)
	}
	booking.UpdatedAt = time.Now()

	if err := s.repo.UpdateBooking(ctx, domain.MapBookingFromDomain(booking)); err != nil {
		return nil, fmt.Errorf("failed to update booking check-in: %w", err)
	}

	return booking, nil
}

// CheckOutBooking records an actual check-out time (host-only).
func (s *BookingServiceImpl) CheckOutBooking(ctx context.Context, bookingID uuid.UUID, actorID uuid.UUID) (*domain.Booking, error) {
	booking, ownerID, err := s.getBookingWithOwner(ctx, bookingID, actorID)
	if err != nil {
		return nil, err
	}
	if actorID != ownerID {
		return nil, domain.ErrUnauthorized
	}

	if booking.Status == domain.BookingStatusConfirmed {
		scheduledCheckIn := booking.ScheduledCheckIn()
		if scheduledCheckIn == nil || time.Now().Before(*scheduledCheckIn) {
			return nil, domain.ErrCannotCheckOut
		}
		if booking.CheckIn == nil {
			checkInAt := *scheduledCheckIn
			booking.CheckIn = &checkInAt
			booking.MarkActive(checkInAt)
		}
	}

	if booking.Status != domain.BookingStatusActive {
		return nil, domain.ErrCannotCheckOut
	}

	if booking.CheckOut != nil {
		return booking, nil
	}

	scheduledCheckOut := booking.ScheduledCheckOut()
	if scheduledCheckOut == nil {
		return nil, domain.ErrInvalidDateRange
	}

	checkOutAt := time.Now()
	if checkOutAt.Before(*scheduledCheckOut) {
		checkOutAt = *scheduledCheckOut
	}
	if booking.CheckIn != nil && checkOutAt.Before(*booking.CheckIn) {
		checkOutAt = *booking.CheckIn
	}

	booking.CheckOut = &checkOutAt
	booking.UpdatedAt = time.Now()

	if err := s.repo.UpdateBooking(ctx, domain.MapBookingFromDomain(booking)); err != nil {
		return nil, fmt.Errorf("failed to update booking check-out: %w", err)
	}

	return booking, nil
}

// AutoPopulateCheckInOut backfills actual check-in/out timestamps after the scheduled time.
func (s *BookingServiceImpl) AutoPopulateCheckInOut(ctx context.Context) (int, int, error) {
	cutoff := time.Now().Add(-autoCheckInOutDelay)

	checkInBookings, err := s.repo.FindBookingsPendingCheckIn(ctx, cutoff, autoCheckInOutLimit)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to find pending check-ins: %w", err)
	}

	checkInUpdated := 0
	for _, schemaBooking := range checkInBookings {
		booking := domain.MapBookingFromSchema(schemaBooking)
		scheduledCheckIn := booking.ScheduledCheckIn()
		if scheduledCheckIn == nil {
			continue
		}

		checkInAt := *scheduledCheckIn
		booking.CheckIn = &checkInAt
		booking.MarkActive(checkInAt)
		booking.UpdatedAt = time.Now()

		if err := s.repo.UpdateBooking(ctx, domain.MapBookingFromDomain(booking)); err != nil {
			if s.log != nil {
				s.log.Warn("failed to auto-populate check-in", "booking_id", booking.ID, "error", err)
			}
			continue
		}
		checkInUpdated++
	}

	checkOutBookings, err := s.repo.FindBookingsPendingCheckOut(ctx, cutoff, autoCheckInOutLimit)
	if err != nil {
		return checkInUpdated, 0, fmt.Errorf("failed to find pending check-outs: %w", err)
	}

	checkOutUpdated := 0
	for _, schemaBooking := range checkOutBookings {
		booking := domain.MapBookingFromSchema(schemaBooking)
		scheduledCheckOut := booking.ScheduledCheckOut()
		if scheduledCheckOut == nil {
			continue
		}

		checkOutAt := *scheduledCheckOut
		if booking.CheckIn != nil && checkOutAt.Before(*booking.CheckIn) {
			checkOutAt = *booking.CheckIn
		}

		booking.CheckOut = &checkOutAt
		booking.UpdatedAt = time.Now()

		if err := s.repo.UpdateBooking(ctx, domain.MapBookingFromDomain(booking)); err != nil {
			if s.log != nil {
				s.log.Warn("failed to auto-populate check-out", "booking_id", booking.ID, "error", err)
			}
			continue
		}
		checkOutUpdated++
	}
	s.log.Info("auto-populate check-in/out completed",
		"checkins_updated", checkInUpdated,
		"checkouts_updated", checkOutUpdated)

	return checkInUpdated, checkOutUpdated, nil
}
