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
		a.log.Info(" marking booking %s as settled (worker)", bookingID)
	}

	// Get the booking
	schemaBooking, err := a.repo.GetBookingByID(ctx, bookingID)
	if err != nil {
		if a.log != nil {
			a.log.Error("failed to get booking %s: %v", bookingID, err)
		}
		return err
	}

	if schemaBooking == nil {
		if a.log != nil {
			a.log.Warn("booking %s not found", bookingID)
		}
		return domain.ErrBookingNotFound
	}

	booking := domain.MapBookingFromSchema(schemaBooking)

	// Only mark as settled if currently completed
	if booking.Status != domain.BookingStatusCompleted {
		if a.log != nil {
			a.log.Warn("booking %s cannot be settled (status=%s, expected=completed)", bookingID, booking.Status)
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
			a.log.Error("failed to update booking status to settled: %v", err)
		}
		return err
	}

	if a.log != nil {
		a.log.Info(" booking %s marked as settled (worker)", bookingID)
	}

	return nil
}
