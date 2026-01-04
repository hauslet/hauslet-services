package service

import (
	"context"
	"fmt"
	"hauslet/internal/modules/finance/domain"
	"sort"

	"github.com/google/uuid"
)

// ListDisbursementsByOwner lists disbursements for a user or business.
func (s *PayoutServiceImpl) ListDisbursementsByOwner(
	ctx context.Context,
	ownerType domain.OwnerType,
	ownerID uuid.UUID,
	status *domain.DisbursementStatus,
	limit, offset int,
) ([]*domain.Disbursement, error) {
	if ownerID == uuid.Nil {
		return nil, domain.ErrInvalidOwner
	}

	wallets, err := s.walletRepo.ListByOwner(ctx, ownerType.String(), ownerID)
	if err != nil {
		return nil, fmt.Errorf("failed to list wallets: %w", err)
	}

	disbursements := make([]*domain.Disbursement, 0)
	for _, wallet := range wallets {
		items, err := s.disbursementRepo.ListByWallet(ctx, wallet.ID, 0, 0)
		if err != nil {
			return nil, fmt.Errorf("failed to list disbursements: %w", err)
		}
		for _, item := range items {
			disbursement := domain.MapDisbursementFromSchema(item)
			if status != nil && disbursement.Status != *status {
				continue
			}
			disbursements = append(disbursements, disbursement)
		}
	}

	sort.Slice(disbursements, func(i, j int) bool {
		return disbursements[i].CreatedAt.After(disbursements[j].CreatedAt)
	})

	if offset >= len(disbursements) {
		return []*domain.Disbursement{}, nil
	}

	end := len(disbursements)
	if limit > 0 && offset+limit < end {
		end = offset + limit
	}

	return disbursements[offset:end], nil
}
