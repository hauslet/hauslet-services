package hooks

import (
	"context"
	"hauslet/internal/modules/finance/service"

	"github.com/google/uuid"
)

// PaymentHooksAdapter adapts the finance service to implement payment webhook FinanceHooks interface
type PaymentHooksAdapter struct {
	financeSvc service.FinanceService
}

// NewPaymentHooksAdapter creates a new finance hooks adapter for payment webhooks
func NewPaymentHooksAdapter(svc service.FinanceService) *PaymentHooksAdapter {
	return &PaymentHooksAdapter{
		financeSvc: svc,
	}
}

// OnPaymentSucceeded records a charge in the finance ledger
func (a *PaymentHooksAdapter) OnPaymentSucceeded(
	ctx context.Context,
	bookingID, paymentID uuid.UUID,
	amount int64,
	currency string,
) error {
	_, err := a.financeSvc.RecordCharge(ctx, bookingID, paymentID, amount, currency)
	return err
}

// OnRefundProcessed records a refund in the finance ledger
func (a *PaymentHooksAdapter) OnRefundProcessed(
	ctx context.Context,
	bookingID, paymentID uuid.UUID,
	amount int64,
	currency string,
) error {
	_, err := a.financeSvc.RecordRefund(ctx, bookingID, paymentID, amount, currency)
	return err
}

// OnBookingCompleted handles booking completion (triggers payout process)
// This is called when a booking is completed and ready for payout
func (a *PaymentHooksAdapter) OnBookingCompleted(
	ctx context.Context,
	bookingID, hostID uuid.UUID,
) error {
	// TODO: Implement payout queueing logic in Phase 4.5
	// This will call payoutService.QueuePayout(ctx, bookingID, hostID, amount, currency)
	// For now, this is a placeholder
	return nil
}
