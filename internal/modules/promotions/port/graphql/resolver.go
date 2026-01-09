package graphql

import (
	"context"
	"fmt"
	"log/slog"

	"hauslet/config"
	"hauslet/internal/modules/promotions/domain"
	"hauslet/internal/modules/promotions/service"
	"hauslet/internal/transport/graph/viewer"

	"github.com/google/uuid"
)

// Resolver handles promotion and subscription GraphQL resolvers
type Resolver struct {
	promotionSvc    service.PromotionService
	subscriptionSvc service.SubscriptionService
	usageSvc        service.UsageService
	config          *config.PromotionYAMLConfig
	log             *slog.Logger
}

// NewResolver creates a new promotions GraphQL resolver
func NewResolver(
	promotionSvc service.PromotionService,
	subscriptionSvc service.SubscriptionService,
	usageSvc service.UsageService,
	config *config.PromotionYAMLConfig,
	log *slog.Logger,
) *Resolver {
	return &Resolver{
		promotionSvc:    promotionSvc,
		subscriptionSvc: subscriptionSvc,
		usageSvc:        usageSvc,
		config:          config,
		log:             log,
	}
}

// =========================================================================
// Mutations
// =========================================================================

// CreatePromotion creates a new paid promotion
func (r *Resolver) CreatePromotion(ctx context.Context, listingID uuid.UUID, promoType domain.PromotionType, duration int) (*CreatePromotionPayload, error) {
	// Get user from context
	userID, err := viewer.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("unauthorized: %w", err)
	}

	// Create promotion via service
	result, err := r.promotionSvc.CreatePromotion(ctx, service.CreatePromotionInput{
		ListingID:  listingID,
		OwnerID:    userID,
		OwnerEmail: "",
		OwnerName:  "",
		Type:       promoType,
		Duration:   duration,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create promotion: %w", err)
	}

	// Convert to GraphQL payload
	paymentURL := result.PaymentURL
	return &CreatePromotionPayload{
		Promotion:  result.Promotion,
		PaymentURL: paymentURL,
		PaymentID:  result.PaymentID,
	}, nil
}

// CreateIncludedPromotion creates a promotion from subscription quota
func (r *Resolver) CreateIncludedPromotion(ctx context.Context, listingID uuid.UUID, promoType domain.PromotionType, duration int) (*domain.ListingPromotion, error) {
	// Get user from context
	userID, err := viewer.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("unauthorized: %w", err)
	}

	// Get user's subscription
	subscription, err := r.subscriptionSvc.GetUserSubscription(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}
	if subscription == nil {
		return nil, fmt.Errorf("no active subscription found")
	}

	// Create included promotion
	promotion, err := r.promotionSvc.CreateIncludedPromotion(ctx, service.CreateIncludedPromotionInput{
		ListingID:      listingID,
		OwnerID:        userID,
		Type:           promoType,
		Duration:       duration,
		SubscriptionID: subscription.ID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create included promotion: %w", err)
	}

	return promotion, nil
}

// CancelPromotion cancels a pending or active promotion
func (r *Resolver) CancelPromotion(ctx context.Context, id uuid.UUID) (*domain.ListingPromotion, error) {
	// Get user from context
	userID, err := viewer.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("unauthorized: %w", err)
	}

	// Get promotion to verify ownership
	promotion, err := r.promotionSvc.GetPromotion(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get promotion: %w", err)
	}
	if promotion.OwnerID != userID {
		return nil, fmt.Errorf("unauthorized: you don't own this promotion")
	}

	// Cancel promotion
	if err := r.promotionSvc.CancelPromotion(ctx, id); err != nil {
		return nil, fmt.Errorf("failed to cancel promotion: %w", err)
	}

	// Return updated promotion
	return r.promotionSvc.GetPromotion(ctx, id)
}

