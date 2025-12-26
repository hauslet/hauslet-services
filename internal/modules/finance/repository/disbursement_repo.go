package repository

import (
	"context"
	"hauslet/internal/modules/finance/repository/schema"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type DisbursementRepositoryImpl struct {
	db *gorm.DB
}

func NewDisbursementRepository(db *gorm.DB) DisbursementRepository {
	return &DisbursementRepositoryImpl{db: db}
}

func (r *DisbursementRepositoryImpl) Create(ctx context.Context, disbursement *schema.Disbursement) error {
	return r.db.WithContext(ctx).Create(disbursement).Error
}

func (r *DisbursementRepositoryImpl) GetByID(ctx context.Context, id uuid.UUID) (*schema.Disbursement, error) {
	var disbursement schema.Disbursement
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&disbursement).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &disbursement, nil
}

func (r *DisbursementRepositoryImpl) GetByTransferCode(ctx context.Context, code string) (*schema.Disbursement, error) {
	var disbursement schema.Disbursement
	err := r.db.WithContext(ctx).Where("transfer_code = ?", code).First(&disbursement).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &disbursement, nil
}

func (r *DisbursementRepositoryImpl) UpdateStatus(ctx context.Context, id uuid.UUID, status string, response *string) error {
	updates := map[string]interface{}{
		"status":     status,
		"updated_at": gorm.Expr("NOW()"),
	}

	if response != nil {
		updates["provider_response"] = *response
	}

	if status == schema.DisbursementStatusCompleted {
		updates["completed_at"] = gorm.Expr("NOW()")
	}

	return r.db.WithContext(ctx).
		Model(&schema.Disbursement{}).
		Where("id = ?", id).
		Updates(updates).Error
}

func (r *DisbursementRepositoryImpl) IncrementAttempts(ctx context.Context, id uuid.UUID, nextRetry *time.Time) error {
	updates := map[string]interface{}{
		"attempts":   gorm.Expr("attempts + 1"),
		"updated_at": gorm.Expr("NOW()"),
	}

	if nextRetry != nil {
		updates["next_retry_at"] = nextRetry
	}

	return r.db.WithContext(ctx).
		Model(&schema.Disbursement{}).
		Where("id = ?", id).
		Updates(updates).Error
}

func (r *DisbursementRepositoryImpl) ListPendingRetries(ctx context.Context) ([]*schema.Disbursement, error) {
	var disbursements []*schema.Disbursement
	now := time.Now()

	err := r.db.WithContext(ctx).
		Where("status = ? AND next_retry_at IS NOT NULL AND next_retry_at <= ?",
			schema.DisbursementStatusFailed, now).
		Order("next_retry_at ASC").
		Find(&disbursements).Error

	if err != nil {
		return nil, err
	}
	return disbursements, nil
}

func (r *DisbursementRepositoryImpl) ListByWallet(ctx context.Context, walletID uuid.UUID, limit, offset int) ([]*schema.Disbursement, error) {
	var disbursements []*schema.Disbursement

	query := r.db.WithContext(ctx).
		Where("wallet_id = ?", walletID).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	err := query.Find(&disbursements).Error
	if err != nil {
		return nil, err
	}
	return disbursements, nil
}
