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
	// Create all entries in bulk - transaction is handled by the caller
	return r.db.WithContext(ctx).Create(entries).Error
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

// FindImbalancedTransactions finds internal transactions where debits don't equal credits
// External transactions (charges with only credits, refunds with only debits) are excluded
func (r *LedgerRepositoryImpl) FindImbalancedTransactions(ctx context.Context) ([]*schema.TransactionBalance, error) {
	var imbalances []*schema.TransactionBalance
	err := r.db.WithContext(ctx).Raw(`
		SELECT
			transaction_id,
			COALESCE(SUM(CASE WHEN debit_wallet_id IS NOT NULL THEN amount ELSE 0 END), 0) as total_debit,
			COALESCE(SUM(CASE WHEN credit_wallet_id IS NOT NULL THEN amount ELSE 0 END), 0) as total_credit
		FROM ledger_entries
		GROUP BY transaction_id
		HAVING
			-- Only validate internal transactions (those with BOTH debit and credit entries)
			-- External transactions (charges: only credits, refunds: only debits) are excluded
			COUNT(CASE WHEN debit_wallet_id IS NOT NULL THEN 1 END) > 0
			AND COUNT(CASE WHEN credit_wallet_id IS NOT NULL THEN 1 END) > 0
			-- Check if debits and credits don't balance
			AND SUM(CASE WHEN debit_wallet_id IS NOT NULL THEN amount ELSE 0 END) !=
			    SUM(CASE WHEN credit_wallet_id IS NOT NULL THEN amount ELSE 0 END)
	`).Scan(&imbalances).Error

	if err != nil {
		return nil, err
	}
	return imbalances, nil
}

// WithTx returns a new repository instance using the provided transaction
func (r *LedgerRepositoryImpl) WithTx(tx *gorm.DB) LedgerRepository {
	return &LedgerRepositoryImpl{db: tx}
}