// CreateSubscription creates a new subscription
func (r *Resolver) CreateSubscription(ctx context.Context, planType domain.PlanType, billingCycle domain.BillingCycle, startTrial bool, paymentMethodID *uuid.UUID) (*CreateSubscriptionPayload, error) {
	// Get user from context
	userID, err := viewer.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("unauthorized: %w", err)
	}

	// Create subscription via service
	result, err := r.subscriptionSvc.CreateSubscription(ctx, service.CreateSubscriptionInput{
		UserID:          userID,
		UserEmail:       "",
		UserName:        "",
		PlanType:        planType,
		BillingCycle:    billingCycle,
		StartTrial:      startTrial,
		PaymentMethodID: paymentMethodID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create subscription: %w", err)
	}

	// Convert to GraphQL payload
	return &CreateSubscriptionPayload{
		Subscription: result.Subscription,
		PaymentURL:   &result.PaymentURL,
		PaymentID:    result.PaymentID,
	}, nil
}

// UpgradeSubscription upgrades to a higher tier
func (r *Resolver) UpgradeSubscription(ctx context.Context, subscriptionID uuid.UUID, newPlan domain.PlanType) (*domain.AgentSubscription, error) {
	if err := r.subscriptionSvc.UpgradeSubscription(ctx, subscriptionID, newPlan); err != nil {
		return nil, fmt.Errorf("failed to upgrade subscription: %w", err)
	}
	return r.subscriptionSvc.GetSubscription(ctx, subscriptionID)
}

// DowngradeSubscription downgrades to a lower tier
func (r *Resolver) DowngradeSubscription(ctx context.Context, subscriptionID uuid.UUID, newPlan domain.PlanType) (*domain.AgentSubscription, error) {
	if err := r.subscriptionSvc.DowngradeSubscription(ctx, subscriptionID, newPlan); err != nil {
		return nil, fmt.Errorf("failed to downgrade subscription: %w", err)
	}
	return r.subscriptionSvc.GetSubscription(ctx, subscriptionID)
}

// CancelSubscription cancels a subscription
func (r *Resolver) CancelSubscription(ctx context.Context, subscriptionID uuid.UUID) (*domain.AgentSubscription, error) {
	if err := r.subscriptionSvc.CancelSubscription(ctx, subscriptionID); err != nil {
		return nil, fmt.Errorf("failed to cancel subscription: %w", err)
	}
	return r.subscriptionSvc.GetSubscription(ctx, subscriptionID)
}

// UseIncludedPromotion marks an included promotion as used
func (r *Resolver) UseIncludedPromotion(ctx context.Context, promoType domain.PromotionType) (bool, error) {
	userID, err := viewer.GetUserIDFromContext(ctx)
	if err != nil {
		return false, fmt.Errorf("unauthorized: %w", err)
	}

	if err := r.subscriptionSvc.UseIncludedPromotion(ctx, userID, promoType); err != nil {
		return false, err
	}
	return true, nil
}

// UseOpenHouse marks an open house slot as used
func (r *Resolver) UseOpenHouse(ctx context.Context) (bool, error) {
	userID, err := viewer.GetUserIDFromContext(ctx)
	if err != nil {
		return false, fmt.Errorf("unauthorized: %w", err)
	}

	if err := r.subscriptionSvc.UseOpenHouse(ctx, userID); err != nil {
		return false, err
	}
	return true, nil
}

// UsePrivateShowing marks a private showing slot as used
func (r *Resolver) UsePrivateShowing(ctx context.Context) (bool, error) {
	userID, err := viewer.GetUserIDFromContext(ctx)
	if err != nil {
		return false, fmt.Errorf("unauthorized: %w", err)
	}

	if err := r.subscriptionSvc.UsePrivateShowing(ctx, userID); err != nil {
		return false, err
	}
	return true, nil
}

// =========================================================================
// Queries
// =========================================================================

// GetPromotion retrieves a promotion by ID
func (r *Resolver) GetPromotion(ctx context.Context, id uuid.UUID) (*domain.ListingPromotion, error) {
	return r.promotionSvc.GetPromotion(ctx, id)
}

