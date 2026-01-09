package http

import (
	"context"
	"fmt"
	paymentdomain "hauslet/internal/modules/payments/domain"
	"hauslet/internal/platform/payment"
)

// ProcessEvent handles a parsed webhook event.
func (h *WebhookHandler) ProcessEvent(ctx context.Context, event *payment.UnifiedEvent) error {
	if event == nil {
		return fmt.Errorf("event is nil")
	}

	switch event.Type {
	case "charge.success":
		return h.handleChargeSuccess(ctx, event)
	case "charge.failed":
		return h.handleChargeFailed(ctx, event)
	case "transfer.success":
		return h.handleTransferSuccess(ctx, event)
	case "transfer.failed":
		return h.handleTransferFailed(ctx, event)
	case "refund.processed":
		return h.handleRefundProcessed(ctx, event)
	case "refund.failed":
		return h.handleRefundFailed(ctx, event)
	default:
		if h.log != nil {
			h.log.Info(" unhandled webhook event type", "type", event.Type)
		}
		return nil
	}
}

// handleChargeSuccess handles successful payment webhook
func (h *WebhookHandler) handleChargeSuccess(ctx context.Context, event *payment.UnifiedEvent) error {
	h.log.Info(" handling charge.success for ref", "ref", event.Reference)

	// Verify payment status
	pmt, err := h.paymentService.VerifyPayment(ctx, event.Reference)
	if err != nil {
		return fmt.Errorf("failed to verify payment: %w", err)
	}

	h.log.Info(" payment verified", "id", pmt.ID, "status", pmt.Status)

	// Extract and save authorization code from webhook for card tokenization
	if err := h.extractAndSaveAuthorization(ctx, event.RawData, pmt); err != nil {
		h.log.Warn("failed to extract authorization code", "error", err)
		// Don't fail the webhook - payment already succeeded
	}

	// If payment is for a booking, record in finance ledger FIRST, then notify booking
	if pmt.BookingID != nil {
		// Finance records transaction FIRST (idempotent)
		if h.financeHooks != nil {
			h.log.Info(" recording charge in finance", "booking", *pmt.BookingID, "amount", pmt.Amount)

			if err := h.financeHooks.OnPaymentSucceeded(ctx, *pmt.BookingID, pmt.ID, pmt.Amount, string(pmt.Currency)); err != nil {
				h.log.Error("failed to record charge in finance", "error", err)
			}
		}

		// Then update booking status
		if h.bookingHooks != nil {
			h.log.Info(" notifying booking module of payment success", "booking", *pmt.BookingID)

			if err := h.bookingHooks.OnPaymentSucceeded(ctx, *pmt.BookingID, pmt); err != nil {
				h.log.Error("failed to notify booking of payment success", "error", err)
			}
		}
	}

	// If payment is for a promotion, activate it
	if pmt.ResourceType == paymentdomain.ResourceTypePromotion && pmt.ResourceID != nil {
		if h.promotionHooks != nil {
			h.log.Info(" activating promotion after payment success", "promotion", *pmt.ResourceID)
			if err := h.promotionHooks.OnPromotionPaymentSucceeded(ctx, *pmt.ResourceID, pmt); err != nil {
				h.log.Error("failed to activate promotion", "error", err)
			}
		} else {
			h.log.Warn("promotion hooks not configured", "promotion", *pmt.ResourceID)
		}
	}

	// If payment is for a subscription, activate it
	if pmt.ResourceType == paymentdomain.ResourceTypeSubscription && pmt.ResourceID != nil {
		if h.promotionHooks != nil {
			h.log.Info(" activating subscription after payment success", "subscription", *pmt.ResourceID)
			if err := h.promotionHooks.OnSubscriptionPaymentSucceeded(ctx, *pmt.ResourceID, pmt); err != nil {
				h.log.Error("failed to activate subscription", "error", err)
			}
		} else {
			h.log.Warn("promotion hooks not configured", "subscription", *pmt.ResourceID)
		}
	}

	return nil
}

// handleChargeFailed handles failed payment webhook
func (h *WebhookHandler) handleChargeFailed(ctx context.Context, event *payment.UnifiedEvent) error {
	h.log.Warn("handling charge.failed for ref", "ref", event.Reference)

	pmt, err := h.paymentService.GetPaymentByReference(ctx, event.Reference)
	if err != nil {
		h.log.Error("failed to get payment by reference", "error", err)
		return fmt.Errorf("failed to get payment: %w", err)
	}

	h.log.Warn("payment failed", "id", pmt.ID, "ref", event.Reference)

	if pmt.BookingID != nil && h.bookingHooks != nil {
		h.log.Info(" notifying booking module of payment failure", "booking", *pmt.BookingID)
		reason := fmt.Sprintf("Payment charge failed (status: %s)", event.Status)

		if err := h.bookingHooks.OnPaymentFailed(ctx, *pmt.BookingID, pmt, reason); err != nil {
			h.log.Error("failed to notify booking of payment failure", "error", err)
		}
	}
	return nil
}

