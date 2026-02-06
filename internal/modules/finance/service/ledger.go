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

// ============================================================================
// Ledger Service Implementation
// ============================================================================

// RecordCharge records a payment charge from guest
func (s *FinanceServiceImpl) RecordCharge(
	ctx context.Context,
	bookingID, paymentID uuid.UUID,
	amount int64,
	currency string,
) (*domain.Transaction, error) {
	if err := validateAmount(amount); err != nil {
		return nil, err
	}

	// Generate idempotency reference
	reference := generateTransactionReference(domain.TransactionTypeCharge, bookingID, paymentID)

	// Check if already recorded
	existing, err := s.ledgerRepo.GetByReference(ctx, reference+":debit")
	if err != nil {
		return nil, fmt.Errorf("failed to check idempotency: %w", err)
	}
	if existing != nil {
		s.log.Warn("duplicate charge detected",
			"booking_id", bookingID,
			"payment_id", paymentID,
		)
		return nil, domain.ErrDuplicateTransaction
	}

	// Get or create booking escrow wallet
	escrowWallet, err := s.GetOrCreateWallet(ctx, domain.OwnerTypeUser, bookingID, domain.WalletTypeEscrow, currency)
	if err != nil {
		return nil, fmt.Errorf("failed to get escrow wallet: %w", err)
	}

	// Record transaction in database transaction
	var transaction *domain.Transaction
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Create transaction-aware repositories
		txRepo := s.transactionRepo.WithTx(tx)
		ledgerRepo := s.ledgerRepo.WithTx(tx)
		walletRepo := s.walletRepo.WithTx(tx)

		// Create transaction record
		transaction = &domain.Transaction{
			ID:           uuid.New(),
			Type:         domain.TransactionTypeCharge,
			Status:       domain.TransactionStatusPending,
			ResourceType: domain.ResourceTypeBooking,
			ResourceID:   bookingID,
			Amount:       amount,
			Currency:     currency,
			PaymentID:    &paymentID,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		txSchema := domain.MapTransactionToSchema(transaction)
		if err := txRepo.Create(ctx, txSchema); err != nil {
			return fmt.Errorf("failed to create transaction: %w", err)
		}

		// Create ledger entries (double-entry: external debit → escrow credit)
		// Note: We don't track external wallet, so only credit entry for escrow
		creditEntry := buildLedgerEntry(
			transaction.ID,
			reference+":credit",
			nil, // No debit wallet (external)
			&escrowWallet.ID,
			amount,
			currency,
			domain.ResourceTypeBooking,
			bookingID,
			fmt.Sprintf("Charge for booking %s", bookingID),
		)

		creditSchema := domain.MapLedgerEntryToSchema(creditEntry)
		if err := ledgerRepo.CreateEntry(ctx, creditSchema); err != nil {
			return fmt.Errorf("failed to create ledger entry: %w", err)
		}

		// Update wallet balance
		// Lock wallet and get fresh balance to prevent race conditions
		lockedWallet, err := walletRepo.GetByIDForUpdate(ctx, escrowWallet.ID)
		if err != nil {
			return fmt.Errorf("failed to lock wallet: %w", err)
		}

		oldBalance := lockedWallet.Balance
		newBalance := lockedWallet.Balance + amount
		if err := walletRepo.UpdateBalance(ctx, escrowWallet.ID, newBalance); err != nil {
			return fmt.Errorf("failed to update wallet balance: %w", err)
		}

		s.log.Info("[AUDIT] wallet_balance_updated",
			"wallet_id", escrowWallet.ID,
			"old_balance", oldBalance,
			"new_balance", newBalance,
			"delta", amount,
			"currency", currency,
			"reason", "charge",
		)

		// Mark transaction as completed
		transaction.MarkCompleted()
		if err := txRepo.UpdateStatus(ctx, transaction.ID, transaction.Status.String(), nil); err != nil {
			return fmt.Errorf("failed to update transaction status: %w", err)
		}

		return nil
	})

	if err != nil {
		s.log.Error("[AUDIT] charge_failed",
			"booking_id", bookingID,
			"payment_id", paymentID,
			"amount", amount,
			"currency", currency,
			"error", err,
		)
		return nil, err
	}

	s.log.Info("[AUDIT] charge_completed",
		"transaction_id", transaction.ID,
		"booking_id", bookingID,
		"payment_id", paymentID,
		"amount", amount,
		"currency", currency,
		"escrow_wallet", escrowWallet.ID,
	)
	return transaction, nil
}

