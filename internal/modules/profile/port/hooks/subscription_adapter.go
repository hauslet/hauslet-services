package hooks

import (
	"context"
	"fmt"

	"hauslet/internal/modules/promotions/domain"
	promotionservice "hauslet/internal/modules/promotions/service"
	profileservice "hauslet/internal/modules/profile/service"

	"github.com/google/uuid"
)

// ProfileSubscriptionAdapter implements SubscriptionAdapter for profile service
// Follows the adapter pattern: adapters in consuming module's port/hooks directory
type ProfileSubscriptionAdapter struct {
	subscriptionSvc promotionservice.SubscriptionService
}

// NewProfileSubscriptionAdapter creates a new subscription adapter
func NewProfileSubscriptionAdapter(subscriptionSvc promotionservice.SubscriptionService) profileservice.SubscriptionAdapter {
	return &ProfileSubscriptionAdapter{
		subscriptionSvc: subscriptionSvc,
	}
}

// GetOrCreateFreeSubscription gets or creates a free subscription
func (a *ProfileSubscriptionAdapter) GetOrCreateFreeSubscription(ctx context.Context, userID uuid.UUID) error {
	// Check if user already has an active subscription
	existing, err := a.subscriptionSvc.GetUserSubscription(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to check existing subscription: %w", err)
	}

	// If user has active subscription, return without creating a new one
	if existing != nil && existing.IsActive() {
		return nil
	}

	// Create free subscription
	_, err = a.subscriptionSvc.CreateSubscription(ctx, promotionservice.CreateSubscriptionInput{
		UserID:       userID,
		PlanType:     domain.PlanTypeFree,
		BillingCycle: domain.BillingCycleMonthly,
		StartTrial:   false, // Free plan doesn't need trial
	})
	if err != nil {
		return fmt.Errorf("failed to create free subscription: %w", err)
	}

	return nil
}

// HasActiveSubscription checks if user has active subscription
func (a *ProfileSubscriptionAdapter) HasActiveSubscription(ctx context.Context, userID uuid.UUID) (bool, error) {
	subscription, err := a.subscriptionSvc.GetUserSubscription(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("failed to get user subscription: %w", err)
	}
	return subscription != nil && subscription.IsActive(), nil
}
