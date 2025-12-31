package service

import (
	"context"
	"fmt"
	"hauslet/internal/modules/finance/domain"
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// Wallet Service Implementation
// ============================================================================

// GetOrCreateWallet retrieves or creates a wallet for the given owner
func (s *FinanceServiceImpl) GetOrCreateWallet(
	ctx context.Context,
	ownerType domain.OwnerType,
	ownerID uuid.UUID,
	walletType domain.WalletType,
	currency string,
) (*domain.Wallet, error) {
	// Try to get existing wallet
	existing, err := s.walletRepo.GetByOwner(ctx, ownerType.String(), ownerID, walletType.String())
	if err != nil {
		return nil, fmt.Errorf("failed to query wallet: %w", err)
	}

	if existing != nil {
		return domain.MapWalletFromSchema(existing), nil
	}

	// Create new wallet
	wallet := &domain.Wallet{
		ID:         uuid.New(),
		OwnerType:  ownerType,
		OwnerID:    ownerID,
		WalletType: walletType,
		Balance:    0,
		Currency:   currency,
		Status:     domain.WalletStatusActive,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	walletSchema := domain.MapWalletToSchema(wallet)
	if err := s.walletRepo.Create(ctx, walletSchema); err != nil {
		return nil, fmt.Errorf("failed to create wallet: %w", err)
	}

	s.log.Info("[AUDIT] wallet_created",
		"wallet_id", wallet.ID,
		"type", walletType,
		"owner_type", ownerType,
		"owner_id", ownerID,
		"currency", currency,
		"balance", wallet.Balance,
		"status", wallet.Status,
	)

	return wallet, nil
}

// GetWallet retrieves a wallet by ID
func (s *FinanceServiceImpl) GetWallet(ctx context.Context, walletID uuid.UUID) (*domain.Wallet, error) {
	walletSchema, err := s.walletRepo.GetByID(ctx, walletID)
	if err != nil {
		return nil, fmt.Errorf("failed to get wallet: %w", err)
	}

	if walletSchema == nil {
		return nil, domain.ErrWalletNotFound
	}

	return domain.MapWalletFromSchema(walletSchema), nil
}

// GetBalance returns the current balance of a wallet
func (s *FinanceServiceImpl) GetBalance(ctx context.Context, walletID uuid.UUID) (int64, error) {
	wallet, err := s.GetWallet(ctx, walletID)
	if err != nil {
		return 0, err
	}

	return wallet.Balance, nil
}

// FreezeWallet locks a wallet
func (s *FinanceServiceImpl) FreezeWallet(ctx context.Context, walletID uuid.UUID, reason string) error {
	wallet, err := s.GetWallet(ctx, walletID)
	if err != nil {
		return err
	}

	if err := wallet.Freeze(); err != nil {
		return err
	}

	if err := s.walletRepo.UpdateStatus(ctx, walletID, wallet.Status.String()); err != nil {
		return fmt.Errorf("failed to freeze wallet: %w", err)
	}

	s.log.Warn("[AUDIT] wallet_frozen",
		"wallet_id", walletID,
		"owner_type", wallet.OwnerType,
		"owner_id", wallet.OwnerID,
		"balance", wallet.Balance,
		"currency", wallet.Currency,
		"reason", reason,
	)
	return nil
}

// UnfreezeWallet unlocks a wallet
func (s *FinanceServiceImpl) UnfreezeWallet(ctx context.Context, walletID uuid.UUID) error {
	wallet, err := s.GetWallet(ctx, walletID)
	if err != nil {
		return err
	}

	if err := wallet.Unfreeze(); err != nil {
		return err
	}

	if err := s.walletRepo.UpdateStatus(ctx, walletID, wallet.Status.String()); err != nil {
		return fmt.Errorf("failed to unfreeze wallet: %w", err)
	}

	s.log.Info("[AUDIT] wallet_unfrozen",
		"wallet_id", walletID,
		"owner_type", wallet.OwnerType,
		"owner_id", wallet.OwnerID,
		"balance", wallet.Balance,
		"currency", wallet.Currency,
	)
	return nil
}

// ListUserWallets returns all wallets for a user
func (s *FinanceServiceImpl) ListUserWallets(ctx context.Context, userID uuid.UUID) ([]*domain.Wallet, error) {
	walletsSchema, err := s.walletRepo.ListByOwner(ctx, domain.OwnerTypeUser.String(), userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list wallets: %w", err)
	}

	wallets := make([]*domain.Wallet, len(walletsSchema))
	for i, ws := range walletsSchema {
		wallets[i] = domain.MapWalletFromSchema(ws)
	}

	return wallets, nil
}