// RecordRefund records a refund to guest
func (s *FinanceServiceImpl) RecordRefund(
	ctx context.Context,
	bookingID, paymentID uuid.UUID,
	amount int64,
	currency string,
) (*domain.Transaction, error) {
	if err := validateAmount(amount); err != nil {
		return nil, err
	}

	// Generate idempotency reference
	reference := generateTransactionReference(domain.TransactionTypeRefund, bookingID, paymentID)

	// Check if already recorded
	existing, err := s.ledgerRepo.GetByReference(ctx, reference+":debit")
	if err != nil {
		return nil, fmt.Errorf("failed to check idempotency: %w", err)
	}
	if existing != nil {
		s.log.Warn("duplicate refund detected",
			"booking_id", bookingID,
			"payment_id", paymentID,
		)
		return nil, domain.ErrDuplicateTransaction
	}

	// Get booking escrow wallet
	escrowWallet, err := s.GetOrCreateWallet(ctx, domain.OwnerTypeUser, bookingID, domain.WalletTypeEscrow, currency)
	if err != nil {
		return nil, fmt.Errorf("failed to get escrow wallet: %w", err)
	}

	// Get platform refund pool wallet
	platformID := uuid.NewSHA1(uuid.NameSpaceDNS, []byte("hauslet"))
	refundPool, err := s.GetOrCreateWallet(ctx, domain.OwnerTypePlatform, platformID, domain.WalletTypeRefundPool, currency)
	if err != nil {
		return nil, fmt.Errorf("failed to get refund pool: %w", err)
	}

	// Check if escrow has sufficient balance
	if escrowWallet.Balance < amount {
		s.log.Warn("[AUDIT] insufficient_balance",
			"wallet_id", escrowWallet.ID,
			"balance", escrowWallet.Balance,
			"required", amount,
			"currency", currency,
			"operation", "refund",
			"booking_id", bookingID,
		)
		return nil, domain.ErrInsufficientBalance
	}

	// Record transaction in database transaction
	var transaction *domain.Transaction
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Create transaction-aware repositories
		txRepo := s.transactionRepo.WithTx(tx)
		ledgerRepo := s.ledgerRepo.WithTx(tx)
		walletRepo := s.walletRepo.WithTx(tx)

		// Create transaction record
		transaction = &domain.Transaction{
			ID:           uuid.New(),
			Type:         domain.TransactionTypeRefund,
			Status:       domain.TransactionStatusPending,
			ResourceType: domain.ResourceTypeBooking,
			ResourceID:   bookingID,
			Amount:       amount,
			Currency:     currency,
			PaymentID:    &paymentID,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		txSchema := domain.MapTransactionToSchema(transaction)
		if err := txRepo.Create(ctx, txSchema); err != nil {
			return fmt.Errorf("failed to create transaction: %w", err)
		}

		// Create ledger entries (double-entry: escrow debit → refund pool credit)
		entries := buildDoubleEntry(
			transaction.ID,
			reference,
			escrowWallet.ID,
			refundPool.ID,
			amount,
			currency,
			domain.ResourceTypeBooking,
			bookingID,
			fmt.Sprintf("Refund for booking %s", bookingID),
		)

		if err := validateDoubleEntry(entries); err != nil {
			return err
		}

		entrySchemas := make([]*schema.LedgerEntry, len(entries))
		for i, entry := range entries {
			entrySchemas[i] = domain.MapLedgerEntryToSchema(entry)
		}

		if err := ledgerRepo.CreateEntries(ctx, entrySchemas); err != nil {
			return fmt.Errorf("failed to create ledger entries: %w", err)
		}

		// Update wallet balances
		// Lock wallets and get fresh balances
		lockedEscrow, err := walletRepo.GetByIDForUpdate(ctx, escrowWallet.ID)
		if err != nil {
			return fmt.Errorf("failed to lock escrow wallet: %w", err)
		}

		escrowOldBalance := lockedEscrow.Balance
		escrowNewBalance := lockedEscrow.Balance - amount
		if err := walletRepo.UpdateBalance(ctx, escrowWallet.ID, escrowNewBalance); err != nil {
			return fmt.Errorf("failed to update escrow balance: %w", err)
		}
		s.log.Info("[AUDIT] wallet_balance_updated",
			"wallet_id", escrowWallet.ID,
			"old_balance", escrowOldBalance,
			"new_balance", escrowNewBalance,
			"delta", -amount,
			"currency", currency,
			"reason", "refund",
		)

		lockedRefundPool, err := walletRepo.GetByIDForUpdate(ctx, refundPool.ID)
		if err != nil {
			return fmt.Errorf("failed to lock refund pool wallet: %w", err)
		}

		refundOldBalance := lockedRefundPool.Balance
		refundNewBalance := lockedRefundPool.Balance + amount
		if err := walletRepo.UpdateBalance(ctx, refundPool.ID, refundNewBalance); err != nil {
			return fmt.Errorf("failed to update refund pool balance: %w", err)
		}
		s.log.Info("[AUDIT] wallet_balance_updated",
			"wallet_id", refundPool.ID,
			"old_balance", refundOldBalance,
			"new_balance", refundNewBalance,
			"delta", amount,
			"currency", currency,
			"reason", "refund_pool",
		)

		// Mark transaction as completed
		transaction.MarkCompleted()
		if err := txRepo.UpdateStatus(ctx, transaction.ID, transaction.Status.String(), nil); err != nil {
			return fmt.Errorf("failed to update transaction status: %w", err)
		}

		return nil
	})

	if err != nil {
		s.log.Error("[AUDIT] refund_failed",
			"booking_id", bookingID,
			"payment_id", paymentID,
			"amount", amount,
			"currency", currency,
			"error", err,
		)
		return nil, err
	}

	s.log.Info("[AUDIT] refund_completed",
		"transaction_id", transaction.ID,
		"booking_id", bookingID,
		"payment_id", paymentID,
		"amount", amount,
		"currency", currency,
		"escrow_wallet", escrowWallet.ID,
		"refund_pool", refundPool.ID,
	)
	return transaction, nil
}

