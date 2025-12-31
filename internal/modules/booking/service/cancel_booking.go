package service

import (
	"context"
	"fmt"
	"hauslet/internal/modules/booking/domain"
	pricingdomain "hauslet/internal/modules/pricing/domain"
	bookingJobs "hauslet/internal/queue/jobs/booking"
	"time"

	"github.com/google/uuid"
)

func (s *BookingServiceImpl) CancelBooking(ctx context.Context, bookingID uuid.UUID, actorID uuid.UUID, reason *string) (*domain.Booking, error) {
	if s.log != nil {
		s.log.Info("cancelling booking", "booking_id", bookingID.String(), "actor_id", actorID.String())
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
			s.log.Warn("failed to get listing constraints", "error", err)
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
				s.log.Warn("failed to calculate refund", "error", err)
			}
		} else {
			// Convert refund amount to minor units
			refundAmount = int64(refundBreakdown.NetRefund * float64(currencyMinorUnit))

			if s.log != nil {
				s.log.Info("refund calculated",
					"booking_id", bookingID.String(),
					"amount", refundBreakdown.NetRefund,
					"currency", booking.Currency,
					"policy", refundBreakdown.AppliedPolicy,
					"cancelled_by", cancelledBy)
			}
		}
	}

	// Process refund if amount > 0 and payment gateway is available
	var refundReference *string
	var refundProcessedAt *time.Time
	refundQueued := false

	if refundAmount > 0 && booking.LastPaymentID != nil && s.payment != nil {
		refundReason := s.buildRefundReason(cancelledBy, reason, refundBreakdown)

		refundInput := RefundPaymentInput{
			PaymentID:  *booking.LastPaymentID,
			Amount:     &refundAmount,
			Reason:     refundReason,
			RefundedBy: actorID,
		}

		if s.refundQueue != nil && s.refundSubject != "" {
			job := bookingJobs.BookingRefundJob{
				BookingID:   booking.ID,
				PaymentID:   *booking.LastPaymentID,
				Amount:      refundAmount,
				Reason:      refundReason,
				RefundedBy:  actorID,
				RequestedAt: time.Now(),
			}

			if err := s.refundQueue.Publish(ctx, s.refundSubject, job); err != nil {
				if s.log != nil {
					s.log.Warn("failed to queue refund job", "booking_id", bookingID.String(), "error", err)
				}
			} else {
				refundQueued = true
				if s.log != nil {
					s.log.Info("refund job queued", "booking_id", bookingID.String(), "payment_id", booking.LastPaymentID.String(), "amount", refundAmount)
				}
			}
		}

		if !refundQueued {
			if s.log != nil {
				s.log.Info("initiating refund", "booking_id", bookingID.String(), "payment_id", booking.LastPaymentID.String(), "amount", refundAmount)
			}

			refundResult, err := s.payment.RefundPayment(ctx, refundInput)
			if err != nil {
				if s.log != nil {
					s.log.Error("failed to process refund", "error", err)
				}
				// Don't fail the cancellation if refund fails - log and continue
				// The refund can be processed manually or retried later
			} else {
				refundReference = &refundResult.RefundID
				if !refundResult.RefundedAt.IsZero() {
					refundProcessedAt = &refundResult.RefundedAt
				} else if !refundResult.ProcessedAt.IsZero() {
					refundProcessedAt = &refundResult.ProcessedAt
				}
				if s.log != nil {
					s.log.Info("refund initiated successfully", "booking_id", bookingID.String(), "refund_id", refundResult.RefundID)
				}
			}
		}
	}

	// Cancel calendar events
	if err := s.calendar.CancelEvent(ctx, booking.CalendarEventID, ownerID); err != nil {
		return nil, err
	}
	if booking.CleaningEventID != nil {
		if err := s.calendar.CancelEvent(ctx, *booking.CleaningEventID, ownerID); err != nil && s.log != nil {
			s.log.Warn("failed to cancel cleaning buffer event for booking", "booking_id", booking.ID, "error", err)
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
		if refundProcessedAt != nil {
			booking.RefundProcessedAt = refundProcessedAt
		}
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
		s.log.Info("booking cancelled successfully", "booking_id", bookingID.String(), "refund_amount", refundAmount, "cancelled_by", cancelledBy)
	}

	return booking, nil
}

// determineCancellationActor determines who initiated the cancellation
func (s *BookingServiceImpl) determineCancellationActor(booking *domain.Booking, actorID uuid.UUID, ownerID uuid.UUID) pricingdomain.CancellationActor {
	switch actorID {
	case booking.GuestID:
		return pricingdomain.CancelledByGuest
	case ownerID:
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
