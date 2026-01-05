package service

import (
	"context"
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
		oldBalance := escrowWallet.Balance
		newBalance := escrowWallet.Balance + amount
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
		escrowOldBalance := escrowWallet.Balance
		escrowNewBalance := escrowWallet.Balance - amount
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

		refundOldBalance := refundPool.Balance
		refundNewBalance := refundPool.Balance + amount
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
		if err := walletRepo.UpdateBalance(ctx, escrowWallet.ID, escrowWallet.Balance-amount); err != nil {
			return fmt.Errorf("failed to update escrow balance: %w", err)
		}
		if err := walletRepo.UpdateBalance(ctx, feeWallet.ID, feeWallet.Balance+amount); err != nil {
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
		if err := walletRepo.UpdateBalance(ctx, escrowWallet.ID, escrowWallet.Balance-amount); err != nil {
			return fmt.Errorf("failed to update escrow balance: %w", err)
		}
		if err := walletRepo.UpdateBalance(ctx, hostWallet.ID, hostWallet.Balance+amount); err != nil {
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
