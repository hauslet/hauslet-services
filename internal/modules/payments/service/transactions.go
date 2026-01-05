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
