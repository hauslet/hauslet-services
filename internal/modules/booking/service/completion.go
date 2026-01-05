package service

import (
	"context"
	"fmt"
	"hauslet/config"
	"hauslet/internal/modules/booking/repository/schema"
	"time"
)

// CompleteBookings finds active bookings ready to be marked as completed and processes them
// This should be called by a cron job (e.g., every hour)
func (s *BookingServiceImpl) CompleteBookings(ctx context.Context) error {
	if s.log != nil {
		s.log.Info(
			"[AUDIT] booking_completion_batch_started",
			"time", time.Now().Format(time.RFC3339),
		)
	}

	// Get escrow release config
	escrowReleaseHours := s.platformConfig.Payouts.EscrowReleaseHours
	if escrowReleaseHours == 0 {
		escrowReleaseHours = 24 // Default to 24 hours
	}

	// Keep typed for validation and type safety
	escrowReleaseEvent := s.platformConfig.Payouts.EscrowReleaseEvent
	if !escrowReleaseEvent.IsValid() {
		escrowReleaseEvent = config.EscrowReleaseCheckoutConfirmed // Type-safe default
	}

	if s.log != nil {
		s.log.Info(
			"[AUDIT] completion_config",
			"escrow_release_event", escrowReleaseEvent,
			"escrow_release_hours", escrowReleaseHours,
		)
	}

	// Convert to string only when passing to the query
	bookings, err := s.repo.FindBookingsReadyForCompletion(
		ctx,
		escrowReleaseEvent.String(),
		escrowReleaseHours,
		100,
	)
	if err != nil {
		if s.log != nil {
			s.log.Error(
				"failed to query bookings ready for completion",
				"error", err,
			)
		}
		return fmt.Errorf("failed to query bookings: %w", err)
	}

	// Nothing to process
	if len(bookings) == 0 {
		if s.log != nil {
			s.log.Info("[AUDIT] no bookings ready for completion")
		}
		return nil
	}

	successCount := 0
	failureCount := 0

	if s.log != nil {
		s.log.Info(
			"[AUDIT] found bookings ready for completion",
			"count", len(bookings),
		)
	}

	// Process each booking
	for _, booking := range bookings {
		err := s.completeSingleBooking(ctx, booking)
		if err != nil {
			if s.log != nil {
				s.log.Error(
					"[AUDIT] completion_failed",
					"booking_id", booking.ID,
					"error", err,
				)
			}
			failureCount++
			continue
		}

		if s.log != nil {
			s.log.Info(
				"[AUDIT] completion_success",
				"booking_id", booking.ID,
			)
		}
		successCount++
	}

	if s.log != nil {
		s.log.Info(
			"[AUDIT] booking_completion_batch_completed",
			"time", time.Now().Format(time.RFC3339),
			"total", len(bookings),
			"success", successCount,
			"failed", failureCount,
		)
	}

	return nil
}

// completeSingleBooking marks a single booking as completed
func (s *BookingServiceImpl) completeSingleBooking(ctx context.Context, booking *schema.Booking) error {
	if s.log != nil {
		s.log.Info("[AUDIT] completing_booking", "booking_id", booking.ID, "listing_id", booking.ListingID)
	}

	// Update booking status to completed
	now := time.Now()
	err := s.repo.UpdateStatus(ctx, booking.ID, schema.BookingStatusCompleted, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to update booking status: %w", err)
	}

	// Update completed_at timestamp
	booking.Status = schema.BookingStatusCompleted
	booking.CompletedAt = &now
	if err := s.repo.UpdateBooking(ctx, booking); err != nil {
		return fmt.Errorf("failed to update booking completed_at: %w", err)
	}

	if s.log != nil {
		s.log.Info("[AUDIT] booking marked as completed", "booking_id", booking.ID)
	}

	// TODO: Add post-completion actions here
	// In the future, additional post-completion actions would be added here
	// Like sending reviews requests, updating host stats, etc. Emmanuel I.

	// Notify finance module that booking is completed
	// This triggers the payout eligibility check
	if s.financeHooks != nil {
		// Get host ID from listing
		hostID, err := s.listingHooks.GetListingOwner(ctx, booking.ListingID)
		if err != nil {
			// Log but don't fail - finance hook is not critical for booking status
			if s.log != nil {
				s.log.Warn("failed to get listing owner for finance hook", "error", err)
			}
		} else {
			if err := s.financeHooks.OnBookingCompleted(ctx, booking.ID, hostID); err != nil {
				// Log but don't fail - finance hook failure shouldn't prevent booking completion
				if s.log != nil {
					s.log.Warn("finance hook failed for booking", "booking_id", booking.ID, "error", err)
				}
			}
		}
	}

	// Send review invites to guest and host (if not already sent)
	if s.reviewHooks != nil && booking.ReviewInviteSentAt == nil {
		if err := s.reviewHooks.SendReviewInvites(ctx, booking.ID); err != nil {
			if s.log != nil {
				s.log.Warn("failed to send review invites for booking", "booking_id", booking.ID, "error", err)
			}
		} else {
			now := time.Now()
			booking.ReviewInviteSentAt = &now
			booking.UpdatedAt = now
			if err := s.repo.UpdateBooking(ctx, booking); err != nil {
				if s.log != nil {
					s.log.Warn("failed to update review invite timestamp for booking", "booking_id", booking.ID, "error", err)
				}
			}
		}
	}

	return nil
}
