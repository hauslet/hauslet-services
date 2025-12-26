package repository

import (
	"context"
	"hauslet/internal/modules/finance/repository/schema"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type LedgerRepositoryImpl struct {
	db *gorm.DB
}

func NewLedgerRepository(db *gorm.DB) LedgerRepository {
	return &LedgerRepositoryImpl{db: db}
}

func (r *LedgerRepositoryImpl) CreateEntry(ctx context.Context, entry *schema.LedgerEntry) error {
	return r.db.WithContext(ctx).Create(entry).Error
}

func (r *LedgerRepositoryImpl) CreateEntries(ctx context.Context, entries []*schema.LedgerEntry) error {
	// Use transaction to ensure atomic creation
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, entry := range entries {
			if err := tx.Create(entry).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *LedgerRepositoryImpl) GetByReference(ctx context.Context, reference string) (*schema.LedgerEntry, error) {
	var entry schema.LedgerEntry
	err := r.db.WithContext(ctx).Where("reference = ?", reference).First(&entry).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &entry, nil
}

func (r *LedgerRepositoryImpl) ListByTransaction(ctx context.Context, txID uuid.UUID) ([]*schema.LedgerEntry, error) {
	var entries []*schema.LedgerEntry
	err := r.db.WithContext(ctx).
		Where("transaction_id = ?", txID).
		Order("created_at ASC").
		Find(&entries).Error

	if err != nil {
		return nil, err
	}
	return entries, nil
}

func (r *LedgerRepositoryImpl) ListByResource(ctx context.Context, resourceType string, resourceID uuid.UUID) ([]*schema.LedgerEntry, error) {
	var entries []*schema.LedgerEntry
	err := r.db.WithContext(ctx).
		Where("resource_type = ? AND resource_id = ?", resourceType, resourceID).
		Order("created_at DESC").
		Find(&entries).Error

	if err != nil {
		return nil, err
	}
	return entries, nil
}

func (r *LedgerRepositoryImpl) ListByWallet(ctx context.Context, walletID uuid.UUID, limit, offset int) ([]*schema.LedgerEntry, error) {
	var entries []*schema.LedgerEntry

	query := r.db.WithContext(ctx).
		Where("debit_wallet_id = ? OR credit_wallet_id = ?", walletID, walletID).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	err := query.Find(&entries).Error
	if err != nil {
		return nil, err
	}
	return entries, nil
}
