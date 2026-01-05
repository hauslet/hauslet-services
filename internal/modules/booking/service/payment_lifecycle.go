package service

import (
	"context"
	"fmt"
	"hauslet/internal/modules/booking/domain"
	"time"

	"github.com/google/uuid"
)

// HandlePaymentSuccess processes successful payment events.
// Transitions booking from awaiting_payment -> confirmed status.
func (s *BookingServiceImpl) HandlePaymentSuccess(ctx context.Context, bookingID uuid.UUID, paymentID uuid.UUID) error {
	if s.log != nil {
		s.log.Info(" handling payment success", "booking_id", bookingID, "payment_id", paymentID)
	}

	// Get the booking
	schemaBooking, err := s.repo.GetBookingByID(ctx, bookingID)
	if err != nil {
		if s.log != nil {
			s.log.Error("failed to get booking", "booking_id", bookingID, "error", err)
		}
		return fmt.Errorf("failed to get booking: %w", err)
	}

	if schemaBooking == nil {
		return domain.ErrBookingNotFound
	}

	booking := domain.MapBookingFromSchema(schemaBooking)

	if booking.IsConfirmed() {
		if s.log != nil {
			s.log.Info(" booking already confirmed, skipping", "booking_id", bookingID)
		}
		return nil
	}

	// Verify booking can be confirmed
	if booking.Status != domain.BookingStatusAwaitingPayment &&
		booking.Status != domain.BookingStatusPaymentFailed &&
		booking.Status != domain.BookingStatusDraft {
		if s.log != nil {
			s.log.Warn("booking cannot be confirmed", "booking_id", bookingID, "status", booking.Status)
		}
		return domain.ErrCannotConfirm
	}

	// Get the listing owner for calendar operations
	ownerID, err := s.listingHooks.GetListingOwner(ctx, booking.ListingID)
	if err != nil {
		if s.log != nil {
			s.log.Error("failed to get listing owner", "error", err)
		}
		return fmt.Errorf("failed to get listing owner: %w", err)
	}

	if _, err := s.confirmBookingAfterPayment(ctx, booking, ownerID, paymentID); err != nil {
		if s.log != nil {
			s.log.Error("failed to confirm booking after payment", "error", err)
		}
		return fmt.Errorf("failed to confirm booking: %w", err)
	}

	if s.log != nil {
		s.log.Info(" booking confirmed successfully", "booking_id", bookingID, "payment_id", paymentID)
	}

	return nil
}

// HandlePaymentFailure processes failed payment events.
// Logs the failure but doesn't immediately cancel - user may retry within hold window.
func (s *BookingServiceImpl) HandlePaymentFailure(ctx context.Context, bookingID uuid.UUID, paymentID uuid.UUID, reason string) error {
	if s.log != nil {
		s.log.Warn("payment failed for booking", "booking_id", bookingID, "payment_id", paymentID, "reason", reason)
	}

	// Get the booking
	schemaBooking, err := s.repo.GetBookingByID(ctx, bookingID)
	if err != nil {
		if s.log != nil {
			s.log.Error("failed to get booking", "booking_id", bookingID, "error", err)
		}
		return fmt.Errorf("failed to get booking: %w", err)
	}

	if schemaBooking == nil {
		return domain.ErrBookingNotFound
	}

	booking := domain.MapBookingFromSchema(schemaBooking)

	// Update last payment ID to track the failed attempt
	booking.LastPaymentID = &paymentID
	booking.Status = domain.BookingStatusPaymentFailed
	booking.UpdatedAt = time.Now()

	// Save updated booking
	if err := s.repo.UpdateBooking(ctx, domain.MapBookingFromDomain(booking)); err != nil {
		if s.log != nil {
			s.log.Error("failed to update booking", "error", err)
		}
		return fmt.Errorf("failed to update booking: %w", err)
	}

	s.notifyPaymentFailed(ctx, booking)

	// NOTE: We don't auto-cancel here. The CRON job will archive bookings
	// where hold_expires_at has passed and no successful payment exists.

	if s.log != nil {
		s.log.Info(" payment failure logged for booking", "booking_id", bookingID)
	}

	return nil
}

// HandlePaymentRefund processes refund events.
// May update booking status or initiate cancellation flow.
func (s *BookingServiceImpl) HandlePaymentRefund(ctx context.Context, bookingID uuid.UUID, paymentID uuid.UUID, refundedAmount int64) error {
	if s.log != nil {
		s.log.Info(" handling payment refund for booking", "booking_id", bookingID, "payment_id", paymentID, "amount", refundedAmount)
	}

	// Get the booking
	schemaBooking, err := s.repo.GetBookingByID(ctx, bookingID)
	if err != nil {
		if s.log != nil {
			s.log.Error("failed to get booking", "booking_id", bookingID, "error", err)
		}
		return fmt.Errorf("failed to get booking: %w", err)
	}

	if schemaBooking == nil {
		return domain.ErrBookingNotFound
	}

	booking := domain.MapBookingFromSchema(schemaBooking)

	// If booking is not already cancelled, transition to disputed status
	// This signals that a refund occurred and may need manual review
	if booking.Status != domain.BookingStatusCancelled {
		booking.Status = domain.BookingStatusDisputed
		booking.UpdatedAt = time.Now()

		if s.log != nil {
			s.log.Info(" booking marked as disputed due to refund", "booking_id", bookingID)
		}
	}

	// Save updated booking
	if err := s.repo.UpdateBooking(ctx, domain.MapBookingFromDomain(booking)); err != nil {
		if s.log != nil {
			s.log.Error("failed to update booking", "error", err)
		}
		return fmt.Errorf("failed to update booking: %w", err)
	}

	if s.log != nil {
		s.log.Info(" refund processed for booking", "booking_id", bookingID)
	}

	return nil
}

