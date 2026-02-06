package repository

import (
	"context"
	"hauslet/internal/modules/finance/domain"
	"hauslet/internal/modules/finance/repository/schema"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PenaltyDebtRepositoryImpl implements PenaltyDebtRepository
// for tracking unpaid host cancellation penalties.
type PenaltyDebtRepositoryImpl struct {
	db *gorm.DB
}

// NewPenaltyDebtRepository creates a new penalty debt repository.
func NewPenaltyDebtRepository(db *gorm.DB) PenaltyDebtRepository {
	return &PenaltyDebtRepositoryImpl{db: db}
}

// WithTx returns a new repository instance using the provided transaction.
func (r *PenaltyDebtRepositoryImpl) WithTx(tx *gorm.DB) PenaltyDebtRepository {
	return &PenaltyDebtRepositoryImpl{db: tx}
}

// Create inserts a new penalty debt record.
func (r *PenaltyDebtRepositoryImpl) Create(ctx context.Context, debt *schema.PenaltyDebt) error {
	return r.db.WithContext(ctx).Create(debt).Error
}

// GetByBookingID retrieves a penalty debt by booking ID.
func (r *PenaltyDebtRepositoryImpl) GetByBookingID(ctx context.Context, bookingID uuid.UUID) (*schema.PenaltyDebt, error) {
	var debt schema.PenaltyDebt
	if err := r.db.WithContext(ctx).Where("booking_id = ?", bookingID).First(&debt).Error; err != nil {
		return nil, err
	}
	return &debt, nil
}

// ListOutstandingByHost lists outstanding debts for a host.
func (r *PenaltyDebtRepositoryImpl) ListOutstandingByHost(ctx context.Context, hostID uuid.UUID) ([]*schema.PenaltyDebt, error) {
	var debts []*schema.PenaltyDebt
	err := r.db.WithContext(ctx).
		Where("host_id = ? AND status = ? AND outstanding_amount > 0", hostID, domain.PenaltyDebtStatusPending).
		Order("created_at ASC").
		Find(&debts).Error
	if err != nil {
		return nil, err
	}
	return debts, nil
}

// ListOutstanding lists outstanding penalty debts across all hosts.
func (r *PenaltyDebtRepositoryImpl) ListOutstanding(ctx context.Context, limit int) ([]*schema.PenaltyDebt, error) {
	if limit <= 0 {
		limit = 200
	}
	var debts []*schema.PenaltyDebt
	err := r.db.WithContext(ctx).
		Where("status = ? AND outstanding_amount > 0", domain.PenaltyDebtStatusPending).
		Order("created_at ASC").
		Limit(limit).
		Find(&debts).Error
	if err != nil {
		return nil, err
	}
	return debts, nil
}

// Update updates an existing penalty debt record.
func (r *PenaltyDebtRepositoryImpl) Update(ctx context.Context, debt *schema.PenaltyDebt) error {
	return r.db.WithContext(ctx).Save(debt).Error
}
