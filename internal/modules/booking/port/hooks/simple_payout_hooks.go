package hooks

import (
	"context"
	"hauslet/internal/modules/booking/domain"
	"hauslet/internal/modules/booking/repository"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// SimplePayoutHooksAdapter provides a lightweight payout hooks implementation
// for the worker environment where a full booking service isn't available.
// This adapter updates the booking status directly via the repository.
type SimplePayoutHooksAdapter struct {
	repo repository.BookingRepository
	log  *slog.Logger
}

// NewSimplePayoutHooksAdapter creates a lightweight payout hooks adapter.
func NewSimplePayoutHooksAdapter(repo repository.BookingRepository, log *slog.Logger) *SimplePayoutHooksAdapter {
	return &SimplePayoutHooksAdapter{
		repo: repo,
		log:  log,
	}
}

// MarkAsSettled marks a booking as settled after payout completes.
// This is a lightweight implementation that updates the repository directly.
func (a *SimplePayoutHooksAdapter) MarkAsSettled(ctx context.Context, bookingID uuid.UUID) error {
	if a.log != nil {
		a.log.Info("marking booking as settled", "booking_id", bookingID)
	}

	// Get the booking
	schemaBooking, err := a.repo.GetBookingByID(ctx, bookingID)
	if err != nil {
		if a.log != nil {
			a.log.Error("failed to get booking", "booking_id", bookingID, "error", err)
		}
		return err
	}

	if schemaBooking == nil {
		if a.log != nil {
			a.log.Warn("booking not found", "booking_id", bookingID)
		}
		return domain.ErrBookingNotFound
	}

	booking := domain.MapBookingFromSchema(schemaBooking)

	// Only mark as settled if currently completed
	if booking.Status != domain.BookingStatusCompleted {
		if a.log != nil {
			a.log.Warn("booking cannot be settled", "booking_id", bookingID, "status", booking.Status, "expected_status", "completed")
		}
		// Don't fail - just log warning (payout already succeeded)
		return nil
	}

	// Update status to settled
	booking.Status = domain.BookingStatusSettled
	booking.UpdatedAt = time.Now()

	// Save updated booking
	if err := a.repo.UpdateBooking(ctx, domain.MapBookingFromDomain(booking)); err != nil {
		if a.log != nil {
			a.log.Error("failed to update booking status to settled", "error", err)
		}
		return err
	}

	if a.log != nil {
		a.log.Info("booking marked as settled", "booking_id", bookingID)
	}

	return nil
}
