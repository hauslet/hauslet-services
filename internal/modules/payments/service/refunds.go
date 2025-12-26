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
	s.log.Logf("INFO processing refund for payment=%s", input.PaymentID)

	// Validate input
	if err := input.Validate(); err != nil {
		s.log.Logf("ERROR refund validation failed: %v", err)
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Get payment
	pmt, err := s.GetPayment(ctx, input.PaymentID)
	if err != nil {
		return nil, err
	}

	// Check if payment can be refunded
	if !pmt.CanRefund() {
		s.log.Logf("WARN payment %s cannot be refunded: status=%s, refunded=%d/%d",
			input.PaymentID, pmt.Status, pmt.RefundedAmount, pmt.Amount)
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

	s.log.Logf("INFO initiating refund: payment=%s, amount=%d", pmt.ID, refundAmount)

	// Process refund with provider
	refundResp, err := s.paymentClient.Refund(
		ctx,
		pmt.Currency,
		pmt.ProviderRef,
		refundAmount,
		input.Reason,
	)
	if err != nil {
		s.log.Logf("ERROR refund failed for payment=%s: %v", pmt.ID, err)
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
		s.log.Logf("ERROR failed to update payment after refund: %v", err)
		return nil, fmt.Errorf("failed to update payment: %w", err)
	}

	// Create refund transaction record
	tx := &domain.Transaction{
		ID:         uuid.New(),
		PaymentID:  &pmt.ID,
		BookingID:  pmt.BookingID,
		BusinessID: pmt.BusinessID,
		Type:       domain.TransactionTypeRefund,
		Reference:  fmt.Sprintf("RFD-%s-%s", pmt.Reference, uuid.New().String()[:8]),
		Amount:     refundAmount,
		Currency:   pmt.Currency,
		Status:     domain.TransactionStatus(refundResp.Status),
		Provider:   pmt.Provider,
		ProviderTxID: &refundResp.RefundID,
		Description: fmt.Sprintf("Refund: %s", input.Reason),
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
		s.log.Logf("WARN failed to create refund transaction: %v", err)
		// Non-critical
	}

	s.log.Logf("INFO refund processed successfully: payment=%s, refund_amount=%d", pmt.ID, refundAmount)

	// Send refund notification
	s.notificationSvc.SendRefundNotification(pmt, refundAmount)

	return pmt, nil
}