// ArchiveExpiredBookings finds bookings with expired payment holds and archives them.
// Called by CRON job to clean up unpaid bookings.
func (s *BookingServiceImpl) ArchiveExpiredBookings(ctx context.Context, expiredBefore time.Time) ([]uuid.UUID, error) {
	// Find bookings where:
	// - status = awaiting_payment
	// - hold_expires_at < expiredBefore
	expiredBookings, err := s.repo.FindExpiredHolds(ctx, expiredBefore)
	if err != nil {
		if s.log != nil {
			s.log.Error("failed to find expired bookings", "expired_before", expiredBefore, "error", err)
		}
		return nil, fmt.Errorf("failed to find expired bookings: %w", err)
	}

	if len(expiredBookings) == 0 {
		return []uuid.UUID{}, nil
	}

	archivedIDs := make([]uuid.UUID, 0, len(expiredBookings))

	for _, schemaBooking := range expiredBookings {
		booking := domain.MapBookingFromSchema(schemaBooking)
		expiredStatus := booking.Status

		// Get listing owner for calendar operations
		ownerID, err := s.listingHooks.GetListingOwner(ctx, booking.ListingID)
		if err != nil {
			if s.log != nil {
				s.log.Warn("failed to get listing owner for booking", "booking_id", booking.ID, "error", err)
			}
			continue
		}

		// Cancel calendar events
		if err := s.calendar.CancelEvent(ctx, booking.CalendarEventID, ownerID); err != nil {
			if s.log != nil {
				s.log.Warn("failed to cancel calendar event", "calendar_event_id", booking.CalendarEventID, "error", err)
			}
		}

		if booking.CleaningEventID != nil {
			if err := s.calendar.CancelEvent(ctx, *booking.CleaningEventID, ownerID); err != nil {
				if s.log != nil {
					s.log.Warn("failed to cancel cleaning event", "cleaning_event_id", *booking.CleaningEventID, "error", err)
				}
			}
		}

		// Mark booking as archived
		now := time.Now()
		booking.MarkArchived(now)

		// Save updated booking
		if err := s.repo.UpdateBooking(ctx, domain.MapBookingFromDomain(booking)); err != nil {
			if s.log != nil {
				s.log.Error("failed to archive booking", "booking_id", booking.ID, "error", err)
			}
			continue
		}

		s.notifyBookingExpired(ctx, booking, expiredStatus)

		archivedIDs = append(archivedIDs, booking.ID)
	}

	return archivedIDs, nil
}

// MarkAsSettled marks a booking as settled after host payout completes.
// Transitions booking from completed -> settled status.
// Called by finance module after successful payout to host.
func (s *BookingServiceImpl) MarkAsSettled(ctx context.Context, bookingID uuid.UUID) error {
	if s.log != nil {
		s.log.Info(" marking booking as settled", "booking_id", bookingID)
	}

	// Get the booking
	schemaBooking, err := s.repo.GetBookingByID(ctx, bookingID)
	if err != nil {
		if s.log != nil {
			s.log.Error("failed to get booking", "booking_id", bookingID, "error", err)
		}
		return fmt.Errorf("failed to get booking: %w", err)
	}

	if schemaBooking == nil {
		return domain.ErrBookingNotFound
	}

	booking := domain.MapBookingFromSchema(schemaBooking)

	// Only mark as settled if currently completed
	if booking.Status != domain.BookingStatusCompleted {
		if s.log != nil {
			s.log.Warn("booking cannot be settled", "booking_id", bookingID, "status", booking.Status)
		}
		// Don't fail - just log warning (payout already succeeded)
		return nil
	}

	// Update status to settled
	booking.Status = domain.BookingStatusSettled
	booking.UpdatedAt = time.Now()

	// Save updated booking
	if err := s.repo.UpdateBooking(ctx, domain.MapBookingFromDomain(booking)); err != nil {
		if s.log != nil {
			s.log.Error("failed to update booking status to settled", "booking_id", bookingID, "error", err)
		}
		return fmt.Errorf("failed to update booking: %w", err)
	}

	if s.log != nil {
		s.log.Info(" booking marked as settled", "booking_id", bookingID)
	}

	return nil
}
