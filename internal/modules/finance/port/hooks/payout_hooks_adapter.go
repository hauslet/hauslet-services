package hooks

import (
	"context"
	"hauslet/internal/modules/finance/domain"
	"hauslet/internal/modules/finance/service"
)

// PayoutHooksAdapter adapts the payout service to implement payment webhook PayoutHooks interface
type PayoutHooksAdapter struct {
	payoutSvc service.PayoutService
}

// NewPayoutHooksAdapter creates a new payout hooks adapter for transfer webhooks
func NewPayoutHooksAdapter(svc service.PayoutService) *PayoutHooksAdapter {
	return &PayoutHooksAdapter{
		payoutSvc: svc,
	}
}

// OnTransferSuccess handles successful transfer webhook
func (a *PayoutHooksAdapter) OnTransferSuccess(ctx context.Context, transferCode string) error {
	// Get disbursement by transfer code
	disbursement, err := a.payoutSvc.GetDisbursementByTransferCode(ctx, transferCode)
	if err != nil {
		return err
	}

	// Update status to completed
	return a.payoutSvc.UpdateDisbursementStatus(ctx, disbursement.ID, domain.DisbursementStatusCompleted, nil)
}

// OnTransferFailed handles failed transfer webhook
func (a *PayoutHooksAdapter) OnTransferFailed(ctx context.Context, transferCode string, reason string) error {
	// Get disbursement by transfer code
	disbursement, err := a.payoutSvc.GetDisbursementByTransferCode(ctx, transferCode)
	if err != nil {
		return err
	}

	// Update status to failed with reason
	return a.payoutSvc.UpdateDisbursementStatus(ctx, disbursement.ID, domain.DisbursementStatusFailed, &reason)
}
