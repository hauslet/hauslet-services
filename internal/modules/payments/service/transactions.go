package service

import (
	"context"
	"fmt"
	"hauslet/internal/modules/payments/domain"

	"github.com/google/uuid"
)

// GetTransaction retrieves a transaction by ID
func (s *PaymentServiceImpl) GetTransaction(ctx context.Context, id uuid.UUID) (*domain.Transaction, error) {
	s.log.Logf("INFO fetching transaction: id=%s", id)

	schemaTx, err := s.repo.GetTransactionByID(ctx, id)
	if err != nil {
		s.log.Logf("ERROR failed to get transaction %s: %v", id, err)
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}

	if schemaTx == nil {
		return nil, domain.ErrTransactionNotFound
	}

	return domain.MapTransactionFromSchema(schemaTx), nil
}

// ListTransactionsByPayment lists transactions for a payment
func (s *PaymentServiceImpl) ListTransactionsByPayment(ctx context.Context, paymentID uuid.UUID) ([]domain.Transaction, error) {
	s.log.Logf("INFO listing transactions for payment=%s", paymentID)

	schemaTxs, err := s.repo.ListTransactionsByPaymentID(ctx, paymentID)
	if err != nil {
		s.log.Logf("ERROR failed to list transactions for payment %s: %v", paymentID, err)
		return nil, fmt.Errorf("failed to list transactions: %w", err)
	}

	return domain.MapTransactionsFromSchema(schemaTxs), nil
}

// ListTransactionsByBooking lists transactions for a booking
func (s *PaymentServiceImpl) ListTransactionsByBooking(ctx context.Context, bookingID uuid.UUID) ([]domain.Transaction, error) {
	s.log.Logf("INFO listing transactions for booking=%s", bookingID)

	schemaTxs, err := s.repo.ListTransactionsByBookingID(ctx, bookingID)
	if err != nil {
		s.log.Logf("ERROR failed to list transactions for booking %s: %v", bookingID, err)
		return nil, fmt.Errorf("failed to list transactions: %w", err)
	}

	return domain.MapTransactionsFromSchema(schemaTxs), nil
}
