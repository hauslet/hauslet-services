package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"hauslet/config"
	paymentDomain "hauslet/internal/modules/payments/domain"
	paymentService "hauslet/internal/modules/payments/service"
	"hauslet/internal/modules/promotions/domain"
	"hauslet/internal/modules/promotions/repository"
	"hauslet/internal/modules/promotions/repository/schema"
	"hauslet/internal/platform/payment"
)

// SubscriptionServiceImpl implements SubscriptionService
type SubscriptionServiceImpl struct {
	subscriptionRepo repository.AgentSubscriptionRepository
	usageService     UsageService
	paymentService   paymentService.PaymentService
	config           *config.PromotionYAMLConfig
	db               *gorm.DB
	log              *slog.Logger
}

// NewSubscriptionService creates a new subscription service
func NewSubscriptionService(
	subscriptionRepo repository.AgentSubscriptionRepository,
	usageService UsageService,
	paymentService paymentService.PaymentService,
	config *config.PromotionYAMLConfig,
	db *gorm.DB,
	log *slog.Logger,
) SubscriptionService {
	return &SubscriptionServiceImpl{
		subscriptionRepo: subscriptionRepo,
		usageService:     usageService,
		paymentService:   paymentService,
		config:           config,
		db:               db,
		log:              log,
	}
}

// CreateSubscription creates a new subscription and initiates payment (or starts trial)
func (s *SubscriptionServiceImpl) CreateSubscription(ctx context.Context, input CreateSubscriptionInput) (*CreateSubscriptionResult, error) {
	// Load plan configuration
	planConfig, err := loadPlanConfig(s.config, input.PlanType, input.BillingCycle)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	var status domain.SubscriptionStatus
	var trialEndsAt *time.Time
	var nextBillingDate *time.Time
	var paymentID *uuid.UUID
	var paymentURL string

	subscriptionID := uuid.New()
	isFreePlan := planConfig.Amount == 0

	// For paid, non-trial subscriptions, create payment first
	if isFreePlan {
		status = domain.SubscriptionStatusActive
	} else if !input.StartTrial {
		paymentInput := paymentDomain.CreatePaymentInput{
			Amount:       planConfig.Amount,
			Currency:     payment.Currency(planConfig.Currency),
			Market:       paymentDomain.MarketNigeria,
			PayerID:      input.UserID,
			PayerEmail:   input.UserEmail,
			PayerName:    input.UserName,
			ResourceType: paymentDomain.ResourceTypeSubscription,
			ResourceID:   &subscriptionID,
			Description:  fmt.Sprintf("%s subscription - %s billing", input.PlanType, input.BillingCycle),
			Metadata: map[string]string{
				"subscription_id": subscriptionID.String(),
				"plan_type":       input.PlanType.String(),
				"billing_cycle":   input.BillingCycle.String(),
			},
		}

		pmt, err := s.paymentService.CreatePayment(ctx, paymentInput)
		if err != nil {
			return nil, fmt.Errorf("failed to create payment for subscription: %w", err)
		}

		paymentID = &pmt.ID
		if pmt.RedirectURL != nil {
			paymentURL = *pmt.RedirectURL
		}

		// Subscription starts pending until payment succeeds
		status = domain.SubscriptionStatusPending
		nextBilling := calculateNextBillingDate(now, input.BillingCycle)
		nextBillingDate = &nextBilling
	} else {
		// Trial subscription
		status = domain.SubscriptionStatusTrial
		trialEnd := calculateTrialEndDate(s.config, now)
		trialEndsAt = &trialEnd
		nextBillingDate = &trialEnd
	}

	subscription := &domain.AgentSubscription{
		ID:                              subscriptionID,
		UserID:                          input.UserID,
		UserEmail:                       input.UserEmail,
		UserName:                        input.UserName,
		PlanType:                        input.PlanType,
		Status:                          status,
		BillingCycle:                    input.BillingCycle,
		Amount:                          planConfig.Amount,
		Currency:                        planConfig.Currency,
		NextBillingDate:                 nextBillingDate,
		TrialEndsAt:                     trialEndsAt,
		StartedAt:                       now,
		MaxListings:                     planConfig.MaxListings,
		MaxPhotosPerListing:             planConfig.MaxPhotosPerListing,
		MaxVirtualTours:                 planConfig.MaxVirtualTours,
		IncludedFeaturedPerMonth:        planConfig.IncludedFeaturedPerMonth,
		IncludedPremiumPerMonth:         planConfig.IncludedPremiumPerMonth,
		IncludedOpenHousesPerMonth:      planConfig.IncludedOpenHousesPerMonth,
		IncludedPrivateShowingsPerMonth: planConfig.IncludedPrivateShowingsPerMonth,
		Features:                        planConfig.Features,
		CreatedAt:                       now,
		UpdatedAt:                       now,
	}

	subscriptionSchema, err := schema.MapAgentSubscriptionToSchema(subscription)
	if err != nil {
		return nil, fmt.Errorf("failed to map subscription: %w", err)
	}

	if err := s.subscriptionRepo.Create(ctx, subscriptionSchema); err != nil {
		return nil, fmt.Errorf("failed to create subscription: %w", err)
	}

	// Create initial usage tracking
	if _, err := s.usageService.GetOrCreateCurrentUsage(ctx, subscription.ID, subscription.UserID); err != nil {
		s.log.Warn("failed to create initial usage tracking", "error", err)
	}

	s.log.Info("created subscription",
		"subscription_id", subscription.ID,
		"user_id", input.UserID,
		"plan_type", input.PlanType,
		"status", status,
		"payment_id", paymentID,
	)

	return &CreateSubscriptionResult{
		Subscription: subscription,
		PaymentURL:   paymentURL,
		PaymentID:    paymentID,
	}, nil
}

