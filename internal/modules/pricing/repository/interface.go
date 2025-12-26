package repository

import (
	"context"
	"hauslet/internal/modules/pricing/repository/schema"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PricingRepository interface {
	// --- Pricing Rule CRUD ---
	CreateRule(ctx context.Context, rule *schema.PricingRule) error
	GetRuleByID(ctx context.Context, id uuid.UUID) (*schema.PricingRule, error)
	UpdateRule(ctx context.Context, rule *schema.PricingRule) error
	DeleteRule(ctx context.Context, id uuid.UUID) error // Soft delete

	// --- Pricing Rule Queries ---
	GetRulesForListing(ctx context.Context, listingID uuid.UUID, activeOnly bool) ([]*schema.PricingRule, error)
	GetActiveRulesForDate(ctx context.Context, listingID uuid.UUID, date time.Time) ([]*schema.PricingRule, error)
	GetRulesByType(ctx context.Context, listingID uuid.UUID, ruleType schema.RuleType) ([]*schema.PricingRule, error)

	// --- Multi-Property Discount CRUD ---
	CreateDiscount(ctx context.Context, discount *schema.MultiPropertyDiscount) error
	GetDiscountByID(ctx context.Context, id uuid.UUID) (*schema.MultiPropertyDiscount, error)
	UpdateDiscount(ctx context.Context, discount *schema.MultiPropertyDiscount) error
	DeleteDiscount(ctx context.Context, id uuid.UUID) error

	// --- Multi-Property Discount Queries ---
	GetDiscountsForOwner(ctx context.Context, ownerID uuid.UUID, activeOnly bool) ([]*schema.MultiPropertyDiscount, error)
	GetDiscountForListings(ctx context.Context, listingIDs []uuid.UUID) (*schema.MultiPropertyDiscount, error)

	// --- Transactions ---
	Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error
}

type PricingRepositoryImpl struct {
	db *gorm.DB
}

func NewPricingRepository(db *gorm.DB) PricingRepository {
	return &PricingRepositoryImpl{db: db}
}
