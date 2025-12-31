package service

import (
	"context"
	"fmt"
	"hauslet/internal/modules/payments/domain"
	"hauslet/internal/platform/payment"
	"time"

	"github.com/google/uuid"
)

// ProcessPayout processes a payout to a recipient
func (s *PaymentServiceImpl) ProcessPayout(ctx context.Context, input domain.ProcessPayoutInput) (*domain.Transaction, error) {
	s.log.Info(" processing payout", "detail", input.PayoutDetailID, "amount", input.Amount)

	// Validate input
	if err := input.Validate(); err != nil {
		s.log.Error("payout validation failed", "error", err)
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Get payout detail
	pd, err := s.GetPayoutDetail(ctx, input.PayoutDetailID)
	if err != nil {
		return nil, err
	}

	// Verify payout detail can receive payouts
	if !pd.CanReceivePayouts() {
		s.log.Warn("payout detail cannot receive payouts", "id", input.PayoutDetailID, "verified", pd.IsVerified, "active", pd.IsActive)
		return nil, domain.ErrBankAccountNotVerified
	}

	// Generate transaction reference
	txID := uuid.New()
	reference := fmt.Sprintf("PAYOUT-%s", txID.String()[:8])

	// Create transaction record
	tx := &domain.Transaction{
		ID:          txID,
		BookingID:   input.BookingID,
		BusinessID:  input.BusinessID,
		Type:        domain.TransactionTypePayout,
		Reference:   reference,
		Amount:      input.Amount,
		Currency:    input.Currency,
		Status:      domain.TransactionStatusPending,
		Provider:    pd.Provider,
		Description: input.Description,
		Metadata:    input.Metadata,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Save transaction first
	if err := s.repo.CreateTransaction(ctx, domain.MapTransactionToSchema(tx)); err != nil {
		s.log.Error("failed to create transaction record", "error", err)
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	// Process payout with provider
	s.log.Info(" initiating payout with provider", "ref", reference)
	payoutResp, err := s.paymentClient.Transfer(ctx, payment.PayoutRequest{
		Amount:        input.Amount,
		Currency:      input.Currency,
		RecipientCode: pd.RecipientCode,
		Reference:     reference,
		Narration:     input.Description,
		Metadata:      input.Metadata,
	})

	if err != nil {
		s.log.Error("payout failed", "error", err)
		// Update transaction status to failed
		tx.Status = domain.TransactionStatusFailed
		errMsg := err.Error()
		tx.ErrorMessage = &errMsg
		if updateErr := s.repo.UpdateTransaction(ctx, domain.MapTransactionToSchema(tx)); updateErr != nil {
			s.log.Warn("failed to update transaction status", "error", updateErr)
		}
		return nil, fmt.Errorf("payout failed: %w", err)
	}

	// Update transaction with provider response
	providerTxID := payoutResp.TransferID
	tx.ProviderTxID = &providerTxID

	switch payoutResp.Status {
	case payment.TransferSuccess:
		tx.Status = domain.TransactionStatusSucceeded
		now := time.Now()
		tx.ProcessedAt = &now
	case payment.TransferPending:
		tx.Status = domain.TransactionStatusPending
	case payment.TransferFailed:
		tx.Status = domain.TransactionStatusFailed
		errMsg := payoutResp.Message
		tx.ErrorMessage = &errMsg
	}

	// Update transaction
	if err := s.repo.UpdateTransaction(ctx, domain.MapTransactionToSchema(tx)); err != nil {
		s.log.Warn("failed to update transaction", "error", err)
		// Non-critical
	}

	s.log.Info(" payout processed", "id", tx.ID, "status", tx.Status)
	// Send notification
	if tx.Status == domain.TransactionStatusSucceeded {
		s.notificationSvc.SendPayoutNotification(tx, pd)
	}

	return tx, nil
}