// RecordCommission records platform commission
func (s *FinanceServiceImpl) RecordCommission(
	ctx context.Context,
	bookingID uuid.UUID,
	amount int64,
	currency string,
) (*domain.Transaction, error) {
	if err := validateAmount(amount); err != nil {
		return nil, err
	}

	// Generate idempotency reference
	reference := generateTimestampReference(domain.TransactionTypeCommission, bookingID, amount)

	// Get booking escrow wallet
	escrowWallet, err := s.GetOrCreateWallet(ctx, domain.OwnerTypeUser, bookingID, domain.WalletTypeEscrow, currency)
	if err != nil {
		return nil, fmt.Errorf("failed to get escrow wallet: %w", err)
	}

	// Get platform fee wallet
	platformID := uuid.NewSHA1(uuid.NameSpaceDNS, []byte("hauslet"))
	feeWallet, err := s.GetOrCreateWallet(ctx, domain.OwnerTypePlatform, platformID, domain.WalletTypePlatformFee, currency)
	if err != nil {
		return nil, fmt.Errorf("failed to get platform fee wallet: %w", err)
	}

	// Check if escrow has sufficient balance
	if escrowWallet.Balance < amount {
		s.log.Warn("[AUDIT] insufficient_balance",
			"wallet_id", escrowWallet.ID,
			"balance", escrowWallet.Balance,
			"required", amount,
			"currency", currency,
			"operation", "commission",
			"booking_id", bookingID,
		)
		return nil, domain.ErrInsufficientBalance
	}

	// Record transaction in database transaction
	var transaction *domain.Transaction
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Create transaction-aware repositories
		txRepo := s.transactionRepo.WithTx(tx)
		ledgerRepo := s.ledgerRepo.WithTx(tx)
		walletRepo := s.walletRepo.WithTx(tx)

		// Create transaction record
		transaction = &domain.Transaction{
			ID:           uuid.New(),
			Type:         domain.TransactionTypeCommission,
			Status:       domain.TransactionStatusPending,
			ResourceType: domain.ResourceTypeBooking,
			ResourceID:   bookingID,
			Amount:       amount,
			Currency:     currency,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		txSchema := domain.MapTransactionToSchema(transaction)
		if err := txRepo.Create(ctx, txSchema); err != nil {
			return fmt.Errorf("failed to create transaction: %w", err)
		}

		// Create ledger entries (double-entry: escrow debit → platform fee credit)
		entries := buildDoubleEntry(
			transaction.ID,
			reference,
			escrowWallet.ID,
			feeWallet.ID,
			amount,
			currency,
			domain.ResourceTypeBooking,
			bookingID,
			fmt.Sprintf("Platform commission for booking %s", bookingID),
		)

		if err := validateDoubleEntry(entries); err != nil {
			return err
		}

		entrySchemas := make([]*schema.LedgerEntry, len(entries))
		for i, entry := range entries {
			entrySchemas[i] = domain.MapLedgerEntryToSchema(entry)
		}

		if err := ledgerRepo.CreateEntries(ctx, entrySchemas); err != nil {
			return fmt.Errorf("failed to create ledger entries: %w", err)
		}

		// Update wallet balances
		// Lock wallets and get fresh balances
		lockedEscrow, err := walletRepo.GetByIDForUpdate(ctx, escrowWallet.ID)
		if err != nil {
			return fmt.Errorf("failed to lock escrow wallet: %w", err)
		}
		if err := walletRepo.UpdateBalance(ctx, escrowWallet.ID, lockedEscrow.Balance-amount); err != nil {
			return fmt.Errorf("failed to update escrow balance: %w", err)
		}

		lockedFeeWallet, err := walletRepo.GetByIDForUpdate(ctx, feeWallet.ID)
		if err != nil {
			return fmt.Errorf("failed to lock fee wallet: %w", err)
		}
		if err := walletRepo.UpdateBalance(ctx, feeWallet.ID, lockedFeeWallet.Balance+amount); err != nil {
			return fmt.Errorf("failed to update fee wallet balance: %w", err)
		}

		// Mark transaction as completed
		transaction.MarkCompleted()
		if err := txRepo.UpdateStatus(ctx, transaction.ID, transaction.Status.String(), nil); err != nil {
			return fmt.Errorf("failed to update transaction status: %w", err)
		}

		return nil
	})

	if err != nil {
		s.log.Error("[AUDIT] commission_failed",
			"booking_id", bookingID,
			"amount", amount,
			"currency", currency,
			"error", err,
		)
		return nil, err
	}

	s.log.Info("[AUDIT] commission_completed",
		"transaction_id", transaction.ID,
		"booking_id", bookingID,
		"amount", amount,
		"currency", currency,
		"platform_fee_wallet", feeWallet.ID,
	)
	return transaction, nil
}