// GetSubscription retrieves a subscription by ID
func (s *SubscriptionServiceImpl) GetSubscription(ctx context.Context, subscriptionID uuid.UUID) (*domain.AgentSubscription, error) {
	subscriptionSchema, err := s.subscriptionRepo.GetByID(ctx, subscriptionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}

	if subscriptionSchema == nil {
		return nil, domain.ErrSubscriptionNotFound
	}

	subscription, err := schema.MapAgentSubscriptionFromSchema(subscriptionSchema)
	if err != nil {
		return nil, fmt.Errorf("failed to map subscription: %w", err)
	}

	return subscription, nil
}

// GetUserSubscription retrieves the active subscription for a user
func (s *SubscriptionServiceImpl) GetUserSubscription(ctx context.Context, userID uuid.UUID) (*domain.AgentSubscription, error) {
	subscriptionSchema, err := s.subscriptionRepo.GetActiveByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user subscription: %w", err)
	}

	if subscriptionSchema == nil {
		return nil, nil // No active subscription (free tier)
	}

	subscription, err := schema.MapAgentSubscriptionFromSchema(subscriptionSchema)
	if err != nil {
		return nil, fmt.Errorf("failed to map subscription: %w", err)
	}

	return subscription, nil
}

// CanAddListing checks if user can add another listing
func (s *SubscriptionServiceImpl) CanAddListing(ctx context.Context, userID uuid.UUID) (bool, error) {
	subscription, err := s.GetUserSubscription(ctx, userID)
	if err != nil {
		return false, err
	}

	// Free tier
	if subscription == nil {
		// TODO: Need to count user's listings - this requires property service integration
		// For now, return true and let property service handle the check
		return true, nil
	}

	// Check subscription limits
	if subscription.HasUnlimitedListings() {
		return true, nil
	}

	// TODO: Need actual listing count from property service
	// For now, return based on limit only
	return subscription.MaxListings > 0, nil
}

// CanAddPhotos checks if user can add photos to a listing
func (s *SubscriptionServiceImpl) CanAddPhotos(ctx context.Context, userID, listingID uuid.UUID, photoCount int) (bool, error) {
	subscription, err := s.GetUserSubscription(ctx, userID)
	if err != nil {
		return false, err
	}

	// Free tier
	if subscription == nil {
		freeTierLimit := getFreeTierLimit(s.config, "max_photos_per_listing")
		return photoCount <= freeTierLimit, nil
	}

	// Check subscription limits
	return photoCount <= subscription.MaxPhotosPerListing, nil
}

