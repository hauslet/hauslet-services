package service

import (
	"context"
	"fmt"
	"hauslet/internal/modules/payments/domain"

	"github.com/google/uuid"
)

// GetTransaction retrieves a transaction by ID
func (s *PaymentServiceImpl) GetTransaction(ctx context.Context, id uuid.UUID) (*domain.Transaction, error) {
	s.log.Info("fetching transaction", "id", id)

	schemaTx, err := s.repo.GetTransactionByID(ctx, id)
	if err != nil {
		s.log.Error("failed to get transaction", "id", id, "error", err)
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}

	if schemaTx == nil {
		return nil, domain.ErrTransactionNotFound
	}

	return domain.MapTransactionFromSchema(schemaTx), nil
}

// GetTransactionByProviderTxID retrieves a transaction by provider transaction ID
func (s *PaymentServiceImpl) GetTransactionByProviderTxID(ctx context.Context, providerTxID string) (*domain.Transaction, error) {
	s.log.Info("fetching transaction by provider tx id", "provider_tx_id", providerTxID)

	schemaTx, err := s.repo.GetTransactionByProviderTxID(ctx, providerTxID)
	if err != nil {
		s.log.Error("failed to get transaction by provider tx id", "provider_tx_id", providerTxID, "error", err)
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}

	if schemaTx == nil {
		return nil, domain.ErrTransactionNotFound
	}

	return domain.MapTransactionFromSchema(schemaTx), nil
}

// UpdateTransaction updates a transaction
func (s *PaymentServiceImpl) UpdateTransaction(ctx context.Context, tx *domain.Transaction) error {
	s.log.Info("updating transaction", "id", tx.ID, "status", tx.Status)

	schemaTx := domain.MapTransactionToSchema(tx)
	if err := s.repo.UpdateTransaction(ctx, schemaTx); err != nil {
		s.log.Error("failed to update transaction", "id", tx.ID, "error", err)
		return fmt.Errorf("failed to update transaction: %w", err)
	}
	return nil
}

// ListTransactionsByPayment lists transactions for a payment
func (s *PaymentServiceImpl) ListTransactionsByPayment(ctx context.Context, paymentID uuid.UUID) ([]domain.Transaction, error) {
	s.log.Info("listing transactions for payment", "payment_id", paymentID)

	schemaTxs, err := s.repo.ListTransactionsByPaymentID(ctx, paymentID)
	if err != nil {
		s.log.Error("failed to list transactions for payment", "payment_id", paymentID, "error", err)
		return nil, fmt.Errorf("failed to list transactions: %w", err)
	}

	return domain.MapTransactionsFromSchema(schemaTxs), nil
}

// ListTransactionsByBooking lists transactions for a booking
func (s *PaymentServiceImpl) ListTransactionsByBooking(ctx context.Context, bookingID uuid.UUID) ([]domain.Transaction, error) {
	s.log.Info("listing transactions for booking", "booking_id", bookingID)

	schemaTxs, err := s.repo.ListTransactionsByBookingID(ctx, bookingID)
	if err != nil {
		s.log.Error("failed to list transactions for booking", "booking_id", bookingID, "error", err)
		return nil, fmt.Errorf("failed to list transactions: %w", err)
	}

	return domain.MapTransactionsFromSchema(schemaTxs), nil
}
