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

func (s *BookingServiceImpl) CreateBooking(ctx context.Context, listingID uuid.UUID, guestID uuid.UUID, checkIn, checkOut time.Time, guestCount int, specialRequests *string) (*domain.Booking, error) {
	if s.log != nil {
		s.log.Info(" creating booking", "listing_id", listingID, "guest_id", guestID)
	}

	guest, err := s.resolveGuestInfo(ctx, guestID)
	if err != nil {
		return nil, err
	}

	constraints, err := s.listingHooks.GetListingConstraints(ctx, listingID)
	if err != nil {
		return nil, err
	}

	if err := s.validateBookingConstraints(checkIn, checkOut, guestCount, constraints); err != nil {
		return nil, err
	}

	availability, err := s.calendar.CheckAvailability(ctx, listingID, checkIn, checkOut)
	if err != nil {
		return nil, err
	}
	if availability == nil || !availability.Available {
		return nil, domain.ErrDatesUnavailable
	}

	var calendarConfig *calendardomain.CalendarConfig
	if cfg, cfgErr := s.calendar.GetCalendarConfig(ctx, listingID); cfgErr == nil {
		calendarConfig = cfg
	} else if s.log != nil {
		s.log.Warn("calendar config unavailable for listing", "listing_id", listingID, "error", cfgErr)
	}

	autoAcceptBookings := false
	if constraints != nil {
		autoAcceptBookings = constraints.AutoAcceptBookings
	}
	if calendarConfig != nil {
		autoAcceptBookings = calendarConfig.InstantBooking
		if constraints != nil {
			constraints.AutoAcceptBookings = autoAcceptBookings
		}
	}

	bufferDuration := cleaningBufferDuration(calendarConfig)
	if bufferDuration > 0 {
		bufferAvailability, err := s.calendar.CheckAvailability(ctx, listingID, checkOut, checkOut.Add(bufferDuration))
		if err != nil {
			return nil, err
		}
		if bufferAvailability == nil || !bufferAvailability.Available {
			return nil, domain.ErrDatesUnavailable
		}
	}

	ownerID, err := s.listingHooks.GetListingOwner(ctx, listingID)
	if err != nil {
		return nil, err
	}

	var priceSnapshot *domain.PriceBreakdownSnapshot
	var total float64
	currency := constraints.Currency

	if s.pricing != nil {
		priceBreakdown, err := s.pricing.CalculatePrice(ctx, listingID, checkIn, checkOut, guestCount)
		if err != nil {
			// fallback to base price if we can
			if s.log != nil {
				s.log.Warn("pricing calculation failed", "error", err)
			}
			baseRate, baseCurrency, baseErr := s.pricing.GetBasePrice(ctx, listingID)
			if baseErr == nil {
				nights := int(checkOut.Sub(checkIn).Hours() / 24)
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
		nights := int(checkOut.Sub(checkIn).Hours() / 24)
		total = float64(nights) * 1 // fallback; replaced by proper pricing
	}

	bookingID := uuid.New()
	event := &calendardomain.CalendarEvent{
		ListingID: listingID,
		EventType: calendardomain.EventTypeBooking,
		Status:    calendardomain.EventStatusPending,
		StartTime: checkIn,
		EndTime:   checkOut,
		BookingID: &bookingID,
	}

	createdEvent, err := s.calendar.CreateEvent(ctx, event)
	if err != nil {
		if errors.Is(err, calendardomain.ErrUnauthorized) {
			if s.log != nil {
				s.log.Warn("calendar disabled for listing; cannot create booking", "listing_id", listingID)
			}
			return nil, fmt.Errorf("calendar disabled for listing; enable calendar to allow bookings")
		}
		return nil, err
	}

	var cleaningEventID *uuid.UUID
	if bufferDuration > 0 {
		cleaningEvent, err := s.createCleaningBufferEvent(ctx, listingID, bookingID, checkOut, bufferDuration)
		if err != nil {
			if s.log != nil {
				s.log.Error("failed to create cleaning buffer event for booking", "booking_id", bookingID, "error", err)
			}
			_ = s.calendar.DeleteEvent(ctx, createdEvent.ID, ownerID)
			return nil, fmt.Errorf("failed to create cleaning buffer event: %w", err)
		}
		if cleaningEvent != nil {
			cleaningEventID = &cleaningEvent.ID
		}
	}

	now := time.Now()

	// Determine booking flow based on AutoAcceptBookings and time to check-in
	bookingType, holdDuration, err := s.determineBookingFlow(checkIn, constraints)
	if err != nil {
		_ = s.calendar.DeleteEvent(ctx, createdEvent.ID, ownerID)
		if cleaningEventID != nil {
			_ = s.calendar.DeleteEvent(ctx, *cleaningEventID, ownerID)
		}
		return nil, err
	}

	holdExpiry := now.Add(holdDuration)
	initialStatus := s.initialStatus(bookingType)

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
		Status:          initialStatus,
		BookingType:     bookingType,
		CheckIn:         checkIn,
		CheckOut:        checkOut,
		CheckInTime:     constraints.CheckInTime,
		CheckOutTime:    constraints.CheckOutTime,
		SpecialRequests: specialRequests,
		PriceBreakdown:  priceSnapshot,
		TotalPrice:      total,
		Currency:        currency,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	booking.HoldExpiresAt = &holdExpiry
	if initialStatus == domain.BookingStatusAwaitingPayment {
		booking.MarkAwaitingPayment(holdExpiry)
	}

	if err := s.repo.CreateBooking(ctx, domain.MapBookingFromDomain(booking)); err != nil {
		if s.log != nil {
			s.log.Error("failed to persist booking, rolling back calendar event", "error", err)
		}
		_ = s.calendar.DeleteEvent(ctx, createdEvent.ID, ownerID)
		return nil, err
	}

	s.notifyBookingCreation(ctx, booking, ownerID)

	return booking, nil
}

func (s *BookingServiceImpl) validateBookingConstraints(checkIn, checkOut time.Time, guestCount int, constraints *ListingConstraints) error {
	if constraints == nil {
		return nil
	}

	if !checkOut.After(checkIn) {
		return domain.ErrInvalidDateRange
	}

	nights := int(checkOut.Sub(checkIn).Hours() / 24)
	if nights < constraints.MinNights {
		return domain.ErrMinimumStayNotMet
	}

	if constraints.MaxNights != nil && nights > *constraints.MaxNights {
		return domain.ErrMaximumStayExceeded
	}

	if guestCount > constraints.MaxGuests {
		return domain.ErrGuestCountExceeded
	}

	if checkIn.Before(time.Now()) {
		return domain.ErrBookingInPast
	}

	return nil
}

func (s *BookingServiceImpl) createCleaningBufferEvent(ctx context.Context, listingID, bookingID uuid.UUID, checkout time.Time, buffer time.Duration) (*calendardomain.CalendarEvent, error) {
	if buffer <= 0 {
		return nil, nil
	}

	event := &calendardomain.CalendarEvent{
		ListingID: listingID,
		EventType: calendardomain.EventTypeBlock,
		Status:    calendardomain.EventStatusConfirmed,
		StartTime: checkout,
		EndTime:   checkout.Add(buffer),
		BookingID: &bookingID,
		BlockDetails: &calendardomain.BlockDetail{
			Reason:    cleaningBlockReason,
			Notes:     stringPtr("Auto-generated cleaning buffer"),
			OwnerStay: false,
		},
	}

	return s.calendar.CreateEvent(ctx, event)
}

func cleaningBufferDuration(cfg *calendardomain.CalendarConfig) time.Duration {
	if cfg != nil && cfg.BufferHours > 0 {
		return time.Duration(cfg.BufferHours) * time.Hour
	}
	return time.Duration(defaultCleaningBufferHours) * time.Hour
}

func (s *BookingServiceImpl) resolveGuestInfo(ctx context.Context, guestID uuid.UUID) (*ContactInfo, error) {
	if s.profiles == nil {
		return nil, domain.ErrGuestProfileNotFound
	}
	contact, err := s.profiles.GetUserContact(ctx, guestID)
	if err != nil {
		return nil, err
	}
	if contact == nil || contact.Name == "" || contact.Email == "" {
		return nil, domain.ErrGuestProfileNotFound
	}
	return contact, nil
}

func (s *BookingServiceImpl) notifyBookingCreation(ctx context.Context, booking *domain.Booking, hostID uuid.UUID) {
	if s.notifier == nil || booking == nil {
		return
	}

	switch booking.BookingType {
	case domain.BookingRequest:
		hostContact := s.getUserContact(ctx, hostID)
		hostName, hostEmail := "", ""
		if hostContact != nil {
			hostName = hostContact.Name
			hostEmail = hostContact.Email
		}
		if hostEmail != "" {
			s.notifier.SendHostApprovalRequest(ctx, booking, hostName, hostEmail)
		}
		s.notifier.SendGuestRequestReceipt(ctx, booking)
	case domain.BookingInstant:
		// No email sent here for instant bookings
		// Instant bookings should use ReserveBooking mutation which initiates payment immediately
		// Confirmation email sent after payment succeeds
		return
	default:
		return
	}
}