// CanUseFeature checks if user has access to a feature
func (s *SubscriptionServiceImpl) CanUseFeature(ctx context.Context, userID uuid.UUID, feature string) (bool, error) {
	subscription, err := s.GetUserSubscription(ctx, userID)
	if err != nil {
		return false, err
	}

	// Free tier
	if subscription == nil {
		return getFreeTierFeature(s.config, feature), nil
	}

	// Check subscription features
	return subscription.GetFeature(feature), nil
}

// CanCreateOpenHouse checks if user can create an open house
func (s *SubscriptionServiceImpl) CanCreateOpenHouse(ctx context.Context, userID uuid.UUID) (bool, int, error) {
	subscription, err := s.GetUserSubscription(ctx, userID)
	if err != nil {
		return false, 0, err
	}

	// Free tier - no open houses
	if subscription == nil {
		return false, 0, nil
	}

	// Check feature access
	if !subscription.GetFeature("open_house_events_enabled") {
		return false, 0, nil
	}

	// Get usage tracking
	usage, err := s.usageService.GetOrCreateCurrentUsage(ctx, subscription.ID, userID)
	if err != nil {
		return false, 0, fmt.Errorf("failed to get usage: %w", err)
	}

	// Check quota
	limit := subscription.IncludedOpenHousesPerMonth
	if limit == -1 {
		return true, -1, nil // Unlimited
	}

	canCreate := usage.CanCreateOpenHouse(limit)
	remaining := usage.GetRemainingOpenHouses(limit)

	return canCreate, remaining, nil
}

// CanCreatePrivateShowing checks if user can create a private showing
func (s *SubscriptionServiceImpl) CanCreatePrivateShowing(ctx context.Context, userID uuid.UUID) (bool, int, error) {
	subscription, err := s.GetUserSubscription(ctx, userID)
	if err != nil {
		return false, 0, err
	}

	// Free tier - no private showings
	if subscription == nil {
		return false, 0, nil
	}

	// Check feature access
	if !subscription.GetFeature("private_showings_enabled") {
		return false, 0, nil
	}

	// Get usage tracking
	usage, err := s.usageService.GetOrCreateCurrentUsage(ctx, subscription.ID, userID)
	if err != nil {
		return false, 0, fmt.Errorf("failed to get usage: %w", err)
	}

	// Check quota
	limit := subscription.IncludedPrivateShowingsPerMonth
	if limit == -1 {
		return true, -1, nil // Unlimited
	}

	canCreate := usage.CanCreatePrivateShowing(limit)
	remaining := usage.GetRemainingPrivateShowings(limit)

	return canCreate, remaining, nil
}

// UseOpenHouse marks an open house slot as used
func (s *SubscriptionServiceImpl) UseOpenHouse(ctx context.Context, userID uuid.UUID) error {
	subscription, err := s.GetUserSubscription(ctx, userID)
	if err != nil {
		return err
	}

	if subscription == nil {
		return domain.ErrFeatureNotAvailable
	}

	return s.usageService.IncrementUsage(ctx, subscription.ID, domain.UsageTypeOpenHouse)
}

// UsePrivateShowing marks a private showing slot as used
func (s *SubscriptionServiceImpl) UsePrivateShowing(ctx context.Context, userID uuid.UUID) error {
	subscription, err := s.GetUserSubscription(ctx, userID)
	if err != nil {
		return err
	}

	if subscription == nil {
		return domain.ErrFeatureNotAvailable
	}

	return s.usageService.IncrementUsage(ctx, subscription.ID, domain.UsageTypePrivateShowing)
}

// Stub implementations for other methods (to be fully implemented in next iteration)

