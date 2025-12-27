package hooks

import (
	"context"
	"fmt"
	"hauslet/internal/modules/booking/service"

	"github.com/google/uuid"
)

// PayoutHooksAdapter provides payout lifecycle hooks for the booking module.
// This adapter is called by the finance/payout module to notify booking
// when a host payout has been successfully processed.
type PayoutHooksAdapter struct {
	svc service.BookingService
}

// NewPayoutHooksAdapter creates a new payout hooks adapter.
func NewPayoutHooksAdapter(svc service.BookingService) *PayoutHooksAdapter {
	return &PayoutHooksAdapter{
		svc: svc,
	}
}

// MarkAsSettled marks a booking as settled after payout completes.
// Called by finance module after successful host payout.
// Transitions booking from completed -> settled status.
func (a *PayoutHooksAdapter) MarkAsSettled(ctx context.Context, bookingID uuid.UUID) error {
	if err := a.svc.MarkAsSettled(ctx, bookingID); err != nil {
		return fmt.Errorf("failed to mark booking as settled: %w", err)
	}

	return nil
}
