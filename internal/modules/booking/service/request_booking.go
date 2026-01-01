package service

import (
	"context"
	"fmt"
	"hauslet/internal/modules/booking/domain"
	calendardomain "hauslet/internal/modules/calendar/domain"
	"time"

	"github.com/google/uuid"
)

// RequestBooking creates a manual-approval booking request.
// This is used for listings that require host approval before payment.
// For instant/auto-accept bookings, use ReserveBooking instead.
func (s *BookingServiceImpl) RequestBooking(
	ctx context.Context,
	listingID uuid.UUID,
	guestID uuid.UUID,
	checkIn, checkOut time.Time,
	guestCount int,
	specialRequests *string,
) (*domain.Booking, error) {
	if s.log != nil {
		s.log.Info(" creating booking request", "listing_id", listingID, "guest_id", guestID)
	}

	// Get listing constraints
	constraints, err := s.listingHooks.GetListingConstraints(ctx, listingID)
	if err != nil {
		return nil, err
	}

	// Get calendar config to check instant booking
	var calendarConfig *calendardomain.CalendarConfig
	if cfg, cfgErr := s.calendar.GetCalendarConfig(ctx, listingID); cfgErr == nil {
		calendarConfig = cfg
	}

	autoAcceptBookings := constraints.AutoAcceptBookings
	if calendarConfig != nil {
		autoAcceptBookings = calendarConfig.InstantBooking
	}

	// This mutation is ONLY for manual-approval bookings
	if autoAcceptBookings {
		return nil, fmt.Errorf("listing supports instant booking; use reserveBooking mutation instead")
	}

	// Check time until scheduled check-in
	scheduledCheckIn, _ := s.buildScheduledTimes(checkIn, checkOut, constraints)
	hoursUntilCheckIn := time.Until(scheduledCheckIn).Hours()
	if hoursUntilCheckIn < 6 {
		return nil, domain.ErrTooCloseToCheckIn
	}

	// Use the existing CreateBooking logic
	// This ensures we don't duplicate the booking creation code
	return s.CreateBooking(ctx, listingID, guestID, checkIn, checkOut, guestCount, specialRequests)
}

// PayForBooking processes payment for an existing booking.
// Use this after a booking has been approved (for manual-approval bookings)
// or for retrying failed payments.
func (s *BookingServiceImpl) PayForBooking(
	ctx context.Context,
	bookingID uuid.UUID,
	actorID uuid.UUID,
	paymentMethodID *uuid.UUID,
) (*domain.Booking, *PaymentResult, error) {
	if s.log != nil {
		s.log.Info(" processing payment for booking", "booking_id", bookingID, "user_id", actorID)
	}

	// Get booking
	schemaBooking, err := s.repo.GetBookingByID(ctx, bookingID)
	if err != nil {
		if s.log != nil {
			s.log.Error("failed to get booking", "booking_id", bookingID, "error", err)
		}
		return nil, nil, fmt.Errorf("failed to get booking: %w", err)
	}

	if schemaBooking == nil {
		return nil, nil, domain.ErrBookingNotFound
	}

	booking := domain.MapBookingFromSchema(schemaBooking)

	// Verify ownership
	if booking.GuestID != actorID {
		if s.log != nil {
			s.log.Warn("unauthorized payment attempt for booking", "booking_id", bookingID, "user_id", actorID)
		}
		return nil, nil, domain.ErrUnauthorized
	}

	// Verify booking can be paid
	if !booking.CanBePaid() {
		if s.log != nil {
			s.log.Warn("booking cannot be paid", "booking_id", bookingID, "status", booking.Status)
		}
		return nil, nil, domain.ErrCannotBePaid
	}

	// Check if hold expired
	if booking.HoldExpiresAt != nil && time.Now().After(*booking.HoldExpiresAt) {
		if s.log != nil {
			s.log.Warn("booking hold expired", "booking_id", bookingID, "hold_expires_at", booking.HoldExpiresAt)
		}
		return nil, nil, domain.ErrBookingExpired
	}

	// Initiate payment
	paymentInput := PaymentInput{
		BookingID:       bookingID,
		Amount:          int64(booking.TotalPrice * 100), // Convert to minor units
		Currency:        booking.Currency,
		PayerID:         booking.GuestID,
		PayerEmail:      booking.GuestEmail,
		PayerName:       booking.GuestName,
		PaymentMethodID: paymentMethodID,
		CallbackURL:     fmt.Sprintf("/bookings/%s/payment-callback", bookingID),
		Description:     fmt.Sprintf("Booking payment for %s", booking.ListingID),
	}

	paymentResult, err := s.payment.InitiatePayment(ctx, paymentInput)
	if err != nil {
		// Mark payment as failed but don't delete booking (user can retry)
		booking.Status = domain.BookingStatusPaymentFailed
		booking.UpdatedAt = time.Now()

		if updateErr := s.repo.UpdateBooking(ctx, domain.MapBookingFromDomain(booking)); updateErr != nil {
			if s.log != nil {
				s.log.Error("failed to update booking status after payment failure", "booking_id", bookingID, "error", updateErr)
			}
		}

		if s.log != nil {
			s.log.Error("payment initiation failed for booking", "booking_id", bookingID, "error", err)
		}
		s.notifyPaymentFailed(ctx, booking)
		return booking, nil, fmt.Errorf("payment initiation failed: %w", err)
	}

	// Update booking with payment reference
	booking.PaymentReference = &paymentResult.Reference
	booking.LastPaymentID = &paymentResult.PaymentID
	booking.UpdatedAt = time.Now()

	// If payment succeeded immediately, confirm booking
	switch paymentResult.Status {
	case "succeeded":
		if s.log != nil {
			s.log.Info(" payment succeeded immediately for booking", "booking_id", bookingID, "confirming_booking", true)
		}

		ownerID, err := s.listingHooks.GetListingOwner(ctx, booking.ListingID)
		if err != nil {
			if s.log != nil {
				s.log.Error("failed to get listing owner", "error", err)
			}
			ownerID = uuid.Nil
		}

		if _, err := s.confirmBookingAfterPayment(ctx, booking, ownerID, paymentResult.PaymentID); err != nil && s.log != nil {
			s.log.Warn("failed to finalize booking", "booking_id", booking.ID, "error", err)
		}
	case "failed":
		booking.Status = domain.BookingStatusPaymentFailed
		booking.UpdatedAt = time.Now()
	}
	// If status is "pending", user needs to complete authorization

	// Save updated booking
	if err := s.repo.UpdateBooking(ctx, domain.MapBookingFromDomain(booking)); err != nil {
		if s.log != nil {
			s.log.Error("failed to update booking", "error	", err)
		}
		return nil, nil, fmt.Errorf("failed to update booking: %w", err)
	}

	if paymentResult.Status == "failed" {
		s.notifyPaymentFailed(ctx, booking)
	}

	if s.log != nil {
		s.log.Info(" booking payment processed", "booking_id", bookingID, "payment_id", paymentResult.PaymentID, "status", paymentResult.Status)
	}

	return booking, paymentResult, nil
}