func (s *SubscriptionServiceImpl) UpgradeSubscription(ctx context.Context, subscriptionID uuid.UUID, newPlan domain.PlanType) error {
	// Get the subscription
	subscription, err := s.GetSubscription(ctx, subscriptionID)
	if err != nil {
		return fmt.Errorf("failed to get subscription: %w", err)
	}

	// Validate upgrade is possible
	if !subscription.CanUpgrade() {
		return domain.ErrCannotUpgrade
	}

	// Ensure new plan is actually an upgrade
	if newPlan <= subscription.PlanType {
		return fmt.Errorf("new plan must be higher tier than current plan")
	}

	// Schedule the plan change for next billing cycle
	if err := subscription.SchedulePlanChange(newPlan); err != nil {
		return fmt.Errorf("failed to schedule plan change: %w", err)
	}

	// Save updated subscription
	subscriptionSchema, err := schema.MapAgentSubscriptionToSchema(subscription)
	if err != nil {
		return fmt.Errorf("failed to map subscription: %w", err)
	}

	if err := s.subscriptionRepo.Update(ctx, subscriptionSchema); err != nil {
		return fmt.Errorf("failed to update subscription: %w", err)
	}

	s.log.Info("scheduled subscription upgrade",
		"subscription_id", subscriptionID,
		"current_plan", subscription.PlanType,
		"pending_plan", newPlan,
		"next_billing_date", subscription.NextBillingDate,
	)

	return nil
}

func (s *SubscriptionServiceImpl) DowngradeSubscription(ctx context.Context, subscriptionID uuid.UUID, newPlan domain.PlanType) error {
	// Get the subscription
	subscription, err := s.GetSubscription(ctx, subscriptionID)
	if err != nil {
		return fmt.Errorf("failed to get subscription: %w", err)
	}

	// Validate downgrade is possible
	if !subscription.CanDowngrade() {
		return domain.ErrCannotDowngrade
	}

	// Ensure new plan is actually a downgrade
	if newPlan >= subscription.PlanType {
		return fmt.Errorf("new plan must be lower tier than current plan")
	}

	// Schedule the plan change for next billing cycle
	if err := subscription.SchedulePlanChange(newPlan); err != nil {
		return fmt.Errorf("failed to schedule plan change: %w", err)
	}

	// Save updated subscription
	subscriptionSchema, err := schema.MapAgentSubscriptionToSchema(subscription)
	if err != nil {
		return fmt.Errorf("failed to map subscription: %w", err)
	}

	if err := s.subscriptionRepo.Update(ctx, subscriptionSchema); err != nil {
		return fmt.Errorf("failed to update subscription: %w", err)
	}

	s.log.Info("scheduled subscription downgrade",
		"subscription_id", subscriptionID,
		"current_plan", subscription.PlanType,
		"pending_plan", newPlan,
		"next_billing_date", subscription.NextBillingDate,
	)

	return nil
}

func (s *SubscriptionServiceImpl) CancelSubscription(ctx context.Context, subscriptionID uuid.UUID) error {
	subscription, err := s.GetSubscription(ctx, subscriptionID)
	if err != nil {
		return err
	}

	if err := subscription.Cancel(); err != nil {
		return err
	}

	subscriptionSchema, err := schema.MapAgentSubscriptionToSchema(subscription)
	if err != nil {
		return err
	}

	return s.subscriptionRepo.Update(ctx, subscriptionSchema)
}

func (s *SubscriptionServiceImpl) RenewSubscription(ctx context.Context, subscriptionID, paymentID uuid.UUID) error {
	// Get the subscription
	subscription, err := s.GetSubscription(ctx, subscriptionID)
	if err != nil {
		return err
	}

	// Update next billing date
	nextBillingDate := calculateNextBillingDate(time.Now(), subscription.BillingCycle)
	subscription.NextBillingDate = &nextBillingDate

	// Activate subscription if it was pending or in trial
	if subscription.Status == domain.SubscriptionStatusPending || subscription.Status == domain.SubscriptionStatusTrial {
		subscription.Status = domain.SubscriptionStatusActive
	}

	// Save updated subscription
	subscriptionSchema, err := schema.MapAgentSubscriptionToSchema(subscription)
	if err != nil {
		return fmt.Errorf("failed to map subscription: %w", err)
	}

	if err := s.subscriptionRepo.Update(ctx, subscriptionSchema); err != nil {
		return fmt.Errorf("failed to update subscription: %w", err)
	}

	// Reset usage for new billing period
	if err := s.usageService.ResetUsage(ctx, subscription.ID, subscription.UserID); err != nil {
		s.log.Warn("failed to reset usage after renewal", "subscription_id", subscription.ID, "error", err)
		// Don't fail renewal if usage reset fails
	}

	s.log.Info("subscription renewed",
		"subscription_id", subscriptionID,
		"payment_id", paymentID,
		"next_billing_date", nextBillingDate,
	)

	return nil
}