// handleTransferSuccess handles successful payout webhook
func (h *WebhookHandler) handleTransferSuccess(ctx context.Context, event *payment.UnifiedEvent) error {
	h.log.Info(" handling transfer.success for ref", "ref", event.Reference)

	if h.payoutHooks != nil {
		transferCode := event.Reference
		if event.ProviderTxID != "" {
			transferCode = event.ProviderTxID
		}

		h.log.Info(" notifying payout service of transfer success", "ref", transferCode)

		if err := h.payoutHooks.OnTransferSuccess(ctx, transferCode); err != nil {
			h.log.Error("failed to update disbursement status", "error", err)
			return nil
		}
		h.log.Info(" disbursement marked as completed", "ref", transferCode)
	} else {
		h.log.Warn("payout hooks not configured, transfer success not processed")
	}
	return nil
}

// handleTransferFailed handles failed payout webhook
func (h *WebhookHandler) handleTransferFailed(ctx context.Context, event *payment.UnifiedEvent) error {
	h.log.Warn("handling transfer.failed for ref", "ref", event.Reference)

	if h.payoutHooks != nil {
		transferCode := event.Reference
		if event.ProviderTxID != "" {
			transferCode = event.ProviderTxID
		}

		reason := "Transfer failed"
		if event.Status != "" {
			reason = fmt.Sprintf("Transfer failed: %s", event.Status)
		}

		h.log.Warn("notifying payout service of transfer failure", "ref", transferCode, "reason", reason)

		if err := h.payoutHooks.OnTransferFailed(ctx, transferCode, reason); err != nil {
			h.log.Error("failed to update disbursement failure status", "error", err)
			return nil
		}
		h.log.Info(" disbursement marked as failed with retry scheduled", "ref", transferCode)
	} else {
		h.log.Warn("payout hooks not configured, transfer failure not processed")
	}
	return nil
}

// handleRefundProcessed handles successful refund webhook
func (h *WebhookHandler) handleRefundProcessed(ctx context.Context, event *payment.UnifiedEvent) error {
	h.log.Info(" handling refund.processed for ref", "ref", event.Reference)

	pmt, err := h.paymentService.GetPaymentByReference(ctx, event.Reference)
	if err != nil {
		return fmt.Errorf("failed to get payment by reference: %w", err)
	}

	verifiedPmt, err := h.paymentService.VerifyPayment(ctx, event.Reference)
	if err != nil {
		return fmt.Errorf("failed to verify payment after refund: %w", err)
	}

	h.log.Info("refund processed", "payment_id", pmt.ID, "refunded_amount", verifiedPmt.RefundedAmount, "status", verifiedPmt.Status)

	if verifiedPmt.BookingID != nil {
		if h.financeHooks != nil {
			h.log.Info(" recording refund in finance", "booking", *verifiedPmt.BookingID, "amount", verifiedPmt.RefundedAmount)
			if err := h.financeHooks.OnRefundProcessed(ctx, *verifiedPmt.BookingID, verifiedPmt.ID, verifiedPmt.RefundedAmount, string(verifiedPmt.Currency)); err != nil {
				h.log.Error("failed to record refund in finance", "error", err)
			}
		}

		if h.bookingHooks != nil {
			h.log.Info(" notifying booking module of refund", "booking", *verifiedPmt.BookingID)
			if err := h.bookingHooks.OnPaymentRefunded(ctx, *verifiedPmt.BookingID, verifiedPmt); err != nil {
				h.log.Error("failed to notify booking of refund", "error", err)
			}
		}
	}
	return nil
}

// handleRefundFailed handles failed refund webhook
func (h *WebhookHandler) handleRefundFailed(ctx context.Context, event *payment.UnifiedEvent) error {
	h.log.WarnContext(ctx, "handling refund.failed", "ref", event.Reference)

	pmt, err := h.paymentService.GetPaymentByReference(ctx, event.Reference)
	if err != nil {
		return fmt.Errorf("failed to get payment by reference: %w", err)
	}

	h.log.Warn("refund failed", "payment_id", pmt.ID, "ref", event.Reference, "provider_tx", event.ProviderTxID)

	// TODO: Create a failed transaction record and notify business/admin
	// For now, just log the failure
	// In a production system, you might want to:
	// 1. Create a failed transaction record
	// 2. Send an alert to the business owner
	// 3. Queue for manual review
	return nil
}
