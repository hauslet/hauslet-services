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
//
// Note: Actual payout processing is handled by the ProcessDuePayouts cron job,
// which automatically finds bookings ready for payout (completed + 48h window).
// This hook is intentionally a no-op since the cron-based approach provides
// better reliability and retry handling than webhook-triggered payouts.
func (a *PaymentHooksAdapter) OnBookingCompleted(
	ctx context.Context,
	bookingID, hostID uuid.UUID,
) error {
	// Payouts are processed by the ProcessDuePayouts cron job
	// No action needed here - cron will pick up completed bookings automatically
	return nil
}

// OnBookingCancelledWithFunds handles booking cancellation fund settlement
// This is called after a booking is cancelled and refund has been processed
// to distribute any remaining non-refunded funds (commission to platform, remainder to host)
func (a *PaymentHooksAdapter) OnBookingCancelledWithFunds(
	ctx context.Context,
	bookingID, hostID uuid.UUID,
	hostAmount, platformAmount int64,
	currency string,
) error {
	return a.financeSvc.SettleCancelledBooking(ctx, bookingID, hostID, hostAmount, platformAmount, currency)
}
