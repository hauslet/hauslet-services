package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"hauslet/internal/modules/promotions/repository/schema"
)

// UsageTrackingRepo implements UsageTrackingRepository using GORM
type UsageTrackingRepo struct {
	db *gorm.DB
}

// NewUsageTrackingRepository creates a new usage tracking repository
func NewUsageTrackingRepository(db *gorm.DB) UsageTrackingRepository {
	return &UsageTrackingRepo{db: db}
}

// Create creates a new usage tracking record
func (r *UsageTrackingRepo) Create(ctx context.Context, usage *schema.UsageTracking) error {
	return r.db.WithContext(ctx).Create(usage).Error
}

// GetCurrentPeriod retrieves the current period usage for a subscription
func (r *UsageTrackingRepo) GetCurrentPeriod(ctx context.Context, subscriptionID uuid.UUID) (*schema.UsageTracking, error) {
	now := time.Now()

	var usage schema.UsageTracking
	err := r.db.WithContext(ctx).
		Where("subscription_id = ? AND period_start <= ? AND period_end >= ?", subscriptionID, now, now).
		First(&usage).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get current period usage: %w", err)
	}

	return &usage, nil
}

// Update updates a usage tracking record
func (r *UsageTrackingRepo) Update(ctx context.Context, usage *schema.UsageTracking) error {
	usage.UpdatedAt = time.Now()
	err := r.db.WithContext(ctx).Save(usage).Error
	if err != nil {
		return fmt.Errorf("failed to update usage tracking: %w", err)
	}
	return nil
}

// CreateNewPeriod creates a new usage period
func (r *UsageTrackingRepo) CreateNewPeriod(ctx context.Context, subscriptionID, userID uuid.UUID, start, end time.Time) (*schema.UsageTracking, error) {
	usage := &schema.UsageTracking{
		ID:                  uuid.New(),
		SubscriptionID:      subscriptionID,
		UserID:              userID,
		PeriodStart:         start,
		PeriodEnd:           end,
		FeaturedUsed:        0,
		PremiumUsed:         0,
		OpenHousesUsed:      0,
		PrivateShowingsUsed: 0,
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}

	err := r.db.WithContext(ctx).Create(usage).Error
	if err != nil {
		return nil, fmt.Errorf("failed to create new usage period: %w", err)
	}

	return usage, nil
}

// IncrementFeatured atomically increments featured usage
func (r *UsageTrackingRepo) IncrementFeatured(ctx context.Context, id uuid.UUID) error {
	err := r.db.WithContext(ctx).
		Model(&schema.UsageTracking{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"featured_used": gorm.Expr("featured_used + ?", 1),
			"updated_at":    time.Now(),
		}).Error

	if err != nil {
		return fmt.Errorf("failed to increment featured usage: %w", err)
	}

	return nil
}

// IncrementPremium atomically increments premium usage
func (r *UsageTrackingRepo) IncrementPremium(ctx context.Context, id uuid.UUID) error {
	err := r.db.WithContext(ctx).
		Model(&schema.UsageTracking{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"premium_used": gorm.Expr("premium_used + ?", 1),
			"updated_at":   time.Now(),
		}).Error

	if err != nil {
		return fmt.Errorf("failed to increment premium usage: %w", err)
	}

	return nil
}

// IncrementOpenHouse atomically increments open house usage
func (r *UsageTrackingRepo) IncrementOpenHouse(ctx context.Context, id uuid.UUID) error {
	err := r.db.WithContext(ctx).
		Model(&schema.UsageTracking{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"open_houses_used": gorm.Expr("open_houses_used + ?", 1),
			"updated_at":       time.Now(),
		}).Error

	if err != nil {
		return fmt.Errorf("failed to increment open house usage: %w", err)
	}

	return nil
}

// IncrementPrivateShowing atomically increments private showing usage
func (r *UsageTrackingRepo) IncrementPrivateShowing(ctx context.Context, id uuid.UUID) error {
	err := r.db.WithContext(ctx).
		Model(&schema.UsageTracking{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"private_showings_used": gorm.Expr("private_showings_used + ?", 1),
			"updated_at":            time.Now(),
		}).Error

	if err != nil {
		return fmt.Errorf("failed to increment private showing usage: %w", err)
	}

	return nil
}
