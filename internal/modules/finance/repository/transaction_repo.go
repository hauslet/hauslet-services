package repository

import (
	"context"
	"hauslet/internal/modules/finance/repository/schema"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TransactionRepositoryImpl struct {
	db *gorm.DB
}

func NewTransactionRepository(db *gorm.DB) TransactionRepository {
	return &TransactionRepositoryImpl{db: db}
}

func (r *TransactionRepositoryImpl) Create(ctx context.Context, tx *schema.Transaction) error {
	return r.db.WithContext(ctx).Create(tx).Error
}

func (r *TransactionRepositoryImpl) GetByID(ctx context.Context, id uuid.UUID) (*schema.Transaction, error) {
	var tx schema.Transaction
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&tx).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &tx, nil
}

func (r *TransactionRepositoryImpl) GetByResource(ctx context.Context, resourceType string, resourceID uuid.UUID) (*schema.Transaction, error) {
	var tx schema.Transaction
	err := r.db.WithContext(ctx).
		Where("resource_type = ? AND resource_id = ?", resourceType, resourceID).
		Order("created_at DESC").
		First(&tx).Error

	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &tx, nil
}

func (r *TransactionRepositoryImpl) ListByResource(ctx context.Context, resourceType string, resourceID uuid.UUID) ([]*schema.Transaction, error) {
	var txs []*schema.Transaction
	err := r.db.WithContext(ctx).
		Where("resource_type = ? AND resource_id = ?", resourceType, resourceID).
		Order("created_at DESC").
		Find(&txs).Error

	if err != nil {
		return nil, err
	}
	return txs, nil
}

func (r *TransactionRepositoryImpl) UpdateStatus(ctx context.Context, id uuid.UUID, status string, errorMsg *string) error {
	updates := map[string]interface{}{
		"status":     status,
		"updated_at": gorm.Expr("NOW()"),
	}

	if errorMsg != nil {
		updates["error_message"] = *errorMsg
	}

	return r.db.WithContext(ctx).
		Model(&schema.Transaction{}).
		Where("id = ?", id).
		Updates(updates).Error
}

func (r *TransactionRepositoryImpl) ListByStatus(ctx context.Context, status string, limit int) ([]*schema.Transaction, error) {
	var txs []*schema.Transaction

	query := r.db.WithContext(ctx).
		Where("status = ?", status).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Find(&txs).Error
	if err != nil {
		return nil, err
	}
	return txs, nil
}

// WithTx returns a new repository instance using the provided transaction
func (r *TransactionRepositoryImpl) WithTx(tx *gorm.DB) TransactionRepository {
	return &TransactionRepositoryImpl{db: tx}
}
