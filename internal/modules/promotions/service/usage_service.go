package service

import (
	"context"
	"fmt"
	"log/slog"

	"hauslet/config"
	"hauslet/internal/modules/promotions/domain"
	"hauslet/internal/modules/promotions/repository"
	"hauslet/internal/modules/promotions/repository/schema"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UsageServiceImpl implements UsageService
type UsageServiceImpl struct {
	usageRepo repository.UsageTrackingRepository
	config    *config.PromotionYAMLConfig
	db        *gorm.DB
	log       *slog.Logger
}

// NewUsageService creates a new usage service
func NewUsageService(
	usageRepo repository.UsageTrackingRepository,
	config *config.PromotionYAMLConfig,
	db *gorm.DB,
	log *slog.Logger,
) UsageService {
	return &UsageServiceImpl{
		usageRepo: usageRepo,
		config:    config,
		db:        db,
		log:       log,
	}
}

// GetCurrentUsage retrieves the current usage period for a subscription
func (s *UsageServiceImpl) GetCurrentUsage(ctx context.Context, subscriptionID uuid.UUID) (*domain.UsageTracking, error) {
	usageSchema, err := s.usageRepo.GetCurrentPeriod(ctx, subscriptionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get current usage: %w", err)
	}

	if usageSchema == nil {
		return nil, nil
	}

	return schema.MapUsageTrackingFromSchema(usageSchema), nil
}

// GetOrCreateCurrentUsage gets or creates the current usage period
func (s *UsageServiceImpl) GetOrCreateCurrentUsage(ctx context.Context, subscriptionID, userID uuid.UUID) (*domain.UsageTracking, error) {
	// Try to get existing usage
	usage, err := s.GetCurrentUsage(ctx, subscriptionID)
	if err != nil {
		return nil, err
	}

	if usage != nil {
		return usage, nil
	}

	// Create new usage period
	periodStart, periodEnd := getCurrentBillingPeriod(s.config)

	usageSchema, err := s.usageRepo.CreateNewPeriod(ctx, subscriptionID, userID, periodStart, periodEnd)
	if err != nil {
		return nil, fmt.Errorf("failed to create usage period: %w", err)
	}

	return schema.MapUsageTrackingFromSchema(usageSchema), nil
}

// IncrementUsage increments usage for a specific type
func (s *UsageServiceImpl) IncrementUsage(ctx context.Context, subscriptionID uuid.UUID, usageType domain.UsageType) error {
	// Get current usage
	usageSchema, err := s.usageRepo.GetCurrentPeriod(ctx, subscriptionID)
	if err != nil {
		return fmt.Errorf("failed to get current usage: %w", err)
	}

	if usageSchema == nil {
		return fmt.Errorf("no usage tracking found for subscription %s", subscriptionID)
	}

	// Increment based on type
	switch usageType {
	case domain.UsageTypeFeatured:
		err = s.usageRepo.IncrementFeatured(ctx, usageSchema.ID)
	case domain.UsageTypePremium:
		err = s.usageRepo.IncrementPremium(ctx, usageSchema.ID)
	case domain.UsageTypeOpenHouse:
		err = s.usageRepo.IncrementOpenHouse(ctx, usageSchema.ID)
	case domain.UsageTypePrivateShowing:
		err = s.usageRepo.IncrementPrivateShowing(ctx, usageSchema.ID)
	default:
		return fmt.Errorf("unknown usage type: %s", usageType)
	}

	if err != nil {
		return fmt.Errorf("failed to increment usage: %w", err)
	}

	s.log.Info("incremented usage",
		"subscription_id", subscriptionID,
		"usage_type", usageType,
	)

	return nil
}

// ResetUsage creates a new usage period (called on billing cycle)
func (s *UsageServiceImpl) ResetUsage(ctx context.Context, subscriptionID, userID uuid.UUID) error {
	periodStart, periodEnd := getCurrentBillingPeriod(s.config)

	// Create new period
	_, err := s.usageRepo.CreateNewPeriod(ctx, subscriptionID, userID, periodStart, periodEnd)
	if err != nil {
		return fmt.Errorf("failed to reset usage: %w", err)
	}

	s.log.Info("reset usage for new billing period",
		"subscription_id", subscriptionID,
		"period_start", periodStart,
		"period_end", periodEnd,
	)

	return nil
}
