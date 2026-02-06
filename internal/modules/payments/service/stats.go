package service

import (
	"context"
	"fmt"
	"hauslet/internal/modules/payments/domain"

	"github.com/google/uuid"
)

func (s *PaymentServiceImpl) GetPaymentStats(ctx context.Context, payerID uuid.UUID) (*domain.PaymentStats, error) {
	stats, err := s.repo.GetPaymentStats(ctx, payerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get payment stats: %w", err)
	}

	return &domain.PaymentStats{
		TotalSpent:       stats.TotalSpent,
		UpcomingPayments: stats.UpcomingPayments,
		TotalRefunds:     stats.TotalRefunds,
	}, nil
}