// RecordPayout records payout to host
func (s *FinanceServiceImpl) RecordPayout(
	ctx context.Context,
	bookingID, hostID uuid.UUID,
	amount int64,
	currency string,
) (*domain.Transaction, error) {
	if err := validateAmount(amount); err != nil {
		return nil, err
	}

	// Generate idempotency reference
	reference := generateTimestampReference(domain.TransactionTypePayout, bookingID, amount)

	// Get booking escrow wallet
	escrowWallet, err := s.GetOrCreateWallet(ctx, domain.OwnerTypeUser, bookingID, domain.WalletTypeEscrow, currency)
	if err != nil {
		return nil, fmt.Errorf("failed to get escrow wallet: %w", err)
	}

	// Get host available wallet
	hostWallet, err := s.GetOrCreateWallet(ctx, domain.OwnerTypeUser, hostID, domain.WalletTypeHostAvailable, currency)
	if err != nil {
		return nil, fmt.Errorf("failed to get host wallet: %w", err)
	}

	// Check if escrow has sufficient balance
	if escrowWallet.Balance < amount {
		s.log.Warn("[AUDIT] insufficient_balance",
			"wallet_id", escrowWallet.ID,
			"balance", escrowWallet.Balance,
			"required", amount,
			"currency", currency,
			"operation", "payout",
			"booking_id", bookingID,
			"host_id", hostID,
		)
		return nil, domain.ErrInsufficientBalance
	}

	// Record transaction in database transaction
	var transaction *domain.Transaction
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Create transaction-aware repositories
		txRepo := s.transactionRepo.WithTx(tx)
		ledgerRepo := s.ledgerRepo.WithTx(tx)
		walletRepo := s.walletRepo.WithTx(tx)

		// Create transaction record
		transaction = &domain.Transaction{
			ID:           uuid.New(),
			Type:         domain.TransactionTypePayout,
			Status:       domain.TransactionStatusPending,
			ResourceType: domain.ResourceTypeBooking,
			ResourceID:   bookingID,
			Amount:       amount,
			Currency:     currency,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		txSchema := domain.MapTransactionToSchema(transaction)
		if err := txRepo.Create(ctx, txSchema); err != nil {
			return fmt.Errorf("failed to create transaction: %w", err)
		}

		// Create ledger entries (double-entry: escrow debit → host available credit)
		entries := buildDoubleEntry(
			transaction.ID,
			reference,
			escrowWallet.ID,
			hostWallet.ID,
			amount,
			currency,
			domain.ResourceTypeBooking,
			bookingID,
			fmt.Sprintf("Payout to host for booking %s", bookingID),
		)

		if err := validateDoubleEntry(entries); err != nil {
			return err
		}

		entrySchemas := make([]*schema.LedgerEntry, len(entries))
		for i, entry := range entries {
			entrySchemas[i] = domain.MapLedgerEntryToSchema(entry)
		}

		if err := ledgerRepo.CreateEntries(ctx, entrySchemas); err != nil {
			return fmt.Errorf("failed to create ledger entries: %w", err)
		}

		// Update wallet balances
		// Lock wallets and get fresh balances
		lockedEscrow, err := walletRepo.GetByIDForUpdate(ctx, escrowWallet.ID)
		if err != nil {
			return fmt.Errorf("failed to lock escrow wallet: %w", err)
		}
		if err := walletRepo.UpdateBalance(ctx, escrowWallet.ID, lockedEscrow.Balance-amount); err != nil {
			return fmt.Errorf("failed to update escrow balance: %w", err)
		}

		lockedHostWallet, err := walletRepo.GetByIDForUpdate(ctx, hostWallet.ID)
		if err != nil {
			return fmt.Errorf("failed to lock host wallet: %w", err)
		}
		if err := walletRepo.UpdateBalance(ctx, hostWallet.ID, lockedHostWallet.Balance+amount); err != nil {
			return fmt.Errorf("failed to update host wallet balance: %w", err)
		}

		// Mark transaction as completed
		transaction.MarkCompleted()
		if err := txRepo.UpdateStatus(ctx, transaction.ID, transaction.Status.String(), nil); err != nil {
			return fmt.Errorf("failed to update transaction status: %w", err)
		}

		return nil
	})

	if err != nil {
		s.log.Error("[AUDIT] payout_failed",
			"booking_id", bookingID,
			"host_id", hostID,
			"amount", amount,
			"currency", currency,
			"error", err,
		)
		return nil, err
	}

	s.log.Info("[AUDIT] payout_completed",
		"transaction_id", transaction.ID,
		"booking_id", bookingID,
		"host_id", hostID,
		"amount", amount,
		"currency", currency,
		"host_wallet", hostWallet.ID,
	)
	return transaction, nil
}

