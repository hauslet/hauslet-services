package hooks

import (
	"context"

	"github.com/google/uuid"
)

// FinanceHooks defines the interface for finance module callbacks
// Finance hooks are called when booking-related financial events occur
type FinanceHooks interface {
	// OnPaymentSucceeded is called when a booking payment succeeds
	// This records the charge in the finance ledger (guest payment → escrow wallet)
	OnPaymentSucceeded(ctx context.Context, bookingID, paymentID uuid.UUID, amount int64, currency string) error

	// OnRefundProcessed is called when a booking refund is processed
	// This records the refund in the finance ledger (escrow → refund pool)
	OnRefundProcessed(ctx context.Context, bookingID, paymentID uuid.UUID, amount int64, currency string) error

	// OnBookingCompleted is called when a booking is completed (checkout + 48h)
	// This triggers the payout process (commission + host payout)
	OnBookingCompleted(ctx context.Context, bookingID, hostID uuid.UUID) error
}
