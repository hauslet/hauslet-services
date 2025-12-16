package repository

import (
	"context"
	"hauslet/internal/modules/business/repository/schema"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CreateBusiness creates a new business
func (r *BusinessRepositoryImpl) CreateBusiness(ctx context.Context, business *schema.Business) error {
	return r.db.WithContext(ctx).Create(business).Error
}

// GetBusinessByID retrieves a business by ID
func (r *BusinessRepositoryImpl) GetBusinessByID(ctx context.Context, id uuid.UUID) (*schema.Business, error) {
	var business schema.Business
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&business).Error
	if err != nil {
		return nil, err
	}
	return &business, nil
}

// GetBusinessBySlug retrieves a business by slug
func (r *BusinessRepositoryImpl) GetBusinessBySlug(ctx context.Context, slug string) (*schema.Business, error) {
	var business schema.Business
	err := r.db.WithContext(ctx).Where("slug = ?", slug).First(&business).Error
	if err != nil {
		return nil, err
	}
	return &business, nil
}

// UpdateBusiness updates an existing business
func (r *BusinessRepositoryImpl) UpdateBusiness(ctx context.Context, business *schema.Business) error {
	return r.db.WithContext(ctx).Save(business).Error
}

// DeleteBusiness soft deletes a business
func (r *BusinessRepositoryImpl) DeleteBusiness(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&schema.Business{}).Error
}

// HardDeleteBusiness permanently deletes a business
func (r *BusinessRepositoryImpl) HardDeleteBusiness(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Unscoped().Where("id = ?", id).Delete(&schema.Business{}).Error
}

// SlugExists checks if a slug is already taken
func (r *BusinessRepositoryImpl) SlugExists(ctx context.Context, slug string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&schema.Business{}).Where("slug = ?", slug).Count(&count).Error
	return count > 0, err
}

// ListBusinesses lists all businesses with pagination
func (r *BusinessRepositoryImpl) ListBusinesses(ctx context.Context, limit, offset int) ([]*schema.Business, error) {
	var businesses []*schema.Business
	err := r.db.WithContext(ctx).
		Limit(limit).
		Offset(offset).
		Order("created_at DESC").
		Find(&businesses).Error
	return businesses, err
}

// SearchBusinesses searches businesses by name or display name
func (r *BusinessRepositoryImpl) SearchBusinesses(ctx context.Context, query string, limit, offset int) ([]*schema.Business, error) {
	var businesses []*schema.Business
	searchPattern := "%" + query + "%"
	err := r.db.WithContext(ctx).
		Where("name ILIKE ? OR display_name ILIKE ?", searchPattern, searchPattern).
		Limit(limit).
		Offset(offset).
		Order("created_at DESC").
		Find(&businesses).Error
	return businesses, err
}

// Transaction executes a function within a database transaction
func (r *BusinessRepositoryImpl) Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return r.db.WithContext(ctx).Transaction(fn)
}