func (s *SubscriptionServiceImpl) ProcessBilling(ctx context.Context) error {
	// Get subscriptions due for billing
	subscriptions, err := s.subscriptionRepo.ListDueForBilling(ctx)
	if err != nil {
		return fmt.Errorf("failed to list subscriptions due for billing: %w", err)
	}

	if len(subscriptions) == 0 {
		s.log.Info("no subscriptions due for billing")
		return nil
	}

	s.log.Info("processing subscription billing", "count", len(subscriptions))

	successCount := 0
	failureCount := 0

	for _, subSchema := range subscriptions {
		subscription, err := schema.MapAgentSubscriptionFromSchema(subSchema)
		if err != nil {
			s.log.Warn("failed to map subscription", "subscription_id", subSchema.ID, "error", err)
			failureCount++
			continue
		}

		// Get user's default payment method
		paymentMethod, err := s.paymentService.GetDefaultPaymentMethod(ctx, subscription.UserID)
		if err != nil {
			s.log.Warn("failed to get payment method for subscription billing",
				"subscription_id", subscription.ID,
				"user_id", subscription.UserID,
				"error", err,
			)
			failureCount++
			continue
		}

		if paymentMethod == nil {
			s.log.Warn("no default payment method found for subscription billing",
				"subscription_id", subscription.ID,
				"user_id", subscription.UserID,
			)
			failureCount++
			continue
		}

		// Determine amount to charge (check for pending plan change)
		amountToCharge := subscription.Amount
		currencyToCharge := subscription.Currency
		planForDescription := subscription.PlanType
		planForMetadata := subscription.PlanType.String()

		if subscription.HasPendingPlanChange() {
			// Load new plan config to get the new amount
			planConfig, err := loadPlanConfig(s.config, *subscription.PendingPlanType, subscription.BillingCycle)
			if err != nil {
				s.log.Warn("failed to load new plan config, charging old amount",
					"subscription_id", subscription.ID,
					"error", err,
				)
			} else {
				amountToCharge = planConfig.Amount
				currencyToCharge = planConfig.Currency
				planForDescription = *subscription.PendingPlanType
				planForMetadata = subscription.PendingPlanType.String()
				s.log.Info("billing with new plan amount due to pending change",
					"subscription_id", subscription.ID,
					"new_plan", *subscription.PendingPlanType,
					"new_amount", amountToCharge,
				)
			}
		}

		if amountToCharge <= 0 {
			s.log.Info("skipping billing for zero-amount subscription",
				"subscription_id", subscription.ID,
				"plan_type", planForDescription,
			)
			continue
		}

		// Create payment for subscription renewal
		paymentInput := paymentDomain.CreatePaymentInput{
			Amount:          amountToCharge,
			Currency:        payment.Currency(currencyToCharge),
			Market:          paymentDomain.MarketNigeria,
			PayerID:         subscription.UserID,
			PayerEmail:      subscription.UserEmail,
			PayerName:       subscription.UserName,
			ResourceType:    paymentDomain.ResourceTypeSubscription,
			ResourceID:      &subscription.ID,
			PaymentMethodID: &paymentMethod.ID,
			Description:     fmt.Sprintf("Subscription renewal - %s plan", planForDescription),
			Metadata: map[string]string{
				"subscription_id": subscription.ID.String(),
				"plan_type":       planForMetadata,
				"billing_cycle":   subscription.BillingCycle.String(),
			},
		}

		// Create payment (will charge the saved payment method)
		pmt, err := s.paymentService.CreatePayment(ctx, paymentInput)
		if err != nil {
			s.log.Warn("failed to create payment for subscription billing",
				"subscription_id", subscription.ID,
				"user_id", subscription.UserID,
				"error", err,
			)
			failureCount++
			continue
		}

		// Check if payment was successful
		if pmt.Status == paymentDomain.PaymentStatusSucceeded {
			// Apply pending plan change if exists
			if subscription.HasPendingPlanChange() {
				s.log.Info("applying pending plan change during billing",
					"subscription_id", subscription.ID,
					"current_plan", subscription.PlanType,
					"new_plan", *subscription.PendingPlanType,
				)

				// Load new plan configuration
				planConfig, err := loadPlanConfig(s.config, *subscription.PendingPlanType, subscription.BillingCycle)
				if err != nil {
					s.log.Warn("failed to load new plan config, skipping plan change",
						"subscription_id", subscription.ID,
						"error", err,
					)
				} else {
					// Apply the pending plan change
					planLimits := domain.PlanLimits{
						MaxListings:                     planConfig.MaxListings,
						MaxPhotosPerListing:             planConfig.MaxPhotosPerListing,
						MaxVirtualTours:                 planConfig.MaxVirtualTours,
						IncludedFeaturedPerMonth:        planConfig.IncludedFeaturedPerMonth,
						IncludedPremiumPerMonth:         planConfig.IncludedPremiumPerMonth,
						IncludedOpenHousesPerMonth:      planConfig.IncludedOpenHousesPerMonth,
						IncludedPrivateShowingsPerMonth: planConfig.IncludedPrivateShowingsPerMonth,
						Features:                        planConfig.Features,
					}

					if err := subscription.ApplyPendingPlanChange(planLimits, planConfig.Amount, planConfig.Currency); err != nil {
						s.log.Warn("failed to apply pending plan change",
							"subscription_id", subscription.ID,
							"error", err,
						)
					} else {
						s.log.Info("successfully applied plan change",
							"subscription_id", subscription.ID,
							"new_plan", subscription.PlanType,
							"new_amount", subscription.Amount,
						)
					}
				}
			}

			// Update next billing date
			nextBillingDate := calculateNextBillingDate(time.Now(), subscription.BillingCycle)
			subscription.NextBillingDate = &nextBillingDate

			// Save updated subscription
			updatedSchema, err := schema.MapAgentSubscriptionToSchema(subscription)
			if err != nil {
				s.log.Warn("failed to map subscription after billing", "subscription_id", subscription.ID, "error", err)
				failureCount++
				continue
			}

			if err := s.subscriptionRepo.Update(ctx, updatedSchema); err != nil {
				s.log.Warn("failed to update subscription after billing", "subscription_id", subscription.ID, "error", err)
				failureCount++
				continue
			}

			// Reset usage for new billing period
			if err := s.usageService.ResetUsage(ctx, subscription.ID, subscription.UserID); err != nil {
				s.log.Warn("failed to reset usage after billing", "subscription_id", subscription.ID, "error", err)
				// Don't count this as a failure since payment succeeded
			}

			s.log.Info("successfully processed subscription billing",
				"subscription_id", subscription.ID,
				"payment_id", pmt.ID,
				"amount", subscription.Amount,
			)
			successCount++
		} else {
			s.log.Warn("subscription billing payment not successful",
				"subscription_id", subscription.ID,
				"payment_id", pmt.ID,
				"payment_status", pmt.Status,
			)
			failureCount++
		}
	}

	s.log.Info("subscription billing processing complete",
		"total", len(subscriptions),
		"success", successCount,
		"failed", failureCount,
	)

	return nil
}

func (s *SubscriptionServiceImpl) CanUseIncludedPromotion(ctx context.Context, userID uuid.UUID, promoType domain.PromotionType) (bool, error) {
	// TODO: Implement included promotion quota check
	return false, fmt.Errorf("not yet implemented")
}

func (s *SubscriptionServiceImpl) UseIncludedPromotion(ctx context.Context, userID uuid.UUID, promoType domain.PromotionType) error {
	// TODO: Implement usage tracking for included promotions
	return fmt.Errorf("not yet implemented")
}

func (s *SubscriptionServiceImpl) GetFeatureLimit(ctx context.Context, userID uuid.UUID, feature string) (int, error) {
	// TODO: Implement feature limit lookup
	return 0, fmt.Errorf("not yet implemented")
}