// DeductPenalty withdraws penalty amount from host wallet.
func (s *FinanceServiceImpl) DeductPenalty(
	ctx context.Context,
	hostID uuid.UUID,
	amount int64,
	bookingID uuid.UUID,
	currency string,
) error {
	if err := validateAmount(amount); err != nil {
		return err
	}
	if hostID == uuid.Nil {
		return errors.New("hostID cannot be nil")
	}
	if bookingID == uuid.Nil {
		return errors.New("bookingID cannot be nil")
	}
	if currency == "" {
		currency = s.platformConfig.Currency.Code
	}

	// Generate idempotency reference
	reference := generateTimestampReference(domain.TransactionTypePenalty, bookingID, amount)

	// Get host available wallet
	hostWallet, err := s.GetOrCreateWallet(ctx, domain.OwnerTypeUser, hostID, domain.WalletTypeHostAvailable, currency)
	if err != nil {
		return fmt.Errorf("failed to get host wallet: %w", err)
	}

	// Get platform fee wallet
	platformID := uuid.NewSHA1(uuid.NameSpaceDNS, []byte("hauslet"))
	feeWallet, err := s.GetOrCreateWallet(ctx, domain.OwnerTypePlatform, platformID, domain.WalletTypePlatformFee, currency)
	if err != nil {
		return fmt.Errorf("failed to get platform fee wallet: %w", err)
	}

	// Check if host has sufficient balance
	if hostWallet.Balance < amount {
		s.log.Warn("[AUDIT] insufficient_balance",
			"wallet_id", hostWallet.ID,
			"balance", hostWallet.Balance,
			"required", amount,
			"currency", currency,
			"operation", "penalty",
			"booking_id", bookingID,
			"host_id", hostID,
		)
		if err := s.ensurePenaltyDebt(ctx, hostID, bookingID, amount, currency); err != nil && s.log != nil {
			s.log.Warn("failed to record penalty debt", "error", err, "booking_id", bookingID, "host_id", hostID)
		}
		return domain.ErrInsufficientBalance
	}

	// Record transaction in database transaction
	var transaction *domain.Transaction
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Create transaction-aware repositories
		txRepo := s.transactionRepo.WithTx(tx)
		ledgerRepo := s.ledgerRepo.WithTx(tx)
		walletRepo := s.walletRepo.WithTx(tx)

		// Create transaction record
		transaction = &domain.Transaction{
			ID:           uuid.New(),
			Type:         domain.TransactionTypePenalty,
			Status:       domain.TransactionStatusPending,
			ResourceType: domain.ResourceTypeBooking,
			ResourceID:   bookingID,
			Amount:       amount,
			Currency:     currency,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		txSchema := domain.MapTransactionToSchema(transaction)
		if err := txRepo.Create(ctx, txSchema); err != nil {
			return fmt.Errorf("failed to create transaction: %w", err)
		}

		// Create ledger entries (double-entry: host debit → platform fee credit)
		entries := buildDoubleEntry(
			transaction.ID,
			reference,
			hostWallet.ID,
			feeWallet.ID,
			amount,
			currency,
			domain.ResourceTypeBooking,
			bookingID,
			fmt.Sprintf("Host cancellation penalty for booking %s", bookingID),
		)

		if err := validateDoubleEntry(entries); err != nil {
			return err
		}

		entrySchemas := make([]*schema.LedgerEntry, len(entries))
		for i, entry := range entries {
			entrySchemas[i] = domain.MapLedgerEntryToSchema(entry)
		}

		if err := ledgerRepo.CreateEntries(ctx, entrySchemas); err != nil {
			return fmt.Errorf("failed to create ledger entries: %w", err)
		}

		// Update wallet balances
		lockedHost, err := walletRepo.GetByIDForUpdate(ctx, hostWallet.ID)
		if err != nil {
			return fmt.Errorf("failed to lock host wallet: %w", err)
		}
		if lockedHost.Balance < amount {
			return domain.ErrInsufficientBalance
		}
		if err := walletRepo.UpdateBalance(ctx, hostWallet.ID, lockedHost.Balance-amount); err != nil {
			return fmt.Errorf("failed to update host balance: %w", err)
		}

		lockedFeeWallet, err := walletRepo.GetByIDForUpdate(ctx, feeWallet.ID)
		if err != nil {
			return fmt.Errorf("failed to lock fee wallet: %w", err)
		}
		if err := walletRepo.UpdateBalance(ctx, feeWallet.ID, lockedFeeWallet.Balance+amount); err != nil {
			return fmt.Errorf("failed to update fee wallet balance: %w", err)
		}

		// Mark transaction as completed
		transaction.MarkCompleted()
		if err := txRepo.UpdateStatus(ctx, transaction.ID, transaction.Status.String(), nil); err != nil {
			return fmt.Errorf("failed to update transaction status: %w", err)
		}

		return nil
	})

	if err != nil {
		if errors.Is(err, domain.ErrInsufficientBalance) {
			if debtErr := s.ensurePenaltyDebt(ctx, hostID, bookingID, amount, currency); debtErr != nil && s.log != nil {
				s.log.Warn("failed to record penalty debt", "error", debtErr, "booking_id", bookingID, "host_id", hostID)
			}
		}
		s.log.Error("[AUDIT] penalty_failed",
			"booking_id", bookingID,
			"host_id", hostID,
			"amount", amount,
			"currency", currency,
			"error", err,
		)
		return err
	}

	s.log.Info("[AUDIT] penalty_completed",
		"transaction_id", transaction.ID,
		"booking_id", bookingID,
		"host_id", hostID,
		"amount", amount,
		"currency", currency,
		"platform_fee_wallet", feeWallet.ID,
	)

	return nil
}

