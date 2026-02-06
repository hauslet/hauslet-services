package service

import (
	"context"
	"fmt"
	"hauslet/internal/modules/finance/domain"
	financeSchema "hauslet/internal/modules/finance/repository/schema"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// recordCommissionInternal records commission in the ledger
func (s *PayoutServiceImpl) recordCommissionInternal(
	ctx context.Context,
	tx *gorm.DB,
	bookingID uuid.UUID,
	escrowWalletID uuid.UUID,
	platformWalletID uuid.UUID,
	amount int64,
	currency string,
) (*domain.Transaction, error) {
	// Create transaction-aware repositories
	txRepo := s.transactionRepo.WithTx(tx)
	ledgerRepo := s.ledgerRepo.WithTx(tx)
	walletRepo := s.walletRepo.WithTx(tx)

	// Generate reference for idempotency
	nonce := fmt.Sprintf("%d", time.Now().UnixNano())
	reference := generateReference(domain.TransactionTypeCommission, bookingID, amount, nonce)

	// Create transaction record
	txRecord := &financeSchema.Transaction{
		ID:           uuid.New(),
		Type:         string(domain.TransactionTypeCommission),
		Status:       string(domain.TransactionStatusPending),
		ResourceType: string(domain.ResourceTypeBooking),
		ResourceID:   bookingID,
		Amount:       amount,
		Currency:     currency,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := txRepo.Create(ctx, txRecord); err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	// Build double-entry ledger entries
	entries := buildDoubleEntry(
		txRecord.ID,
		reference,
		escrowWalletID,
		platformWalletID,
		amount,
		currency,
		domain.ResourceTypeBooking,
		bookingID,
		fmt.Sprintf("Platform commission for booking %s", bookingID),
	)

	// Validate double-entry
	if err := validateDoubleEntry(entries); err != nil {
		return nil, err
	}

	// Create ledger entries
	schemaEntries := make([]*financeSchema.LedgerEntry, len(entries))
	for i, entry := range entries {
		schemaEntries[i] = domain.MapLedgerEntryToSchema(entry)
	}

	if err := ledgerRepo.CreateEntries(ctx, schemaEntries); err != nil {
		return nil, fmt.Errorf("failed to create ledger entries: %w", err)
	}

	// Fetch current wallet balances and calculate new balances
	escrowWallet, err := walletRepo.GetByID(ctx, escrowWalletID)
	if err != nil {
		return nil, fmt.Errorf("failed to get escrow wallet: %w", err)
	}
	if escrowWallet == nil {
		return nil, fmt.Errorf("escrow wallet not found: %s", escrowWalletID)
	}

	platformWallet, err := walletRepo.GetByID(ctx, platformWalletID)
	if err != nil {
		return nil, fmt.Errorf("failed to get platform wallet: %w", err)
	}
	if platformWallet == nil {
		return nil, fmt.Errorf("platform wallet not found: %s", platformWalletID)
	}

	// Update escrow wallet balance (debit)
	escrowNewBalance := escrowWallet.Balance - amount
	if err := walletRepo.UpdateBalance(ctx, escrowWalletID, escrowNewBalance); err != nil {
		return nil, fmt.Errorf("failed to update escrow wallet: %w", err)
	}

	// Update platform wallet balance (credit)
	platformNewBalance := platformWallet.Balance + amount
	if err := walletRepo.UpdateBalance(ctx, platformWalletID, platformNewBalance); err != nil {
		return nil, fmt.Errorf("failed to update platform wallet: %w", err)
	}

	// Mark transaction as completed
	if err := txRepo.UpdateStatus(ctx, txRecord.ID, string(domain.TransactionStatusCompleted), nil); err != nil {
		return nil, fmt.Errorf("failed to update transaction status: %w", err)
	}

	return domain.MapTransactionFromSchema(txRecord), nil
}

// recordPenaltyInternal records a penalty collection in the ledger (escrow → platform fee).
func (s *PayoutServiceImpl) recordPenaltyInternal(
	ctx context.Context,
	tx *gorm.DB,
	penaltyBookingID uuid.UUID,
	escrowWalletID uuid.UUID,
	platformWalletID uuid.UUID,
	amount int64,
	currency string,
	memo string,
) (*domain.Transaction, error) {
	// Create transaction-aware repositories
	txRepo := s.transactionRepo.WithTx(tx)
	ledgerRepo := s.ledgerRepo.WithTx(tx)
	walletRepo := s.walletRepo.WithTx(tx)

	// Generate reference for idempotency
	nonce := fmt.Sprintf("%d", time.Now().UnixNano())
	reference := generateReference(domain.TransactionTypePenalty, penaltyBookingID, amount, nonce)

	// Create transaction record
	txRecord := &financeSchema.Transaction{
		ID:           uuid.New(),
		Type:         string(domain.TransactionTypePenalty),
		Status:       string(domain.TransactionStatusPending),
		ResourceType: string(domain.ResourceTypeBooking),
		ResourceID:   penaltyBookingID,
		Amount:       amount,
		Currency:     currency,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := txRepo.Create(ctx, txRecord); err != nil {
		return nil, fmt.Errorf("failed to create penalty transaction: %w", err)
	}

	// Build double-entry ledger entries
	entries := buildDoubleEntry(
		txRecord.ID,
		reference,
		escrowWalletID,
		platformWalletID,
		amount,
		currency,
		domain.ResourceTypeBooking,
		penaltyBookingID,
		memo,
	)

	// Validate double-entry
	if err := validateDoubleEntry(entries); err != nil {
		return nil, err
	}

	// Create ledger entries
	schemaEntries := make([]*financeSchema.LedgerEntry, len(entries))
	for i, entry := range entries {
		schemaEntries[i] = domain.MapLedgerEntryToSchema(entry)
	}

	if err := ledgerRepo.CreateEntries(ctx, schemaEntries); err != nil {
		return nil, fmt.Errorf("failed to create penalty ledger entries: %w", err)
	}

	// Fetch current wallet balances and calculate new balances
	escrowWallet, err := walletRepo.GetByID(ctx, escrowWalletID)
	if err != nil {
		return nil, fmt.Errorf("failed to get escrow wallet: %w", err)
	}
	if escrowWallet == nil {
		return nil, fmt.Errorf("escrow wallet not found: %s", escrowWalletID)
	}

	platformWallet, err := walletRepo.GetByID(ctx, platformWalletID)
	if err != nil {
		return nil, fmt.Errorf("failed to get platform wallet: %w", err)
	}
	if platformWallet == nil {
		return nil, fmt.Errorf("platform wallet not found: %s", platformWalletID)
	}

	// Update escrow wallet balance (debit)
	escrowNewBalance := escrowWallet.Balance - amount
	if err := walletRepo.UpdateBalance(ctx, escrowWalletID, escrowNewBalance); err != nil {
		return nil, fmt.Errorf("failed to update escrow wallet: %w", err)
	}

	// Update platform wallet balance (credit)
	platformNewBalance := platformWallet.Balance + amount
	if err := walletRepo.UpdateBalance(ctx, platformWalletID, platformNewBalance); err != nil {
		return nil, fmt.Errorf("failed to update platform wallet: %w", err)
	}

	// Mark transaction as completed
	if err := txRepo.UpdateStatus(ctx, txRecord.ID, string(domain.TransactionStatusCompleted), nil); err != nil {
		return nil, fmt.Errorf("failed to update penalty transaction status: %w", err)
	}

	return domain.MapTransactionFromSchema(txRecord), nil
}

// recordPayoutInternal records payout in the ledger
func (s *PayoutServiceImpl) recordPayoutInternal(
	ctx context.Context,
	tx *gorm.DB,
	bookingID uuid.UUID,
	escrowWalletID uuid.UUID,
	hostWalletID uuid.UUID,
	amount int64,
	currency string,
) (*domain.Transaction, error) {
	// Create transaction-aware repositories
	txRepo := s.transactionRepo.WithTx(tx)
	ledgerRepo := s.ledgerRepo.WithTx(tx)
	walletRepo := s.walletRepo.WithTx(tx)

	// Generate reference for idempotency
	nonce := fmt.Sprintf("%d", time.Now().UnixNano())
	reference := generateReference(domain.TransactionTypePayout, bookingID, amount, nonce)

	// Create transaction record
	txRecord := &financeSchema.Transaction{
		ID:           uuid.New(),
		Type:         string(domain.TransactionTypePayout),
		Status:       string(domain.TransactionStatusPending),
		ResourceType: string(domain.ResourceTypeBooking),
		ResourceID:   bookingID,
		Amount:       amount,
		Currency:     currency,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := txRepo.Create(ctx, txRecord); err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	// Build double-entry ledger entries
	entries := buildDoubleEntry(
		txRecord.ID,
		reference,
		escrowWalletID,
		hostWalletID,
		amount,
		currency,
		domain.ResourceTypeBooking,
		bookingID,
		fmt.Sprintf("Host payout for booking %s", bookingID),
	)

	// Validate double-entry
	if err := validateDoubleEntry(entries); err != nil {
		return nil, err
	}

	// Create ledger entries
	schemaEntries := make([]*financeSchema.LedgerEntry, len(entries))
	for i, entry := range entries {
		schemaEntries[i] = domain.MapLedgerEntryToSchema(entry)
	}

	if err := ledgerRepo.CreateEntries(ctx, schemaEntries); err != nil {
		return nil, fmt.Errorf("failed to create ledger entries: %w", err)
	}

	// Fetch current wallet balances and calculate new balances
	escrowWallet, err := walletRepo.GetByID(ctx, escrowWalletID)
	if err != nil {
		return nil, fmt.Errorf("failed to get escrow wallet: %w", err)
	}
	if escrowWallet == nil {
		return nil, fmt.Errorf("escrow wallet not found: %s", escrowWalletID)
	}

	hostWallet, err := walletRepo.GetByID(ctx, hostWalletID)
	if err != nil {
		return nil, fmt.Errorf("failed to get host wallet: %w", err)
	}
	if hostWallet == nil {
		return nil, fmt.Errorf("host wallet not found: %s", hostWalletID)
	}

	// Update escrow wallet balance (debit)
	escrowNewBalance := escrowWallet.Balance - amount
	if err := walletRepo.UpdateBalance(ctx, escrowWalletID, escrowNewBalance); err != nil {
		return nil, fmt.Errorf("failed to update escrow wallet: %w", err)
	}

	// Update host wallet balance (credit)
	hostNewBalance := hostWallet.Balance + amount
	if err := walletRepo.UpdateBalance(ctx, hostWalletID, hostNewBalance); err != nil {
		return nil, fmt.Errorf("failed to update host wallet: %w", err)
	}

	// Mark transaction as completed
	if err := txRepo.UpdateStatus(ctx, txRecord.ID, string(domain.TransactionStatusCompleted), nil); err != nil {
		return nil, fmt.Errorf("failed to update transaction status: %w", err)
	}

	return domain.MapTransactionFromSchema(txRecord), nil
}
