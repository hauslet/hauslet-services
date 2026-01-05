package service

import (
	"context"
	"fmt"
	"hauslet/internal/modules/finance/domain"
	"sort"

	"github.com/google/uuid"
)

// ListTransactionsByOwner returns transactions for a user or business.
func (s *FinanceServiceImpl) ListTransactionsByOwner(
	ctx context.Context,
	ownerType domain.OwnerType,
	ownerID uuid.UUID,
	txType *domain.TransactionType,
	status *domain.TransactionStatus,
	limit, offset int,
) ([]*domain.Transaction, error) {
	if ownerID == uuid.Nil {
		return nil, domain.ErrInvalidOwner
	}

	wallets, err := s.walletRepo.ListByOwner(ctx, ownerType.String(), ownerID)
	if err != nil {
		return nil, fmt.Errorf("failed to list wallets: %w", err)
	}

	transactionIDs := make(map[uuid.UUID]struct{})
	for _, wallet := range wallets {
		entries, err := s.ledgerRepo.ListByWallet(ctx, wallet.ID, 0, 0)
		if err != nil {
			return nil, fmt.Errorf("failed to list ledger entries: %w", err)
		}
		for _, entry := range entries {
			transactionIDs[entry.TransactionID] = struct{}{}
		}
	}

	transactions := make([]*domain.Transaction, 0, len(transactionIDs))
	for id := range transactionIDs {
		schemaTx, err := s.transactionRepo.GetByID(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("failed to get transaction: %w", err)
		}
		if schemaTx == nil {
			continue
		}

		tx := domain.MapTransactionFromSchema(schemaTx)
		if txType != nil && tx.Type != *txType {
			continue
		}
		if status != nil && tx.Status != *status {
			continue
		}
		transactions = append(transactions, tx)
	}

	sort.Slice(transactions, func(i, j int) bool {
		return transactions[i].CreatedAt.After(transactions[j].CreatedAt)
	})

	if offset >= len(transactions) {
		return []*domain.Transaction{}, nil
	}

	end := len(transactions)
	if limit > 0 && offset+limit < end {
		end = offset + limit
	}

	return transactions[offset:end], nil
}
