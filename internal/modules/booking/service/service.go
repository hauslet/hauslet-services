package service

import (
	"context"
	"fmt"
	"hauslet/internal/modules/booking/domain"
	calendardomain "hauslet/internal/modules/calendar/domain"

	"time"

	"github.com/google/uuid"
)

const (
	softHoldDuration           = 10 * time.Minute
	defaultCleaningBufferHours = 1
	cleaningBlockReason        = "cleaning_buffer"
)

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
		s.log.Warn("auto-charge attempt failed for booking", "bookingID", booking.ID, "error", err)
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

func stringPtr(v string) *string {
	return &v
}

func (s *BookingServiceImpl) getUserContact(ctx context.Context, userID uuid.UUID) *ContactInfo {
	if s.profiles == nil {
		return nil
	}
	contact, err := s.profiles.GetUserContact(ctx, userID)
	if err != nil {
		if s.log != nil {
			s.log.Warn("failed to fetch user contact", "userID", userID, "error", err)
		}
		return nil
	}
	return contact
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
			s.log.Error("failed to update booking status after auto-charge failure", "error", updateErr)
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
				s.log.Warn("failed to get calendar event", "eventID", booking.CalendarEventID, "error", err)
			}
		} else if event != nil {
			event.Status = calendardomain.EventStatusConfirmed
			if _, err := s.calendar.UpdateEvent(ctx, event, ownerID); err != nil {
				if s.log != nil {
					s.log.Warn("failed to update calendar event status", "error", err)
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
