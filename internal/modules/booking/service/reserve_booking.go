package service

import (
	"context"
	"errors"
	"fmt"
	"hauslet/internal/modules/booking/domain"
	calendardomain "hauslet/internal/modules/calendar/domain"
	"time"

	"github.com/google/uuid"
)

// ReserveBooking creates an instant booking and initiates payment in a single operation.
// This is used for auto-accept (instant) bookings only.
// For manual approval bookings, use CreateBooking (RequestBooking) instead.
func (s *BookingServiceImpl) ReserveBooking(
	ctx context.Context,
	listingID uuid.UUID,
	guestID uuid.UUID,
	checkIn, checkOut time.Time,
	guestCount int,
	paymentMethodID *uuid.UUID,
	specialRequests *string,
) (*domain.Booking, *PaymentResult, error) {
	if s.log != nil {
		s.log.Info("reserving instant booking", "listingID", listingID, "guestID", guestID)
	}

	// Get guest info
	guest, err := s.resolveGuestInfo(ctx, guestID)
	if err != nil {
		return nil, nil, err
	}

	// Get listing constraints
	constraints, err := s.listingHooks.GetListingConstraints(ctx, listingID)
	if err != nil {
		return nil, nil, err
	}

	// Get calendar config to check instant booking
	var calendarConfig *calendardomain.CalendarConfig
	if cfg, cfgErr := s.calendar.GetCalendarConfig(ctx, listingID); cfgErr == nil {
		calendarConfig = cfg
	}

	scheduledCheckIn, scheduledCheckOut, _ := s.normalizeScheduledTimes(checkIn, checkOut, constraints, calendarConfig)

	if err := s.ensureGuestMeetsBookingSettings(guest, constraints); err != nil {
		return nil, nil, err
	}

	// Validate constraints
	if err := s.validateBookingConstraints(checkIn, checkOut, guestCount, constraints, calendarConfig); err != nil {
		return nil, nil, err
	}

	availability, err := s.calendar.CheckAvailability(ctx, listingID, scheduledCheckIn, scheduledCheckOut)
	if err != nil {
		return nil, nil, err
	}
	if availability == nil || !availability.Available {
		return nil, nil, domain.ErrDatesUnavailable
	}

	autoAcceptBookings := constraints.AutoAcceptBookings
	if calendarConfig != nil {
		autoAcceptBookings = calendarConfig.InstantBooking
	}

	// This mutation is ONLY for instant bookings
	if !autoAcceptBookings {
		return nil, nil, fmt.Errorf("listing does not support instant booking; use requestBooking mutation instead")
	}

	// Check time until check-in
	hoursUntilCheckIn := time.Until(scheduledCheckIn).Hours()
	if hoursUntilCheckIn < 6 {
		// For very close bookings, instant is required but we're already in instant flow
		if s.log != nil {
			s.log.Info("booking very close to check-in, instant required", "hoursUntilCheckIn", hoursUntilCheckIn)
		}
	}

	// Check cleaning buffer availability
	bufferDuration := cleaningBufferDuration(calendarConfig)
	if bufferDuration > 0 {
		bufferAvailability, err := s.calendar.CheckAvailability(ctx, listingID, scheduledCheckOut, scheduledCheckOut.Add(bufferDuration))
		if err != nil {
			return nil, nil, err
		}
		if bufferAvailability == nil || !bufferAvailability.Available {
			return nil, nil, domain.ErrDatesUnavailable
		}
	}

	// Get listing owner
	ownerID, err := s.listingHooks.GetListingOwner(ctx, listingID)
	if err != nil {
		return nil, nil, err
	}

	// Prevent hosts from booking their own listings
	if ownerID == guestID {
		if s.log != nil {
			s.log.Warn("host attempted to reserve own listing", "listingID", listingID, "userID", guestID)
		}
		return nil, nil, domain.ErrCannotBookOwnListing
	}

	// Calculate pricing using scheduled times to ensure consistency with listing's check-in/out times
	var priceSnapshot *domain.PriceBreakdownSnapshot
	var total float64
	currency := constraints.Currency

	if s.pricing != nil {
		priceBreakdown, err := s.pricing.CalculatePrice(ctx, listingID, scheduledCheckIn, scheduledCheckOut, guestCount)
		if err != nil {
			if s.log != nil {
				s.log.Warn("pricing calculation failed", "error", err)
			}
			baseRate, baseCurrency, baseErr := s.pricing.GetBasePrice(ctx, listingID)
			if baseErr == nil {
				// Calculate nights using scheduled times normalized to dates
				checkInDate := time.Date(scheduledCheckIn.Year(), scheduledCheckIn.Month(), scheduledCheckIn.Day(), 0, 0, 0, 0, scheduledCheckIn.Location())
				checkOutDate := time.Date(scheduledCheckOut.Year(), scheduledCheckOut.Month(), scheduledCheckOut.Day(), 0, 0, 0, 0, scheduledCheckOut.Location())
				nights := int(checkOutDate.Sub(checkInDate).Hours() / 24)
				total = baseRate * float64(nights)
				currency = baseCurrency
			}
		} else {
			priceSnapshot = domain.NewPriceBreakdownSnapshotFromPricing(priceBreakdown)
			total = priceBreakdown.Total
			currency = priceBreakdown.Currency
		}
	}

	if total == 0 {
		// Calculate nights using scheduled times normalized to dates
		checkInDate := time.Date(scheduledCheckIn.Year(), scheduledCheckIn.Month(), scheduledCheckIn.Day(), 0, 0, 0, 0, scheduledCheckIn.Location())
		checkOutDate := time.Date(scheduledCheckOut.Year(), scheduledCheckOut.Month(), scheduledCheckOut.Day(), 0, 0, 0, 0, scheduledCheckOut.Location())
		nights := int(checkOutDate.Sub(checkInDate).Hours() / 24)
		total = float64(nights) * 1 // fallback
	}

	// Create calendar event
	bookingID := uuid.New()
	event := &calendardomain.CalendarEvent{
		ListingID: listingID,
		EventType: calendardomain.EventTypeBooking,
		Status:    calendardomain.EventStatusPending,
		StartTime: scheduledCheckIn,
		EndTime:   scheduledCheckOut,
		BookingID: &bookingID,
	}

	createdEvent, err := s.calendar.CreateEvent(ctx, event)
	if err != nil {
		if errors.Is(err, calendardomain.ErrUnauthorized) {
			if s.log != nil {
				s.log.Warn("calendar disabled for listing; cannot reserve booking", "listingID", listingID)
			}
			return nil, nil, fmt.Errorf("calendar disabled for listing; enable calendar to allow bookings")
		}
		return nil, nil, err
	}

	// Create cleaning buffer event if needed
	var cleaningEventID *uuid.UUID
	if bufferDuration > 0 {
		cleaningEvent, err := s.createCleaningBufferEvent(ctx, listingID, bookingID, scheduledCheckOut, bufferDuration)
		if err != nil {
			if s.log != nil {
				s.log.Error("failed to create cleaning buffer event", "error", err)
			}
			_ = s.calendar.DeleteEvent(ctx, createdEvent.ID, ownerID)
			return nil, nil, fmt.Errorf("failed to create cleaning buffer event: %w", err)
		}
		if cleaningEvent != nil {
			cleaningEventID = &cleaningEvent.ID
		}
	}

	// Create booking with awaiting_payment status and soft hold
	now := time.Now()
	holdExpiry := now.Add(softHoldDuration)

	booking := &domain.Booking{
		ID:              bookingID,
		ListingID:       listingID,
		CalendarEventID: createdEvent.ID,
		CleaningEventID: cleaningEventID,
		GuestID:         guest.ID,
		GuestName:       guest.Name,
		GuestEmail:      guest.Email,
		GuestPhone:      guest.Phone,
		GuestCount:      guestCount,
		Status:          domain.BookingStatusAwaitingPayment,
		BookingType:     domain.BookingInstant,
		CheckIn:         nil,
		CheckOut:        nil,
		CheckInTime:     &scheduledCheckIn,
		CheckOutTime:    &scheduledCheckOut,
		SpecialRequests: specialRequests,
		PriceBreakdown:  priceSnapshot,
		TotalPrice:      total,
		Currency:        currency,
		HoldExpiresAt:   &holdExpiry,
		PaymentDueAt:    &holdExpiry,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	// Persist booking
	schemaBooking := domain.MapBookingFromDomain(booking)
	if err := s.repo.CreateBooking(ctx, schemaBooking); err != nil {
		if s.log != nil {
			s.log.Error("failed to persist booking", "error", err)
		}
		_ = s.calendar.DeleteEvent(ctx, createdEvent.ID, ownerID)
		if cleaningEventID != nil {
			_ = s.calendar.DeleteEvent(ctx, *cleaningEventID, ownerID)
		}
		return nil, nil, err
	}

	booking.BookingReference = schemaBooking.BookingReference

	// Initiate payment immediately
	if s.payment == nil {
		return nil, nil, fmt.Errorf("payment service unavailable")
	}

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
		// Mark payment as failed but don't delete booking (can retry)
		booking.Status = domain.BookingStatusPaymentFailed
		booking.UpdatedAt = time.Now()
		if updateErr := s.repo.UpdateBooking(ctx, domain.MapBookingFromDomain(booking)); updateErr != nil && s.log != nil {
			s.log.Error("failed to update booking after payment failure", "error", updateErr)
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
			s.log.Info(" payment succeeded immediately for booking", "bookingID", bookingID)
		}
		if _, err := s.confirmBookingAfterPayment(ctx, booking, ownerID, paymentResult.PaymentID); err != nil && s.log != nil {
			s.log.Warn("failed to confirm booking after immediate payment success", "error", err)
		}
	case "failed":
		booking.Status = domain.BookingStatusPaymentFailed
		booking.UpdatedAt = time.Now()
		s.notifyPaymentFailed(ctx, booking)
	}
	// If status is "pending", user needs to complete 3DS authentication

	// Save updated booking
	if err := s.repo.UpdateBooking(ctx, domain.MapBookingFromDomain(booking)); err != nil {
		if s.log != nil {
			s.log.Error("failed to update booking", "error", err)
		}
		return nil, nil, err
	}

	if s.log != nil {
		s.log.Info("instant booking reserved", "bookingID", bookingID, "paymentID", paymentResult.PaymentID, "status", paymentResult.Status)
	}

	return booking, paymentResult, nil
}
