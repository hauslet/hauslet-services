package repository

import (
	"context"
	"hauslet/internal/modules/finance/repository/schema"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type WalletRepositoryImpl struct {
	db *gorm.DB
}

func NewWalletRepository(db *gorm.DB) WalletRepository {
	return &WalletRepositoryImpl{db: db}
}

func (r *WalletRepositoryImpl) Create(ctx context.Context, wallet *schema.Wallet) error {
	return r.db.WithContext(ctx).Create(wallet).Error
}

func (r *WalletRepositoryImpl) GetByID(ctx context.Context, id uuid.UUID) (*schema.Wallet, error) {
	var wallet schema.Wallet
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&wallet).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &wallet, nil
}

func (r *WalletRepositoryImpl) GetByOwner(ctx context.Context, ownerType string, ownerID uuid.UUID, walletType string) (*schema.Wallet, error) {
	var wallet schema.Wallet
	err := r.db.WithContext(ctx).
		Where("owner_type = ? AND owner_id = ? AND wallet_type = ?", ownerType, ownerID, walletType).
		First(&wallet).Error

	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &wallet, nil
}

func (r *WalletRepositoryImpl) UpdateBalance(ctx context.Context, id uuid.UUID, newBalance int64) error {
	return r.db.WithContext(ctx).
		Model(&schema.Wallet{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"balance":    newBalance,
			"updated_at": gorm.Expr("NOW()"),
		}).Error
}

func (r *WalletRepositoryImpl) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	return r.db.WithContext(ctx).
		Model(&schema.Wallet{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":     status,
			"updated_at": gorm.Expr("NOW()"),
		}).Error
}

func (r *WalletRepositoryImpl) ListByOwner(ctx context.Context, ownerType string, ownerID uuid.UUID) ([]*schema.Wallet, error) {
	var wallets []*schema.Wallet
	err := r.db.WithContext(ctx).
		Where("owner_type = ? AND owner_id = ?", ownerType, ownerID).
		Order("created_at DESC").
		Find(&wallets).Error

	if err != nil {
		return nil, err
	}
	return wallets, nil
}

// WithTx returns a new repository instance using the provided transaction
func (r *WalletRepositoryImpl) WithTx(tx *gorm.DB) WalletRepository {
	return &WalletRepositoryImpl{db: tx}
}
