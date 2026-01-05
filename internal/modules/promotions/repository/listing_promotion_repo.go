package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"hauslet/internal/modules/promotions/repository/schema"
)

// ListingPromotionRepo implements ListingPromotionRepository using GORM
type ListingPromotionRepo struct {
	db *gorm.DB
}

// NewListingPromotionRepository creates a new listing promotion repository
func NewListingPromotionRepository(db *gorm.DB) ListingPromotionRepository {
	return &ListingPromotionRepo{db: db}
}

// Create creates a new promotion
func (r *ListingPromotionRepo) Create(ctx context.Context, promo *schema.ListingPromotion) error {
	return r.db.WithContext(ctx).Create(promo).Error
}

// GetByID retrieves a promotion by ID
func (r *ListingPromotionRepo) GetByID(ctx context.Context, id uuid.UUID) (*schema.ListingPromotion, error) {
	var promo schema.ListingPromotion
	err := r.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", id).
		First(&promo).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get promotion by id: %w", err)
	}

	return &promo, nil
}

// GetActiveByListing retrieves the active promotion for a listing
func (r *ListingPromotionRepo) GetActiveByListing(ctx context.Context, listingID uuid.UUID) (*schema.ListingPromotion, error) {
	var promo schema.ListingPromotion
	err := r.db.WithContext(ctx).
		Where("listing_id = ? AND status = ? AND deleted_at IS NULL", listingID, "active").
		First(&promo).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get active promotion: %w", err)
	}

	return &promo, nil
}

// ListByOwner lists promotions owned by a user
func (r *ListingPromotionRepo) ListByOwner(ctx context.Context, ownerID uuid.UUID, limit, offset int) ([]*schema.ListingPromotion, error) {
	var promos []*schema.ListingPromotion
	err := r.db.WithContext(ctx).
		Where("owner_id = ? AND deleted_at IS NULL", ownerID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&promos).Error

	if err != nil {
		return nil, fmt.Errorf("failed to list promotions by owner: %w", err)
	}

	return promos, nil
}

// ListExpiring lists promotions expiring within specified hours
func (r *ListingPromotionRepo) ListExpiring(ctx context.Context, withinHours int) ([]*schema.ListingPromotion, error) {
	expiryThreshold := time.Now().Add(time.Duration(withinHours) * time.Hour)

	var promos []*schema.ListingPromotion
	err := r.db.WithContext(ctx).
		Where("status = ? AND expires_at <= ? AND deleted_at IS NULL", "active", expiryThreshold).
		Find(&promos).Error

	if err != nil {
		return nil, fmt.Errorf("failed to list expiring promotions: %w", err)
	}

	return promos, nil
}

// UpdateStatus updates the status of a promotion
func (r *ListingPromotionRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	err := r.db.WithContext(ctx).
		Model(&schema.ListingPromotion{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":     status,
			"updated_at": time.Now(),
		}).Error

	if err != nil {
		return fmt.Errorf("failed to update promotion status: %w", err)
	}

	return nil
}

// Update updates a promotion
func (r *ListingPromotionRepo) Update(ctx context.Context, promo *schema.ListingPromotion) error {
	promo.UpdatedAt = time.Now()
	err := r.db.WithContext(ctx).Save(promo).Error
	if err != nil {
		return fmt.Errorf("failed to update promotion: %w", err)
	}
	return nil
}

// ListActivePromotions lists active promotions of a specific type
func (r *ListingPromotionRepo) ListActivePromotions(ctx context.Context, promoType string, limit int) ([]*schema.ListingPromotion, error) {
	var promos []*schema.ListingPromotion
	err := r.db.WithContext(ctx).
		Where("type = ? AND status = ? AND deleted_at IS NULL", promoType, "active").
		Order("started_at DESC").
		Limit(limit).
		Find(&promos).Error

	if err != nil {
		return nil, fmt.Errorf("failed to list active promotions: %w", err)
	}

	return promos, nil
}
