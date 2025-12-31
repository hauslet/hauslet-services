package service

import (
	"context"
	"fmt"
	"hauslet/internal/modules/payments/domain"
	"time"

	"github.com/google/uuid"
)

// RefundPayment processes a refund for a payment
func (s *PaymentServiceImpl) RefundPayment(ctx context.Context, input domain.RefundPaymentInput) (*domain.Payment, error) {
	s.log.Info("processing refund for payment", "payment_id", input.PaymentID)

	// Validate input
	if err := input.Validate(); err != nil {
		s.log.Error("refund validation failed", "error", err)
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Get payment
	pmt, err := s.GetPayment(ctx, input.PaymentID)
	if err != nil {
		return nil, err
	}

	// Check if payment can be refunded
	if !pmt.CanRefund() {
		s.log.Warn("payment cannot be refunded", "payment_id", input.PaymentID, "status", pmt.Status, "refunded", pmt.RefundedAmount, "amount", pmt.Amount)
		return nil, domain.ErrCannotRefundPayment
	}

	// Determine refund amount
	refundAmount := pmt.Amount - pmt.RefundedAmount
	if input.Amount != nil {
		refundAmount = *input.Amount
		if refundAmount > pmt.RemainingRefundableAmount() {
			return nil, domain.ErrRefundAmountExceeded
		}
	}

	s.log.Info(" initiating refund", "payment", pmt.ID, "amount", refundAmount)

	// Process refund with provider
	refundResp, err := s.paymentClient.Refund(
		ctx,
		pmt.Currency,
		pmt.ProviderRef,
		refundAmount,
		input.Reason,
	)
	if err != nil {
		s.log.Error("refund failed", "payment_id", pmt.ID, "error", err)
		return nil, fmt.Errorf("refund failed: %w", err)
	}

	// Update payment with refund
	pmt.RefundedAmount += refundAmount
	if pmt.IsFullyRefunded() {
		pmt.Status = domain.PaymentStatusRefunded
	}
	now := time.Now()
	pmt.RefundedAt = &now
	pmt.UpdatedAt = now

	// Save updated payment
	if err := s.repo.UpdatePayment(ctx, domain.MapPaymentToSchema(pmt)); err != nil {
		s.log.Error("failed to update payment after refund", "error", err)
		return nil, fmt.Errorf("failed to update payment: %w", err)
	}

	// Create refund transaction record
	tx := &domain.Transaction{
		ID:           uuid.New(),
		PaymentID:    &pmt.ID,
		BookingID:    pmt.BookingID,
		BusinessID:   pmt.BusinessID,
		Type:         domain.TransactionTypeRefund,
		Reference:    fmt.Sprintf("RFD-%s-%s", pmt.Reference, uuid.New().String()[:8]),
		Amount:       refundAmount,
		Currency:     pmt.Currency,
		Status:       domain.TransactionStatus(refundResp.Status),
		Provider:     pmt.Provider,
		ProviderTxID: &refundResp.RefundID,
		Description:  fmt.Sprintf("Refund: %s", input.Reason),
		Metadata: map[string]string{
			"refunded_by":     input.RefundedBy.String(),
			"refund_reason":   input.Reason,
			"original_amount": fmt.Sprintf("%d", pmt.Amount),
		},
		ProcessedAt: &now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.repo.CreateTransaction(ctx, domain.MapTransactionToSchema(tx)); err != nil {
		s.log.Warn("failed to create refund transaction", "error", err)
		// Non-critical
	}

	s.log.Info(" refund processed successfully", "payment", pmt.ID, "refund_amount", refundAmount)

	// Send refund notification
	s.notificationSvc.SendRefundNotification(pmt, refundAmount)

	return pmt, nil
}