// SettleCancelledBooking settles remaining escrow funds after a booking cancellation.
// This handles the case where a guest receives a partial or zero refund due to cancellation
// policy (e.g., strict policy, late cancellation). The remaining funds are distributed:
// - Platform commission goes to platform fee wallet
// - Host's share goes to host's available wallet
//
// Note: Unlike normal payouts, this does NOT initiate a bank disbursement automatically.
// The host will need to request withdrawal, or admin can process it manually.
func (s *FinanceServiceImpl) SettleCancelledBooking(
	ctx context.Context,
	bookingID, hostID uuid.UUID,
	hostAmount, platformAmount int64,
	currency string,
) error {
	// Get the escrow wallet for this booking
	escrowWallet, err := s.GetOrCreateWallet(ctx, domain.OwnerTypeUser, bookingID, domain.WalletTypeEscrow, currency)
	if err != nil {
		return fmt.Errorf("failed to get escrow wallet: %w", err)
	}

	// Check if there are remaining funds to settle
	remainingBalance := escrowWallet.Balance
	if remainingBalance <= 0 {
		s.log.Info("[AUDIT] cancellation_settlement_skipped",
			"booking_id", bookingID,
			"reason", "no_remaining_balance",
		)
		return nil // Nothing to settle
	}

	s.log.Info("[AUDIT] cancellation_settlement_started",
		"booking_id", bookingID,
		"host_id", hostID,
		"remaining_balance", remainingBalance,
		"currency", currency,
		"expected_host_amount", hostAmount,
		"expected_platform_amount", platformAmount,
	)

	// Validate amounts against remaining balance
	totalRequired := hostAmount + platformAmount
	if totalRequired > remainingBalance {
		s.log.Warn("[AUDIT] settlement_amount_mismatch_underflow",
			"booking_id", bookingID,
			"remaining", remainingBalance,
			"required", totalRequired,
		)
		// Adjust platform amount downwards to match available funds
		diff := totalRequired - remainingBalance
		platformAmount -= diff
		if platformAmount < 0 {
			// If platform amount becomes negative, reduce host amount (highly unlikely)
			hostAmount += platformAmount
			platformAmount = 0
		}
	}
	// Removed logic that auto-sweeps excess funds to platform.
	// Excess funds must remain in escrow for refund processing.

	// Use finalised amounts
	commission := platformAmount
	hostPayout := hostAmount

	s.log.Info("[AUDIT] cancellation_settlement_breakdown",
		"booking_id", bookingID,
		"remaining_balance", remainingBalance,
		"commission", commission,
		"host_payout", hostPayout,
	)

	// Generate idempotency reference
	reference := generateTimestampReference(domain.TransactionTypeCancellationSettlement, bookingID, remainingBalance)

	// Check if already settled
	existing, _ := s.ledgerRepo.GetByReference(ctx, reference+":commission:debit")
	if existing != nil {
		s.log.Warn("[AUDIT] duplicate_cancellation_settlement",
			"booking_id", bookingID,
		)
		return domain.ErrDuplicateTransaction
	}

	// Get platform fee wallet
	platformID := uuid.NewSHA1(uuid.NameSpaceDNS, []byte("hauslet"))
	platformWallet, err := s.GetOrCreateWallet(ctx, domain.OwnerTypePlatform, platformID, domain.WalletTypePlatformFee, currency)
	if err != nil {
		return fmt.Errorf("failed to get platform wallet: %w", err)
	}

	// Get or create host available wallet
	hostWallet, err := s.GetOrCreateWallet(ctx, domain.OwnerTypeUser, hostID, domain.WalletTypeHostAvailable, currency)
	if err != nil {
		return fmt.Errorf("failed to get host wallet: %w", err)
	}

	// Record settlement in database transaction
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Create transaction-aware repositories
		txRepo := s.transactionRepo.WithTx(tx)
		ledgerRepo := s.ledgerRepo.WithTx(tx)
		walletRepo := s.walletRepo.WithTx(tx)

		// Create settlement transaction record
		transaction := &domain.Transaction{
			ID:           uuid.New(),
			Type:         domain.TransactionTypeCancellationSettlement,
			Status:       domain.TransactionStatusPending,
			ResourceType: domain.ResourceTypeBooking,
			ResourceID:   bookingID,
			Amount:       remainingBalance,
			Currency:     currency,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		txSchema := domain.MapTransactionToSchema(transaction)
		if err := txRepo.Create(ctx, txSchema); err != nil {
			return fmt.Errorf("failed to create transaction: %w", err)
		}

		// Lock escrow wallet
		lockedEscrow, err := walletRepo.GetByIDForUpdate(ctx, escrowWallet.ID)
		if err != nil {
			return fmt.Errorf("failed to lock escrow wallet: %w", err)
		}

		// Verify escrow still has the expected balance
		if lockedEscrow.Balance != remainingBalance {
			s.log.Warn("[AUDIT] escrow_balance_changed",
				"booking_id", bookingID,
				"expected", remainingBalance,
				"actual", lockedEscrow.Balance,
			)
			// Use actual balance
			remainingBalance = lockedEscrow.Balance

			// Re-apply adjustment logic
			commission = platformAmount
			hostPayout = hostAmount

			if hostPayout+commission > remainingBalance {
				diff := (hostPayout + commission) - remainingBalance
				commission -= diff
				if commission < 0 {
					hostPayout += commission
					commission = 0
				}
			} else if hostPayout+commission < remainingBalance {
				commission += remainingBalance - (hostPayout + commission)
			}
		}

		if remainingBalance <= 0 {
			return nil // Nothing to settle
		}

		// Step 1: Record commission (escrow → platform fee wallet)
		if commission > 0 {
			commissionEntries := buildDoubleEntry(
				transaction.ID,
				reference+":commission",
				escrowWallet.ID,
				platformWallet.ID,
				commission,
				currency,
				domain.ResourceTypeBooking,
				bookingID,
				fmt.Sprintf("Cancellation commission for booking %s", bookingID),
			)

			if err := validateDoubleEntry(commissionEntries); err != nil {
				return err
			}

			entrySchemas := make([]*schema.LedgerEntry, len(commissionEntries))
			for i, entry := range commissionEntries {
				entrySchemas[i] = domain.MapLedgerEntryToSchema(entry)
			}

			if err := ledgerRepo.CreateEntries(ctx, entrySchemas); err != nil {
				return fmt.Errorf("failed to create commission ledger entries: %w", err)
			}

			// Update platform fee wallet balance
			lockedPlatform, err := walletRepo.GetByIDForUpdate(ctx, platformWallet.ID)
			if err != nil {
				return fmt.Errorf("failed to lock platform wallet: %w", err)
			}
			if err := walletRepo.UpdateBalance(ctx, platformWallet.ID, lockedPlatform.Balance+commission); err != nil {
				return fmt.Errorf("failed to update platform wallet balance: %w", err)
			}

			s.log.Info("[AUDIT] cancellation_commission_recorded",
				"booking_id", bookingID,
				"amount", commission,
				"currency", currency,
			)
		}

		// Step 2: Record host payout (escrow → host available wallet)
		if hostPayout > 0 {
			payoutEntries := buildDoubleEntry(
				transaction.ID,
				reference+":payout",
				escrowWallet.ID,
				hostWallet.ID,
				hostPayout,
				currency,
				domain.ResourceTypeBooking,
				bookingID,
				fmt.Sprintf("Cancellation payout to host for booking %s", bookingID),
			)

			if err := validateDoubleEntry(payoutEntries); err != nil {
				return err
			}

			entrySchemas := make([]*schema.LedgerEntry, len(payoutEntries))
			for i, entry := range payoutEntries {
				entrySchemas[i] = domain.MapLedgerEntryToSchema(entry)
			}

			if err := ledgerRepo.CreateEntries(ctx, entrySchemas); err != nil {
				return fmt.Errorf("failed to create payout ledger entries: %w", err)
			}

			// Update host wallet balance
			lockedHost, err := walletRepo.GetByIDForUpdate(ctx, hostWallet.ID)
			if err != nil {
				return fmt.Errorf("failed to lock host wallet: %w", err)
			}
			if err := walletRepo.UpdateBalance(ctx, hostWallet.ID, lockedHost.Balance+hostPayout); err != nil {
				return fmt.Errorf("failed to update host wallet balance: %w", err)
			}

			s.log.Info("[AUDIT] cancellation_payout_recorded",
				"booking_id", bookingID,
				"host_id", hostID,
				"amount", hostPayout,
				"currency", currency,
			)
		}

		// Zero out escrow wallet
		if err := walletRepo.UpdateBalance(ctx, escrowWallet.ID, 0); err != nil {
			return fmt.Errorf("failed to zero escrow wallet: %w", err)
		}

		s.log.Info("[AUDIT] escrow_zeroed",
			"booking_id", bookingID,
			"wallet_id", escrowWallet.ID,
		)

		// Mark transaction as completed
		transaction.MarkCompleted()
		if err := txRepo.UpdateStatus(ctx, transaction.ID, transaction.Status.String(), nil); err != nil {
			return fmt.Errorf("failed to update transaction status: %w", err)
		}

		return nil
	})

	if err != nil {
		s.log.Error("[AUDIT] cancellation_settlement_failed",
			"booking_id", bookingID,
			"host_id", hostID,
			"remaining_balance", remainingBalance,
			"error", err,
		)
		return err
	}

	s.log.Info("[AUDIT] cancellation_settlement_completed",
		"booking_id", bookingID,
		"host_id", hostID,
		"commission", commission,
		"host_payout", hostPayout,
		"currency", currency,
	)

	return nil
}

// GetTransactionHistory returns all transactions for a resource
func (s *FinanceServiceImpl) GetTransactionHistory(
	ctx context.Context,
	resourceType domain.ResourceType,
	resourceID uuid.UUID,
) ([]*domain.Transaction, error) {
	txSchemas, err := s.transactionRepo.ListByResource(ctx, resourceType.String(), resourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction history: %w", err)
	}

	transactions := make([]*domain.Transaction, len(txSchemas))
	for i, txSchema := range txSchemas {
		transactions[i] = domain.MapTransactionFromSchema(txSchema)
	}

	return transactions, nil
}

// GetWalletHistory returns ledger entries for a wallet
func (s *FinanceServiceImpl) GetWalletHistory(
	ctx context.Context,
	walletID uuid.UUID,
	limit, offset int,
) ([]*domain.LedgerEntry, error) {
	entrySchemas, err := s.ledgerRepo.ListByWallet(ctx, walletID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get wallet history: %w", err)
	}

	entries := make([]*domain.LedgerEntry, len(entrySchemas))
	for i, entrySchema := range entrySchemas {
		entries[i] = domain.MapLedgerEntryFromSchema(entrySchema)
	}

	return entries, nil
}
