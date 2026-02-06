package service

import (
	"context"
	"errors"
	"fmt"
	"hauslet/internal/modules/finance/domain"
	"hauslet/internal/modules/finance/repository/schema"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PenaltyDebtCollectionReport summarizes a debt collection run.
type PenaltyDebtCollectionReport struct {
	DebtsProcessed int
	DebtsSettled   int
	AmountCollected int64
}

// CollectOutstandingPenaltyDebts attempts to collect unpaid penalties from host wallets.
func (s *FinanceServiceImpl) CollectOutstandingPenaltyDebts(ctx context.Context) (*PenaltyDebtCollectionReport, error) {
	if s.penaltyDebtRepo == nil {
		return &PenaltyDebtCollectionReport{}, nil
	}

	debts, err := s.penaltyDebtRepo.ListOutstanding(ctx, 200)
	if err != nil {
		return nil, err
	}

	report := &PenaltyDebtCollectionReport{DebtsProcessed: len(debts)}
	for _, debt := range debts {
		collected, settled, err := s.collectPenaltyDebtFromWallet(ctx, debt)
		if err != nil {
			if s.log != nil {
				s.log.Warn("penalty debt collection failed",
					"debt_id", debt.ID,
					"booking_id", debt.BookingID,
					"error", err,
				)
			}
			continue
		}

		report.AmountCollected += collected
		if settled {
			report.DebtsSettled++
		}
	}

	return report, nil
}

func (s *FinanceServiceImpl) collectPenaltyDebtFromWallet(ctx context.Context, debt *schema.PenaltyDebt) (int64, bool, error) {
	if debt == nil {
		return 0, false, nil
	}
	if debt.OutstandingAmount <= 0 || debt.Status == string(domain.PenaltyDebtStatusSettled) {
		return 0, debt.OutstandingAmount <= 0, nil
	}

	hostWallet, err := s.GetOrCreateWallet(ctx, domain.OwnerTypeUser, debt.HostID, domain.WalletTypeHostAvailable, debt.Currency)
	if err != nil {
		return 0, false, err
	}

	if hostWallet.Balance <= 0 {
		return 0, false, nil
	}

	// Ensure platform wallet exists
	platformID := uuid.NewSHA1(uuid.NameSpaceDNS, []byte("hauslet"))
	platformWallet, err := s.GetOrCreateWallet(ctx, domain.OwnerTypePlatform, platformID, domain.WalletTypePlatformFee, debt.Currency)
	if err != nil {
		return 0, false, err
	}

	collectAmount := debt.OutstandingAmount
	if hostWallet.Balance < collectAmount {
		collectAmount = hostWallet.Balance
	}
	if collectAmount <= 0 {
		return 0, false, nil
	}

	var collected int64
	var settled bool
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		debtRepo := s.penaltyDebtRepo.WithTx(tx)
		walletRepo := s.walletRepo.WithTx(tx)
		ledgerRepo := s.ledgerRepo.WithTx(tx)
		txRepo := s.transactionRepo.WithTx(tx)

		lockedHost, err := walletRepo.GetByIDForUpdate(ctx, hostWallet.ID)
		if err != nil {
			return err
		}
		if lockedHost.Balance <= 0 {
			return nil
		}

		available := lockedHost.Balance
		if available < collectAmount {
			collectAmount = available
		}
		if collectAmount <= 0 {
			return nil
		}

		lockedPlatform, err := walletRepo.GetByIDForUpdate(ctx, platformWallet.ID)
		if err != nil {
			return err
		}

		txRecord := &schema.Transaction{
			ID:           uuid.New(),
			Type:         string(domain.TransactionTypePenalty),
			Status:       string(domain.TransactionStatusPending),
			ResourceType: string(domain.ResourceTypeBooking),
			ResourceID:   debt.BookingID,
			Amount:       collectAmount,
			Currency:     debt.Currency,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		if err := txRepo.Create(ctx, txRecord); err != nil {
			return fmt.Errorf("failed to create penalty transaction: %w", err)
		}

		reference := generateReference(domain.TransactionTypePenalty, debt.BookingID, collectAmount, fmt.Sprintf("debt:%s:%d", debt.ID, time.Now().UnixNano()))
		entries := buildDoubleEntry(
			txRecord.ID,
			reference,
			hostWallet.ID,
			platformWallet.ID,
			collectAmount,
			debt.Currency,
			domain.ResourceTypeBooking,
			debt.BookingID,
			fmt.Sprintf("Penalty debt collection for booking %s", debt.BookingID),
		)

		if err := validateDoubleEntry(entries); err != nil {
			return err
		}

		schemaEntries := make([]*schema.LedgerEntry, len(entries))
		for i, entry := range entries {
			schemaEntries[i] = domain.MapLedgerEntryToSchema(entry)
		}

		if err := ledgerRepo.CreateEntries(ctx, schemaEntries); err != nil {
			return fmt.Errorf("failed to create penalty ledger entries: %w", err)
		}

		if lockedHost.Balance < collectAmount {
			return domain.ErrInsufficientBalance
		}
		if err := walletRepo.UpdateBalance(ctx, hostWallet.ID, lockedHost.Balance-collectAmount); err != nil {
			return fmt.Errorf("failed to update host wallet balance: %w", err)
		}
		if err := walletRepo.UpdateBalance(ctx, platformWallet.ID, lockedPlatform.Balance+collectAmount); err != nil {
			return fmt.Errorf("failed to update platform wallet balance: %w", err)
		}

		if err := txRepo.UpdateStatus(ctx, txRecord.ID, string(domain.TransactionStatusCompleted), nil); err != nil {
			return fmt.Errorf("failed to update penalty transaction status: %w", err)
		}

		debt.OutstandingAmount -= collectAmount
		if debt.OutstandingAmount < 0 {
			debt.OutstandingAmount = 0
		}
		debt.UpdatedAt = time.Now()
		if debt.OutstandingAmount == 0 {
			debt.Status = string(domain.PenaltyDebtStatusSettled)
			now := time.Now()
			debt.SettledAt = &now
			settled = true
		}

		if err := debtRepo.Update(ctx, debt); err != nil {
			return fmt.Errorf("failed to update penalty debt: %w", err)
		}

		if settled {
			if err := markHostCancellationPenaltyPaid(ctx, tx, debt.BookingID); err != nil {
				return fmt.Errorf("failed to mark host cancellation penalty paid: %w", err)
			}
		}

		collected = collectAmount
		return nil
	})
	if err != nil {
		if errors.Is(err, domain.ErrInsufficientBalance) {
			return 0, false, nil
		}
		return 0, false, err
	}

	return collected, settled, nil
}

func (s *FinanceServiceImpl) ensurePenaltyDebt(ctx context.Context, hostID, bookingID uuid.UUID, amount int64, currency string) error {
	if s.penaltyDebtRepo == nil {
		return nil
	}
	if hostID == uuid.Nil || bookingID == uuid.Nil || amount <= 0 {
		return nil
	}
	if currency == "" {
		currency = s.platformConfig.Currency.Code
	}

	existing, err := s.penaltyDebtRepo.GetByBookingID(ctx, bookingID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if existing != nil {
		return nil
	}

	debt := &schema.PenaltyDebt{
		ID:                uuid.New(),
		HostID:            hostID,
		BookingID:         bookingID,
		Amount:            amount,
		OutstandingAmount: amount,
		Currency:          currency,
		Status:            string(domain.PenaltyDebtStatusPending),
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	return s.penaltyDebtRepo.Create(ctx, debt)
}

func markHostCancellationPenaltyPaid(ctx context.Context, db *gorm.DB, bookingID uuid.UUID) error {
	if db == nil || bookingID == uuid.Nil {
		return nil
	}
	return db.WithContext(ctx).
		Table("host_cancellation_records").
		Where("booking_id = ?", bookingID).
		Update("penalty_paid", true).
		Error
}
