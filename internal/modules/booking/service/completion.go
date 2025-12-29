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
		s.log.Logf("INFO [AUDIT] booking_completion_batch_started time=%s", time.Now().Format(time.RFC3339))
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
		s.log.Logf("INFO [AUDIT] completion_config escrow_release_event=%s escrow_release_hours=%d",
			escrowReleaseEvent, escrowReleaseHours)
	}

	// Convert to string only when passing to the query
	bookings, err := s.repo.FindBookingsReadyForCompletion(ctx, escrowReleaseEvent.String(), escrowReleaseHours, 100)
	if err != nil {
		if s.log != nil {
			s.log.Logf("ERROR failed to query bookings ready for completion: %v", err)
		}
		return fmt.Errorf("failed to query bookings: %w", err)
	}

	successCount := 0
	failureCount := 0

	if s.log != nil {
		s.log.Logf("INFO [AUDIT] found %d bookings ready for completion", len(bookings))
	}

	// Process each booking
	for _, booking := range bookings {
		err := s.completeSingleBooking(ctx, booking)
		if err != nil {
			if s.log != nil {
				s.log.Logf("ERROR [AUDIT] completion_failed booking_id=%s error=%v",
					booking.ID, err)
			}
			failureCount++
		} else {
			if s.log != nil {
				s.log.Logf("INFO [AUDIT] completion_success booking_id=%s",
					booking.ID)
			}
			successCount++
		}
	}

	if s.log != nil {
		s.log.Logf("INFO [AUDIT] booking_completion_batch_completed time=%s total=%d success=%d failed=%d",
			time.Now().Format(time.RFC3339), len(bookings), successCount, failureCount)
	}

	return nil
}

// completeSingleBooking marks a single booking as completed
func (s *BookingServiceImpl) completeSingleBooking(ctx context.Context, booking *schema.Booking) error {
	if s.log != nil {
		s.log.Logf("INFO [AUDIT] completing_booking booking_id=%s listing_id=%s",
			booking.ID, booking.ListingID)
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
		s.log.Logf("INFO booking marked as completed: booking_id=%s", booking.ID)
	}

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
				s.log.Logf("WARN failed to get listing owner for finance hook: %v", err)
			}
		} else {
			if err := s.financeHooks.OnBookingCompleted(ctx, booking.ID, hostID); err != nil {
				// Log but don't fail - finance hook failure shouldn't prevent booking completion
				if s.log != nil {
					s.log.Logf("WARN finance hook failed for booking_id=%s: %v", booking.ID, err)
				}
			}
		}
	}

	return nil
}