// ListMyPromotions lists promotions for the current user
func (r *Resolver) ListMyPromotions(ctx context.Context, limit, offset int) ([]*domain.ListingPromotion, error) {
	userID, err := viewer.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("unauthorized: %w", err)
	}

	return r.promotionSvc.ListUserPromotions(ctx, userID, limit, offset)
}

// GetActivePromotionForListing gets the active promotion for a listing
func (r *Resolver) GetActivePromotionForListing(ctx context.Context, listingID uuid.UUID) (*domain.ListingPromotion, error) {
	return r.promotionSvc.GetActivePromotion(ctx, listingID)
}

// GetFeaturedListings gets currently featured listings
func (r *Resolver) GetFeaturedListings(ctx context.Context, limit int) ([]*domain.ListingPromotion, error) {
	return r.promotionSvc.GetFeaturedListings(ctx, limit)
}

// GetPremiumListings gets currently premium listings
func (r *Resolver) GetPremiumListings(ctx context.Context, limit int) ([]*domain.ListingPromotion, error) {
	return r.promotionSvc.GetPremiumListings(ctx, limit)
}

// GetSubscription retrieves a subscription by ID
func (r *Resolver) GetSubscription(ctx context.Context, id uuid.UUID) (*domain.AgentSubscription, error) {
	return r.subscriptionSvc.GetSubscription(ctx, id)
}

// GetMySubscription retrieves the current user's subscription
func (r *Resolver) GetMySubscription(ctx context.Context) (*domain.AgentSubscription, error) {
	userID, err := viewer.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("unauthorized: %w", err)
	}

	return r.subscriptionSvc.GetUserSubscription(ctx, userID)
}

// CanAddListing checks if user can add another listing
func (r *Resolver) CanAddListing(ctx context.Context) (bool, error) {
	userID, err := viewer.GetUserIDFromContext(ctx)
	if err != nil {
		return false, fmt.Errorf("unauthorized: %w", err)
	}

	return r.subscriptionSvc.CanAddListing(ctx, userID)
}

// CanAddPhotos checks if user can add photos to a listing
func (r *Resolver) CanAddPhotos(ctx context.Context, listingID uuid.UUID, photoCount int) (bool, error) {
	userID, err := viewer.GetUserIDFromContext(ctx)
	if err != nil {
		return false, fmt.Errorf("unauthorized: %w", err)
	}

	return r.subscriptionSvc.CanAddPhotos(ctx, userID, listingID, photoCount)
}

// CanUseFeature checks if user has access to a feature
func (r *Resolver) CanUseFeature(ctx context.Context, feature string) (bool, error) {
	userID, err := viewer.GetUserIDFromContext(ctx)
	if err != nil {
		return false, fmt.Errorf("unauthorized: %w", err)
	}

	return r.subscriptionSvc.CanUseFeature(ctx, userID, feature)
}

// CanUseIncludedPromotion checks if user can use an included promotion
func (r *Resolver) CanUseIncludedPromotion(ctx context.Context, promoType domain.PromotionType) (bool, error) {
	userID, err := viewer.GetUserIDFromContext(ctx)
	if err != nil {
		return false, fmt.Errorf("unauthorized: %w", err)
	}

	return r.subscriptionSvc.CanUseIncludedPromotion(ctx, userID, promoType)
}

// CanCreateOpenHouse checks if user can create an open house
func (r *Resolver) CanCreateOpenHouse(ctx context.Context) (*FeatureLimitCheckResult, error) {
	userID, err := viewer.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("unauthorized: %w", err)
	}

	allowed, limit, err := r.subscriptionSvc.CanCreateOpenHouse(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Get current usage
	subscription, err := r.subscriptionSvc.GetUserSubscription(ctx, userID)
	if err != nil || subscription == nil {
		return &FeatureLimitCheckResult{
			Allowed:   false,
			Limit:     0,
			Used:      0,
			Remaining: 0,
		}, nil
	}

	usage, err := r.usageSvc.GetCurrentUsage(ctx, subscription.ID)
	if err != nil || usage == nil {
		return &FeatureLimitCheckResult{
			Allowed:   allowed,
			Limit:     limit,
			Used:      0,
			Remaining: limit,
		}, nil
	}

	used := usage.OpenHousesUsed
	remaining := limit - used
	if remaining < 0 {
		remaining = 0
	}

	return &FeatureLimitCheckResult{
		Allowed:   allowed,
		Limit:     limit,
		Used:      used,
		Remaining: remaining,
	}, nil
}

