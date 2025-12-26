package service

import (
	"context"
	"fmt"
	"hauslet/internal/modules/booking/domain"
	calendardomain "hauslet/internal/modules/calendar/domain"
	pricingdomain "hauslet/internal/modules/pricing/domain"
	"time"

	"github.com/google/uuid"
)

const (
	softHoldDuration           = 10 * time.Minute
	defaultCleaningBufferHours = 1
	cleaningBlockReason        = "cleaning_buffer"
)

func (s *BookingServiceImpl) CreateBooking(ctx context.Context, listingID uuid.UUID, guestID uuid.UUID, checkIn, checkOut time.Time, guestCount int, specialRequests *string) (*domain.Booking, error) {
	if s.log != nil {
		s.log.Logf("INFO creating booking listing=%s guest=%s", listingID, guestID)
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
		s.log.Logf("WARN calendar config unavailable for listing=%s: %v", listingID, cfgErr)
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
				s.log.Logf("WARN pricing calculation failed: %v", err)
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
		return nil, err
	}

	var cleaningEventID *uuid.UUID
	if bufferDuration > 0 {
		cleaningEvent, err := s.createCleaningBufferEvent(ctx, listingID, bookingID, checkOut, bufferDuration)
		if err != nil {
			if s.log != nil {
				s.log.Logf("ERROR failed to create cleaning buffer event for booking=%s: %v", bookingID, err)
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
			s.log.Logf("ERROR failed to persist booking, rolling back calendar event: %v", err)
		}
		_ = s.calendar.DeleteEvent(ctx, createdEvent.ID, ownerID)
		return nil, err
	}

	s.notifyBookingCreation(ctx, booking, ownerID)

	return booking, nil
}

func (s *BookingServiceImpl) ConfirmBooking(ctx context.Context, bookingID uuid.UUID, actorID uuid.UUID) (*domain.Booking, error) {
	booking, ownerID, err := s.getBookingWithOwner(ctx, bookingID, actorID)
	if err != nil {
		return nil, err
	}

	if booking.Status != domain.BookingStatusPendingApproval {
		return nil, domain.ErrCannotConfirm
	}

	now := time.Now()
	holdExpiry := now.Add(softHoldDuration)
	booking.MarkAwaitingPayment(holdExpiry)
	booking.HoldExpiresAt = &holdExpiry
	booking.UpdatedAt = now

	schemaBooking := domain.MapBookingFromDomain(booking)
	if err := s.repo.UpdateBooking(ctx, schemaBooking); err != nil {
		return nil, err
	}

	approvedBooking := booking
	paymentResult, err := s.tryAutoCharge(ctx, booking, ownerID)
	if err != nil && s.log != nil {
		s.log.Logf("WARN auto-charge attempt failed for booking=%s: %v", booking.ID, err)
	}

	if paymentResult != nil && paymentResult.Status == "succeeded" {
		if err != nil {
			return nil, err
		}
		return approvedBooking, nil
	}

	s.notifyBookingApproval(ctx, booking, paymentResult)
	return approvedBooking, nil
}

func (s *BookingServiceImpl) CancelBooking(ctx context.Context, bookingID uuid.UUID, actorID uuid.UUID, reason *string) (*domain.Booking, error) {
	if s.log != nil {
		s.log.Logf("INFO cancelling booking=%s actor=%s", bookingID, actorID)
	}

	booking, ownerID, err := s.getBookingWithOwner(ctx, bookingID, actorID)
	if err != nil {
		return nil, err
	}

	if !booking.CanBeCancelled() {
		return nil, domain.ErrCannotCancel
	}

	// Determine who is cancelling
	cancelledBy := s.determineCancellationActor(booking, actorID, ownerID)
	wasPendingApproval := booking.Status == domain.BookingStatusPendingApproval

	// Get listing constraints for refund policy
	constraints, err := s.listingHooks.GetListingConstraints(ctx, booking.ListingID)
	if err != nil {
		if s.log != nil {
			s.log.Logf("WARN failed to get listing constraints: %v", err)
		}
		// Continue with cancellation even if we can't get constraints
	}

	// Calculate refund if payment exists and pricing service is available
	var refundAmount int64
	var refundBreakdown *pricingdomain.RefundBreakdown
	if booking.LastPaymentID != nil && s.pricing != nil && booking.TotalPrice > 0 {
		// Convert TotalPrice to minor units
		currencyMinorUnit := int64(100) // Default to 100 (for most currencies)

		refundInput := pricingdomain.RefundCalculationInput{
			BookingID:        booking.ID,
			TotalPaid:        booking.TotalPrice,
			Currency:         booking.Currency,
			BookingCreatedAt: booking.CreatedAt,
			CheckInTime:      booking.CheckIn,
			CancellationTime: time.Now(),
			RefundPolicy:     "moderate", // Default
			CancelledBy:      pricingdomain.CancellationActor(cancelledBy),
			Reason:           reason,
		}

		// Use listing's refund policy if available
		if constraints != nil && constraints.RefundPolicy != "" {
			refundInput.RefundPolicy = constraints.RefundPolicy
		}

		refundBreakdown, err = s.pricing.CalculateRefund(ctx, refundInput)
		if err != nil {
			if s.log != nil {
				s.log.Logf("WARN failed to calculate refund: %v", err)
			}
		} else {
			// Convert refund amount to minor units
			refundAmount = int64(refundBreakdown.NetRefund * float64(currencyMinorUnit))

			if s.log != nil {
				s.log.Logf("INFO refund calculated: booking=%s, amount=%.2f %s, policy=%s, cancelled_by=%s",
					bookingID, refundBreakdown.NetRefund, booking.Currency, refundBreakdown.AppliedPolicy, cancelledBy)
			}
		}
	}

	// Process refund if amount > 0 and payment gateway is available
	var refundReference *string
	if refundAmount > 0 && booking.LastPaymentID != nil && s.payment != nil {
		if s.log != nil {
			s.log.Logf("INFO initiating refund: booking=%s, payment=%s, amount=%d",
				bookingID, *booking.LastPaymentID, refundAmount)
		}

		refundInput := RefundPaymentInput{
			PaymentID:  *booking.LastPaymentID,
			Amount:     &refundAmount,
			Reason:     s.buildRefundReason(cancelledBy, reason, refundBreakdown),
			RefundedBy: actorID,
		}

		refundResult, err := s.payment.RefundPayment(ctx, refundInput)
		if err != nil {
			if s.log != nil {
				s.log.Logf("ERROR failed to process refund: %v", err)
			}
			// Don't fail the cancellation if refund fails - log and continue
			// The refund can be processed manually or retried later
		} else {
			refundReference = &refundResult.RefundID
			if s.log != nil {
				s.log.Logf("INFO refund initiated successfully: booking=%s, refund_id=%s",
					bookingID, refundResult.RefundID)
			}
		}
	}

	// Cancel calendar events
	if err := s.calendar.CancelEvent(ctx, booking.CalendarEventID, ownerID); err != nil {
		return nil, err
	}
	if booking.CleaningEventID != nil {
		if err := s.calendar.CancelEvent(ctx, *booking.CleaningEventID, ownerID); err != nil && s.log != nil {
			s.log.Logf("WARN failed to cancel cleaning buffer event for booking=%s: %v", booking.ID, err)
		}
	}

	// Update booking with cancellation and refund details
	now := time.Now()
	booking.MarkCancelled(now)

	cancelledByStr := string(cancelledBy)
	booking.CancelledBy = &cancelledByStr

	if refundAmount > 0 {
		booking.RefundAmount = refundAmount
		booking.RefundInitiatedAt = &now
		booking.RefundReason = reason
		booking.RefundReference = refundReference
	}

	if err := s.repo.UpdateBooking(ctx, domain.MapBookingFromDomain(booking)); err != nil {
		return nil, err
	}

	// Send appropriate notifications
	if cancelledBy == pricingdomain.CancelledByHost && wasPendingApproval {
		// Host rejected a pending request - send rejection email
		s.notifyBookingRejection(ctx, booking, reason)
	} else {
		// Booking was confirmed/in-progress and got cancelled - send cancellation emails
		s.notifyBookingCancellation(ctx, booking, ownerID, string(cancelledBy), reason)
	}

	if s.log != nil {
		s.log.Logf("INFO booking cancelled successfully: booking=%s, refund_amount=%d, cancelled_by=%s",
			bookingID, refundAmount, cancelledBy)
	}

	return booking, nil
}

// determineCancellationActor determines who initiated the cancellation
func (s *BookingServiceImpl) determineCancellationActor(booking *domain.Booking, actorID uuid.UUID, ownerID uuid.UUID) pricingdomain.CancellationActor {
	switch {
	case actorID == booking.GuestID:
		return pricingdomain.CancelledByGuest
	case actorID == ownerID:
		return pricingdomain.CancelledByHost
	default:
		// Admin or system cancellation
		return pricingdomain.CancelledByAdmin
	}
}

// buildRefundReason creates a human-readable refund reason
func (s *BookingServiceImpl) buildRefundReason(cancelledBy pricingdomain.CancellationActor, userReason *string, breakdown *pricingdomain.RefundBreakdown) string {
	baseReason := fmt.Sprintf("Booking cancelled by %s", cancelledBy)

	if breakdown != nil {
		baseReason = fmt.Sprintf("%s - %s", baseReason, breakdown.Summary)
	}

	if userReason != nil && *userReason != "" {
		baseReason = fmt.Sprintf("%s. Reason: %s", baseReason, *userReason)
	}

	return baseReason
}

func (s *BookingServiceImpl) GetBooking(ctx context.Context, bookingID uuid.UUID, requestorID uuid.UUID) (*domain.Booking, error) {
	schemaBooking, err := s.repo.GetBookingByID(ctx, bookingID)
	if err != nil {
		return nil, err
	}

	booking := domain.MapBookingFromSchema(schemaBooking)

	if booking.GuestID != requestorID {
		ownerID, err := s.listingHooks.GetListingOwner(ctx, booking.ListingID)
		if err != nil {
			return nil, err
		}
		if ownerID != requestorID {
			return nil, domain.ErrUnauthorized
		}
	}

	return booking, nil
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

func (s *BookingServiceImpl) getBookingWithOwner(ctx context.Context, bookingID uuid.UUID, actorID uuid.UUID) (*domain.Booking, uuid.UUID, error) {
	schemaBooking, err := s.repo.GetBookingByID(ctx, bookingID)
	if err != nil {
		return nil, uuid.Nil, err
	}

	booking := domain.MapBookingFromSchema(schemaBooking)
	ownerID, err := s.listingHooks.GetListingOwner(ctx, booking.ListingID)
	if err != nil {
		return nil, uuid.Nil, err
	}

	if actorID != booking.GuestID && actorID != ownerID {
		return nil, uuid.Nil, domain.ErrUnauthorized
	}

	return booking, ownerID, nil
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

func stringPtr(v string) *string {
	return &v
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

func (s *BookingServiceImpl) getUserContact(ctx context.Context, userID uuid.UUID) *ContactInfo {
	if s.profiles == nil {
		return nil
	}
	contact, err := s.profiles.GetUserContact(ctx, userID)
	if err != nil {
		if s.log != nil {
			s.log.Logf("WARN failed to fetch user contact user=%s: %v", userID, err)
		}
		return nil
	}
	return contact
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

func (s *BookingServiceImpl) notifyBookingApproval(ctx context.Context, booking *domain.Booking, paymentResult *PaymentResult) {
	if s.notifier == nil || booking == nil {
		return
	}

	var paymentURL *string
	if paymentResult != nil && paymentResult.AuthorizationURL != nil {
		paymentURL = paymentResult.AuthorizationURL
	}

	s.notifier.SendGuestApprovalGranted(ctx, booking, paymentURL)
}

func (s *BookingServiceImpl) notifyBookingConfirmed(ctx context.Context, booking *domain.Booking, hostID uuid.UUID) {
	if s.notifier == nil || booking == nil {
		return
	}

	s.notifier.SendGuestBookingConfirmed(ctx, booking)

	if hostID == uuid.Nil {
		return
	}

	hostContact := s.getUserContact(ctx, hostID)
	if hostContact == nil || hostContact.Email == "" {
		return
	}

	s.notifier.SendHostBookingConfirmed(ctx, booking, hostContact.Name, hostContact.Email)
}

func (s *BookingServiceImpl) notifyBookingRejection(ctx context.Context, booking *domain.Booking, reason *string) {
	if s.notifier == nil || booking == nil {
		return
	}

	s.notifier.SendGuestBookingRejected(ctx, booking, reason)
}

func (s *BookingServiceImpl) notifyBookingCancellation(ctx context.Context, booking *domain.Booking, ownerID uuid.UUID, cancelledBy string, reason *string) {
	if s.notifier == nil || booking == nil {
		return
	}

	// Notify guest about cancellation
	s.notifier.SendGuestBookingCancelled(ctx, booking, cancelledBy, reason)

	// Notify host about cancellation
	hostContact := s.getUserContact(ctx, ownerID)
	if hostContact != nil && hostContact.Email != "" {
		s.notifier.SendHostBookingCancelled(ctx, booking, hostContact.Name, hostContact.Email, cancelledBy, reason)
	}
}

func (s *BookingServiceImpl) notifyPaymentFailed(ctx context.Context, booking *domain.Booking) {
	if s.notifier == nil || booking == nil {
		return
	}

	s.notifier.SendGuestPaymentFailed(ctx, booking)
}

func (s *BookingServiceImpl) notifyBookingExpired(ctx context.Context, booking *domain.Booking, previousStatus domain.BookingStatus) {
	if s.notifier == nil || booking == nil {
		return
	}

	s.notifier.SendGuestBookingExpired(ctx, booking, previousStatus)
}

func (s *BookingServiceImpl) tryAutoCharge(ctx context.Context, booking *domain.Booking, ownerID uuid.UUID) (*PaymentResult, error) {
	if s.payment == nil || booking == nil {
		return nil, nil
	}

	methodID, err := s.payment.GetDefaultPaymentMethodID(ctx, booking.GuestID)
	if err != nil {
		return nil, err
	}
	if methodID == nil {
		return nil, nil
	}

	paymentInput := PaymentInput{
		BookingID:       booking.ID,
		Amount:          int64(booking.TotalPrice * 100),
		Currency:        booking.Currency,
		PayerID:         booking.GuestID,
		PayerEmail:      booking.GuestEmail,
		PayerName:       booking.GuestName,
		PaymentMethodID: methodID,
		CallbackURL:     fmt.Sprintf("/bookings/%s/payment-callback", booking.ID),
		Description:     fmt.Sprintf("Booking payment for %s", booking.ListingID),
	}

	paymentResult, err := s.payment.InitiatePayment(ctx, paymentInput)
	if err != nil {
		booking.Status = domain.BookingStatusPaymentFailed
		booking.UpdatedAt = time.Now()
		if updateErr := s.repo.UpdateBooking(ctx, domain.MapBookingFromDomain(booking)); updateErr != nil && s.log != nil {
			s.log.Logf("ERROR failed to update booking status after auto-charge failure: %v", updateErr)
		}
		return nil, err
	}

	booking.PaymentReference = &paymentResult.Reference
	booking.LastPaymentID = &paymentResult.PaymentID
	booking.UpdatedAt = time.Now()

	switch paymentResult.Status {
	case "failed":
		booking.Status = domain.BookingStatusPaymentFailed
	case "succeeded":
		if _, err := s.confirmBookingAfterPayment(ctx, booking, ownerID, paymentResult.PaymentID); err != nil {
			return paymentResult, err
		}
	}

	if err := s.repo.UpdateBooking(ctx, domain.MapBookingFromDomain(booking)); err != nil {
		return paymentResult, err
	}

	return paymentResult, nil
}

func (s *BookingServiceImpl) confirmBookingAfterPayment(ctx context.Context, booking *domain.Booking, ownerID uuid.UUID, paymentID uuid.UUID) (*domain.Booking, error) {
	if booking == nil {
		return nil, nil
	}
	if booking.IsConfirmed() {
		return booking, nil
	}

	now := time.Now()
	booking.LastPaymentID = &paymentID
	booking.MarkConfirmed(now)
	booking.UpdatedAt = now

	if err := s.repo.UpdateBooking(ctx, domain.MapBookingFromDomain(booking)); err != nil {
		return nil, err
	}

	if ownerID != uuid.Nil {
		event, err := s.calendar.GetEvent(ctx, booking.CalendarEventID, ownerID)
		if err != nil {
			if s.log != nil {
				s.log.Logf("WARN failed to get calendar event %s: %v", booking.CalendarEventID, err)
			}
		} else if event != nil {
			event.Status = calendardomain.EventStatusConfirmed
			if _, err := s.calendar.UpdateEvent(ctx, event, ownerID); err != nil {
				if s.log != nil {
					s.log.Logf("WARN failed to update calendar event status: %v", err)
				}
			}
		}

		if booking.CleaningEventID != nil {
			cleanEvent, err := s.calendar.GetEvent(ctx, *booking.CleaningEventID, ownerID)
			if err == nil && cleanEvent != nil {
				cleanEvent.Status = calendardomain.EventStatusConfirmed
				_, _ = s.calendar.UpdateEvent(ctx, cleanEvent, ownerID)
			}
		}
	}

	s.notifyBookingConfirmed(ctx, booking, ownerID)
	return booking, nil
}

// determineBookingFlow calculates booking type and hold duration based on AutoAcceptBookings and time to check-in
func (s *BookingServiceImpl) determineBookingFlow(
	checkIn time.Time,
	constraints *ListingConstraints,
) (domain.BookingType, time.Duration, error) {
	hoursUntilCheckIn := time.Until(checkIn).Hours()
	autoAccept := constraints != nil && constraints.AutoAcceptBookings

	// Less than 6 hours: Must be instant book
	if hoursUntilCheckIn < 6 {
		if !autoAccept {
			return "", 0, domain.ErrTooCloseToCheckIn
		}
		return domain.BookingInstant, softHoldDuration, nil
	}

	// Auto-accept enabled: instant book
	if autoAccept {
		return domain.BookingInstant, softHoldDuration, nil
	}

	// Manual approval required: request-to-book
	responseWindow := s.calculateResponseWindow(hoursUntilCheckIn)
	return domain.BookingRequest, responseWindow, nil
}

// calculateResponseWindow determines how long host has to respond based on urgency
func (s *BookingServiceImpl) calculateResponseWindow(hoursUntilCheckIn float64) time.Duration {
	switch {
	case hoursUntilCheckIn > 72:
		return 24 * time.Hour

	case hoursUntilCheckIn >= 24:
		return 12 * time.Hour

	case hoursUntilCheckIn >= 6:
		// Sliding scale: 6 hours at 24h out, 2 hours at 6h out
		// Formula: 2 + ((hoursUntilCheckIn - 6) / 18) * 4
		hours := 2.0 + ((hoursUntilCheckIn-6.0)/18.0)*4.0
		return time.Duration(hours) * time.Hour

	default:
		return 2 * time.Hour // Minimum
	}
}

// initialStatus determines the initial booking status based on booking type
func (s *BookingServiceImpl) initialStatus(bookingType domain.BookingType) domain.BookingStatus {
	if bookingType == domain.BookingInstant {
		return domain.BookingStatusAwaitingPayment
	}
	return domain.BookingStatusPendingApproval
}
