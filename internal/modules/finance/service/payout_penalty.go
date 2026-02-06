package service

import (
	"context"
	"fmt"
	"hauslet/internal/modules/finance/domain"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// applyPenaltyDebtsToPayout nets outstanding penalties from the host payout amount.
// Returns the amount applied to penalties.
func (s *PayoutServiceImpl) applyPenaltyDebtsToPayout(
	ctx context.Context,
	tx *gorm.DB,
	hostID uuid.UUID,
	escrowWalletID uuid.UUID,
	platformWalletID uuid.UUID,
	hostPayout int64,
	currency string,
	payoutBookingID uuid.UUID,
) (int64, error) {
	if s.penaltyDebtRepo == nil || hostPayout <= 0 {
		return 0, nil
	}

	debtRepo := s.penaltyDebtRepo.WithTx(tx)
	debts, err := debtRepo.ListOutstandingByHost(ctx, hostID)
	if err != nil {
		return 0, err
	}
	if len(debts) == 0 {
		return 0, nil
	}

	remaining := hostPayout
	applied := int64(0)

	for _, debt := range debts {
		if remaining <= 0 {
			break
		}
		if debt.OutstandingAmount <= 0 {
			continue
		}
		if debt.Currency != "" && currency != "" && debt.Currency != currency {
			if s.log != nil {
				s.log.Warn("penalty debt currency mismatch; skipping",
					"debt_id", debt.ID,
					"debt_currency", debt.Currency,
					"payout_currency", currency,
				)
			}
			continue
		}

		collect := debt.OutstandingAmount
		if collect > remaining {
			collect = remaining
		}
		if collect <= 0 {
			continue
		}

		memo := fmt.Sprintf("Penalty collection for booking %s (netted from payout booking %s)", debt.BookingID, payoutBookingID)
		if _, err := s.recordPenaltyInternal(ctx, tx, debt.BookingID, escrowWalletID, platformWalletID, collect, currency, memo); err != nil {
			return 0, err
		}

		debt.OutstandingAmount -= collect
		if debt.OutstandingAmount < 0 {
			debt.OutstandingAmount = 0
		}
		debt.UpdatedAt = time.Now()
		if debt.OutstandingAmount == 0 {
			debt.Status = string(domain.PenaltyDebtStatusSettled)
			now := time.Now()
			debt.SettledAt = &now
		}

		if err := debtRepo.Update(ctx, debt); err != nil {
			return 0, err
		}

		if debt.OutstandingAmount == 0 {
			if err := markHostCancellationPenaltyPaid(ctx, tx, debt.BookingID); err != nil {
				return 0, err
			}
		}

		remaining -= collect
		applied += collect
	}

	return applied, nil
}
