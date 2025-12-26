package repository

import (
	"context"
	"hauslet/internal/modules/pricing/repository/schema"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// --- Pricing Rule CRUD ---

func (r *PricingRepositoryImpl) CreateRule(ctx context.Context, rule *schema.PricingRule) error {
	return r.db.WithContext(ctx).Create(rule).Error
}

func (r *PricingRepositoryImpl) GetRuleByID(ctx context.Context, id uuid.UUID) (*schema.PricingRule, error) {
	var rule schema.PricingRule
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&rule).Error
	if err != nil {
		return nil, err
	}
	return &rule, nil
}

func (r *PricingRepositoryImpl) UpdateRule(ctx context.Context, rule *schema.PricingRule) error {
	return r.db.WithContext(ctx).Save(rule).Error
}

func (r *PricingRepositoryImpl) DeleteRule(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&schema.PricingRule{}).Error
}

// --- Pricing Rule Queries ---

func (r *PricingRepositoryImpl) GetRulesForListing(ctx context.Context, listingID uuid.UUID, activeOnly bool) ([]*schema.PricingRule, error) {
	query := r.db.WithContext(ctx).Where("listing_id = ?", listingID)

	if activeOnly {
		query = query.Where("active = ?", true)
	}

	var rules []*schema.PricingRule
	err := query.Order("priority DESC, created_at ASC").Find(&rules).Error
	return rules, err
}

func (r *PricingRepositoryImpl) GetActiveRulesForDate(ctx context.Context, listingID uuid.UUID, date time.Time) ([]*schema.PricingRule, error) {
	query := r.db.WithContext(ctx).
		Where("listing_id = ?", listingID).
		Where("active = ?", true).
		Where("(start_date IS NULL OR start_date <= ?)", date).
		Where("(end_date IS NULL OR end_date >= ?)", date)

	var rules []*schema.PricingRule
	err := query.Order("priority DESC, created_at ASC").Find(&rules).Error
	return rules, err
}

func (r *PricingRepositoryImpl) GetRulesByType(ctx context.Context, listingID uuid.UUID, ruleType schema.RuleType) ([]*schema.PricingRule, error) {
	var rules []*schema.PricingRule
	err := r.db.WithContext(ctx).
		Where("listing_id = ?", listingID).
		Where("rule_type = ?", ruleType).
		Where("active = ?", true).
		Order("priority DESC").
		Find(&rules).Error
	return rules, err
}

// --- Multi-Property Discount CRUD ---

func (r *PricingRepositoryImpl) CreateDiscount(ctx context.Context, discount *schema.MultiPropertyDiscount) error {
	return r.db.WithContext(ctx).Create(discount).Error
}

func (r *PricingRepositoryImpl) GetDiscountByID(ctx context.Context, id uuid.UUID) (*schema.MultiPropertyDiscount, error) {
	var discount schema.MultiPropertyDiscount
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&discount).Error
	if err != nil {
		return nil, err
	}
	return &discount, nil
}

func (r *PricingRepositoryImpl) UpdateDiscount(ctx context.Context, discount *schema.MultiPropertyDiscount) error {
	return r.db.WithContext(ctx).Save(discount).Error
}

func (r *PricingRepositoryImpl) DeleteDiscount(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&schema.MultiPropertyDiscount{}).Error
}

// --- Multi-Property Discount Queries ---

func (r *PricingRepositoryImpl) GetDiscountsForOwner(ctx context.Context, ownerID uuid.UUID, activeOnly bool) ([]*schema.MultiPropertyDiscount, error) {
	query := r.db.WithContext(ctx).Where("owner_id = ?", ownerID)

	if activeOnly {
		query = query.Where("active = ?", true)
	}

	var discounts []*schema.MultiPropertyDiscount
	err := query.Order("created_at DESC").Find(&discounts).Error
	return discounts, err
}

func (r *PricingRepositoryImpl) GetDiscountForListings(ctx context.Context, listingIDs []uuid.UUID) (*schema.MultiPropertyDiscount, error) {
	var discount schema.MultiPropertyDiscount

	// Find discount where listing_ids contains all of the provided listing IDs
	err := r.db.WithContext(ctx).
		Where("active = ?", true).
		Where("listing_ids @> ?", listingIDs).
		First(&discount).Error

	if err != nil {
		return nil, err
	}

	return &discount, nil
}

// --- Transactions ---

func (r *PricingRepositoryImpl) Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return r.db.WithContext(ctx).Transaction(fn)
}
