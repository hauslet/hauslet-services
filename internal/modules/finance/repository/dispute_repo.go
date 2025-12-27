package repository

import (
	"context"
	"hauslet/internal/modules/finance/repository/schema"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// DisputeRepositoryImpl implements DisputeRepository
type DisputeRepositoryImpl struct {
	db *gorm.DB
}

// NewDisputeRepository creates a new dispute repository
func NewDisputeRepository(db *gorm.DB) DisputeRepository {
	return &DisputeRepositoryImpl{db: db}
}

// WithTx returns a new repository instance using the provided transaction
func (r *DisputeRepositoryImpl) WithTx(tx *gorm.DB) DisputeRepository {
	return &DisputeRepositoryImpl{db: tx}
}

// Create creates a new dispute record
func (r *DisputeRepositoryImpl) Create(ctx context.Context, dispute *schema.Dispute) error {
	return r.db.WithContext(ctx).Create(dispute).Error
}

// GetByID retrieves a dispute by ID
func (r *DisputeRepositoryImpl) GetByID(ctx context.Context, id uuid.UUID) (*schema.Dispute, error) {
	var dispute schema.Dispute
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&dispute).Error; err != nil {
		return nil, err
	}
	return &dispute, nil
}

// GetByBookingID retrieves a dispute by booking ID
func (r *DisputeRepositoryImpl) GetByBookingID(ctx context.Context, bookingID uuid.UUID) (*schema.Dispute, error) {
	var dispute schema.Dispute
	if err := r.db.WithContext(ctx).Where("booking_id = ?", bookingID).First(&dispute).Error; err != nil {
		return nil, err
	}
	return &dispute, nil
}

// Update updates an existing dispute
func (r *DisputeRepositoryImpl) Update(ctx context.Context, dispute *schema.Dispute) error {
	return r.db.WithContext(ctx).Save(dispute).Error
}

// UpdateStatus updates the status of a dispute
func (r *DisputeRepositoryImpl) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	return r.db.WithContext(ctx).
		Model(&schema.Dispute{}).
		Where("id = ?", id).
		Update("status", status).
		Error
}

// ListByStatus retrieves disputes by status with pagination
func (r *DisputeRepositoryImpl) ListByStatus(ctx context.Context, status string, limit, offset int) ([]*schema.Dispute, error) {
	var disputes []*schema.Dispute
	err := r.db.WithContext(ctx).
		Where("status = ?", status).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&disputes).Error
	if err != nil {
		return nil, err
	}
	return disputes, nil
}

// ListByFiledBy retrieves disputes filed by a specific user with pagination
func (r *DisputeRepositoryImpl) ListByFiledBy(ctx context.Context, filedByID uuid.UUID, limit, offset int) ([]*schema.Dispute, error) {
	var disputes []*schema.Dispute
	err := r.db.WithContext(ctx).
		Where("filed_by_id = ?", filedByID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&disputes).Error
	if err != nil {
		return nil, err
	}
	return disputes, nil
}

// ListAll retrieves all disputes with pagination
func (r *DisputeRepositoryImpl) ListAll(ctx context.Context, limit, offset int) ([]*schema.Dispute, error) {
	var disputes []*schema.Dispute
	err := r.db.WithContext(ctx).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&disputes).Error
	if err != nil {
		return nil, err
	}
	return disputes, nil
}