// CanCreatePrivateShowing checks if user can create a private showing
func (r *Resolver) CanCreatePrivateShowing(ctx context.Context) (*FeatureLimitCheckResult, error) {
	userID, err := viewer.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("unauthorized: %w", err)
	}

	allowed, limit, err := r.subscriptionSvc.CanCreatePrivateShowing(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Get current usage
	subscription, err := r.subscriptionSvc.GetUserSubscription(ctx, userID)
	if err != nil || subscription == nil {
		return &FeatureLimitCheckResult{
			Allowed:   false,
			Limit:     0,
			Used:      0,
			Remaining: 0,
		}, nil
	}

	usage, err := r.usageSvc.GetCurrentUsage(ctx, subscription.ID)
	if err != nil || usage == nil {
		return &FeatureLimitCheckResult{
			Allowed:   allowed,
			Limit:     limit,
			Used:      0,
			Remaining: limit,
		}, nil
	}

	used := usage.PrivateShowingsUsed
	remaining := limit - used
	if remaining < 0 {
		remaining = 0
	}

	return &FeatureLimitCheckResult{
		Allowed:   allowed,
		Limit:     limit,
		Used:      used,
		Remaining: remaining,
	}, nil
}

// GetFeatureLimit gets the limit for a specific feature
func (r *Resolver) GetFeatureLimit(ctx context.Context, feature string) (int, error) {
	userID, err := viewer.GetUserIDFromContext(ctx)
	if err != nil {
		return 0, fmt.Errorf("unauthorized: %w", err)
	}

	return r.subscriptionSvc.GetFeatureLimit(ctx, userID, feature)
}

// GetPlanLimits gets the limits for a specific plan type
func (r *Resolver) GetPlanLimits(ctx context.Context, planType domain.PlanType) (*PlanLimits, error) {
	limits, ok := r.config.GetPlanLimits(planType.String())
	if !ok {
		return nil, fmt.Errorf("unknown plan type: %s", planType)
	}

	features, ok := r.config.GetPlanFeatures(planType.String())
	if !ok {
		features = make(map[string]bool)
	}

	// Convert feature map to slice
	featureList := make([]string, 0, len(features))
	for feature, enabled := range features {
		if enabled {
			featureList = append(featureList, feature)
		}
	}

	return &PlanLimits{
		MaxListings:                limits.MaxListings,
		MaxPhotosPerListing:        limits.MaxPhotosPerListing,
		MaxVideosPerListing:        limits.MaxVirtualTours,
		FeaturedPromotionsPerMonth: limits.IncludedFeaturedPerMonth,
		PremiumPromotionsPerMonth:  limits.IncludedPremiumPerMonth,
		OpenHousesPerMonth:         limits.IncludedOpenHousesPerMonth,
		PrivateShowingsPerMonth:    limits.IncludedPrivateShowingsPerMonth,
		Features:                   featureList,
	}, nil
}

// GetCurrentUsage retrieves the current usage for the user's subscription
func (r *Resolver) GetCurrentUsage(ctx context.Context) (*domain.UsageTracking, error) {
	userID, err := viewer.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("unauthorized: %w", err)
	}

	subscription, err := r.subscriptionSvc.GetUserSubscription(ctx, userID)
	if err != nil || subscription == nil {
		return nil, nil
	}

	return r.usageSvc.GetCurrentUsage(ctx, subscription.ID)
}
