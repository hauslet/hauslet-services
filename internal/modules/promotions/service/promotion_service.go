package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"hauslet/config"
	paymentDomain "hauslet/internal/modules/payments/domain"
	paymentService "hauslet/internal/modules/payments/service"
	"hauslet/internal/modules/promotions/domain"
	"hauslet/internal/modules/promotions/repository"
	"hauslet/internal/modules/promotions/repository/schema"
	"hauslet/internal/platform/payment"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PromotionServiceImpl implements PromotionService
type PromotionServiceImpl struct {
	promoRepo        repository.ListingPromotionRepository
	subscriptionRepo repository.AgentSubscriptionRepository
	usageService     UsageService
	paymentService   paymentService.PaymentService
	config           *config.PromotionYAMLConfig
	db               *gorm.DB
	log              *slog.Logger
}

// NewPromotionService creates a new promotion service
func NewPromotionService(
	promoRepo repository.ListingPromotionRepository,
	subscriptionRepo repository.AgentSubscriptionRepository,
	usageService UsageService,
	paymentService paymentService.PaymentService,
	config *config.PromotionYAMLConfig,
	db *gorm.DB,
	log *slog.Logger,
) PromotionService {
	return &PromotionServiceImpl{
		promoRepo:        promoRepo,
		subscriptionRepo: subscriptionRepo,
		usageService:     usageService,
		paymentService:   paymentService,
		config:           config,
		db:               db,
		log:              log,
	}
}

// CreatePromotion creates a new promotion and initiates payment
func (s *PromotionServiceImpl) CreatePromotion(ctx context.Context, input CreatePromotionInput) (*CreatePromotionResult, error) {
	// Validate promotion type
	if !input.Type.IsValid() {
		return nil, domain.ErrInvalidPromotionType
	}

	// Validate duration
	if err := validatePromotionDuration(s.config, input.Type, input.Duration); err != nil {
		return nil, err
	}

	// Check for existing active promotion
	existingPromo, err := s.promoRepo.GetActiveByListing(ctx, input.ListingID)
	if err != nil {
		return nil, fmt.Errorf("failed to check for existing promotion: %w", err)
	}
	if existingPromo != nil {
		return nil, domain.ErrDuplicateActivePromotion
	}

	// Calculate price
	amount, err := calculatePromotionPrice(s.config, input.Type, input.Duration)
	if err != nil {
		return nil, err
	}

	// Calculate boost multiplier
	boostMultiplier := calculateBoostMultiplier(s.config, input.Type)

	// Create promotion ID upfront to link with payment
	promotionID := uuid.New()

	// Create payment first
	paymentInput := paymentDomain.CreatePaymentInput{
		Amount:       amount,
		Currency:     payment.Currency(s.config.Currency.Code),
		Market:       paymentDomain.MarketNigeria,
		PayerID:      input.OwnerID,
		PayerEmail:   input.OwnerEmail,
		PayerName:    input.OwnerName,
		ResourceType: paymentDomain.ResourceTypePromotion,
		ResourceID:   &promotionID,
		Description:  fmt.Sprintf("%s promotion for %d days", input.Type, input.Duration),
		Metadata: map[string]string{
			"promotion_id": promotionID.String(),
			"listing_id":   input.ListingID.String(),
			"type":         input.Type.String(),
			"duration":     fmt.Sprintf("%d", input.Duration),
		},
	}

	pmt, err := s.paymentService.CreatePayment(ctx, paymentInput)
	if err != nil {
		return nil, fmt.Errorf("failed to create payment: %w", err)
	}

	// Create promotion with payment ID
	now := time.Now()
	promotion := &domain.ListingPromotion{
		ID:              promotionID,
		ListingID:       input.ListingID,
		OwnerID:         input.OwnerID,
		Type:            input.Type,
		Status:          domain.PromotionStatusPending,
		Amount:          amount,
		Currency:        s.config.Currency.Code,
		Duration:        input.Duration,
		PaymentID:       &pmt.ID,
		IsIncluded:      false,
		BoostMultiplier: boostMultiplier,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	promotionSchema, err := schema.MapListingPromotionToSchema(promotion)
	if err != nil {
		return nil, fmt.Errorf("failed to map promotion: %w", err)
	}

	if err := s.promoRepo.Create(ctx, promotionSchema); err != nil {
		return nil, fmt.Errorf("failed to create promotion: %w", err)
	}

	s.log.Info("created promotion with payment",
		"promotion_id", promotion.ID,
		"payment_id", pmt.ID,
		"listing_id", input.ListingID,
		"type", input.Type,
		"duration", input.Duration,
		"amount", amount,
	)

	paymentURL := ""
	if pmt.RedirectURL != nil {
		paymentURL = *pmt.RedirectURL
	}

	return &CreatePromotionResult{
		Promotion:  promotion,
		PaymentURL: paymentURL,
		PaymentID:  pmt.ID,
	}, nil
}

// CreateIncludedPromotion creates a promotion from subscription quota (no payment)
func (s *PromotionServiceImpl) CreateIncludedPromotion(ctx context.Context, input CreateIncludedPromotionInput) (*domain.ListingPromotion, error) {
	// Validate promotion type
	if !input.Type.IsValid() {
		return nil, domain.ErrInvalidPromotionType
	}

	// Validate duration
	if err := validatePromotionDuration(s.config, input.Type, input.Duration); err != nil {
		return nil, err
	}

	// Check for existing active promotion
	existingPromo, err := s.promoRepo.GetActiveByListing(ctx, input.ListingID)
	if err != nil {
		return nil, fmt.Errorf("failed to check for existing promotion: %w", err)
	}
	if existingPromo != nil {
		return nil, domain.ErrDuplicateActivePromotion
	}

	// Calculate price (for reference, even though it's included)
	amount, err := calculatePromotionPrice(s.config, input.Type, input.Duration)
	if err != nil {
		return nil, err
	}

	// Calculate boost multiplier
	boostMultiplier := calculateBoostMultiplier(s.config, input.Type)

	now := time.Now()
	promotion := &domain.ListingPromotion{
		ID:              uuid.New(),
		ListingID:       input.ListingID,
		OwnerID:         input.OwnerID,
		Type:            input.Type,
		Status:          domain.PromotionStatusPending,
		Amount:          amount,
		Currency:        s.config.Currency.Code,
		Duration:        input.Duration,
		SubscriptionID:  &input.SubscriptionID,
		IsIncluded:      true,
		BoostMultiplier: boostMultiplier,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	promotionSchema, err := schema.MapListingPromotionToSchema(promotion)
	if err != nil {
		return nil, fmt.Errorf("failed to map promotion: %w", err)
	}

	if err := s.promoRepo.Create(ctx, promotionSchema); err != nil {
		return nil, fmt.Errorf("failed to create included promotion: %w", err)
	}

	s.log.Info("created included promotion",
		"promotion_id", promotion.ID,
		"listing_id", input.ListingID,
		"type", input.Type,
		"subscription_id", input.SubscriptionID,
	)

	return promotion, nil
}

// StartPromotion activates a promotion (called after payment confirmation)
func (s *PromotionServiceImpl) StartPromotion(ctx context.Context, promoID uuid.UUID) error {
	promotionSchema, err := s.promoRepo.GetByID(ctx, promoID)
	if err != nil {
		return fmt.Errorf("failed to get promotion: %w", err)
	}

	if promotionSchema == nil {
		return domain.ErrPromotionNotFound
	}

	promotion, err := schema.MapListingPromotionFromSchema(promotionSchema)
	if err != nil {
		return fmt.Errorf("failed to map promotion: %w", err)
	}

	// Start the promotion
	if err := promotion.Start(); err != nil {
		return err
	}

	// Update in database
	updatedSchema, err := schema.MapListingPromotionToSchema(promotion)
	if err != nil {
		return fmt.Errorf("failed to map promotion: %w", err)
	}

	if err := s.promoRepo.Update(ctx, updatedSchema); err != nil {
		return fmt.Errorf("failed to update promotion: %w", err)
	}

	// If it's an included promotion, increment usage
	if promotion.IsIncluded && promotion.SubscriptionID != nil {
		var usageType domain.UsageType
		switch promotion.Type {
		case domain.PromotionTypeFeatured:
			usageType = domain.UsageTypeFeatured
		case domain.PromotionTypePremium:
			usageType = domain.UsageTypePremium
		}

		if err := s.usageService.IncrementUsage(ctx, *promotion.SubscriptionID, usageType); err != nil {
			s.log.Warn("failed to increment usage for included promotion", "error", err)
		}
	}

	s.log.Info("started promotion",
		"promotion_id", promoID,
		"started_at", promotion.StartedAt,
		"expires_at", promotion.ExpiresAt,
	)

	return nil
}

// CancelPromotion cancels a pending or active promotion
func (s *PromotionServiceImpl) CancelPromotion(ctx context.Context, promoID uuid.UUID) error {
	promotionSchema, err := s.promoRepo.GetByID(ctx, promoID)
	if err != nil {
		return fmt.Errorf("failed to get promotion: %w", err)
	}

	if promotionSchema == nil {
		return domain.ErrPromotionNotFound
	}

	promotion, err := schema.MapListingPromotionFromSchema(promotionSchema)
	if err != nil {
		return fmt.Errorf("failed to map promotion: %w", err)
	}

	// Cancel the promotion
	if err := promotion.Cancel(); err != nil {
		return err
	}

	// Update in database
	updatedSchema, err := schema.MapListingPromotionToSchema(promotion)
	if err != nil {
		return fmt.Errorf("failed to map promotion: %w", err)
	}

	if err := s.promoRepo.Update(ctx, updatedSchema); err != nil {
		return fmt.Errorf("failed to update promotion: %w", err)
	}

	s.log.Info("cancelled promotion", "promotion_id", promoID)

	return nil
}

// GetPromotion retrieves a promotion by ID
func (s *PromotionServiceImpl) GetPromotion(ctx context.Context, promoID uuid.UUID) (*domain.ListingPromotion, error) {
	promotionSchema, err := s.promoRepo.GetByID(ctx, promoID)
	if err != nil {
		return nil, fmt.Errorf("failed to get promotion: %w", err)
	}

	if promotionSchema == nil {
		return nil, domain.ErrPromotionNotFound
	}

	promotion, err := schema.MapListingPromotionFromSchema(promotionSchema)
	if err != nil {
		return nil, fmt.Errorf("failed to map promotion: %w", err)
	}

	return promotion, nil
}

// ListUserPromotions lists promotions for a user
func (s *PromotionServiceImpl) ListUserPromotions(ctx context.Context, ownerID uuid.UUID, limit, offset int) ([]*domain.ListingPromotion, error) {
	promotionSchemas, err := s.promoRepo.ListByOwner(ctx, ownerID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list promotions: %w", err)
	}

	promotions, err := schema.MapListingPromotionsFromSchema(promotionSchemas)
	if err != nil {
		return nil, fmt.Errorf("failed to map promotions: %w", err)
	}

	return promotions, nil
}

// GetActivePromotion gets the active promotion for a listing
func (s *PromotionServiceImpl) GetActivePromotion(ctx context.Context, listingID uuid.UUID) (*domain.ListingPromotion, error) {
	promotionSchema, err := s.promoRepo.GetActiveByListing(ctx, listingID)
	if err != nil {
		return nil, fmt.Errorf("failed to get active promotion: %w", err)
	}

	if promotionSchema == nil {
		return nil, nil
	}

	promotion, err := schema.MapListingPromotionFromSchema(promotionSchema)
	if err != nil {
		return nil, fmt.Errorf("failed to map promotion: %w", err)
	}

	return promotion, nil
}

// GetActivePromotions batch fetches active promotions for multiple listings
func (s *PromotionServiceImpl) GetActivePromotions(ctx context.Context, listingIDs []uuid.UUID) (map[uuid.UUID]*domain.ListingPromotion, error) {
	if len(listingIDs) == 0 {
		return make(map[uuid.UUID]*domain.ListingPromotion), nil
	}

	promoSchemas, err := s.promoRepo.GetActiveByListingIDs(ctx, listingIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to batch get active promotions: %w", err)
	}

	result := make(map[uuid.UUID]*domain.ListingPromotion, len(promoSchemas))
	for _, ps := range promoSchemas {
		promo, err := schema.MapListingPromotionFromSchema(ps)
		if err != nil {
			continue
		}
		result[promo.ListingID] = promo
	}

	return result, nil
}

// ExpirePromotions expires all promotions that have passed their expiry date (cron job)
func (s *PromotionServiceImpl) ExpirePromotions(ctx context.Context) error {
	// Get promotions expiring now
	promotionSchemas, err := s.promoRepo.ListExpiring(ctx, 0)
	if err != nil {
		return fmt.Errorf("failed to list expiring promotions: %w", err)
	}

	expiredCount := 0
	for _, promoSchema := range promotionSchemas {
		promotion, err := schema.MapListingPromotionFromSchema(promoSchema)
		if err != nil {
			s.log.Warn("failed to map promotion for expiry", "error", err)
			continue
		}

		if err := promotion.Expire(); err != nil {
			s.log.Warn("failed to expire promotion", "promotion_id", promotion.ID, "error", err)
			continue
		}

		updatedSchema, err := schema.MapListingPromotionToSchema(promotion)
		if err != nil {
			s.log.Warn("failed to map expired promotion", "error", err)
			continue
		}

		if err := s.promoRepo.Update(ctx, updatedSchema); err != nil {
			s.log.Warn("failed to update expired promotion", "promotion_id", promotion.ID, "error", err)
			continue
		}

		expiredCount++
	}

	s.log.Info("expired promotions", "count", expiredCount)

	return nil
}

// GetFeaturedListings gets currently featured listings
func (s *PromotionServiceImpl) GetFeaturedListings(ctx context.Context, limit int) ([]*domain.ListingPromotion, error) {
	promotionSchemas, err := s.promoRepo.ListActivePromotions(ctx, domain.PromotionTypeFeatured.String(), limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list featured promotions: %w", err)
	}

	promotions, err := schema.MapListingPromotionsFromSchema(promotionSchemas)
	if err != nil {
		return nil, fmt.Errorf("failed to map promotions: %w", err)
	}

	return promotions, nil
}

// GetPremiumListings gets currently premium listings
func (s *PromotionServiceImpl) GetPremiumListings(ctx context.Context, limit int) ([]*domain.ListingPromotion, error) {
	promotionSchemas, err := s.promoRepo.ListActivePromotions(ctx, domain.PromotionTypePremium.String(), limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list premium promotions: %w", err)
	}

	promotions, err := schema.MapListingPromotionsFromSchema(promotionSchemas)
	if err != nil {
		return nil, fmt.Errorf("failed to map promotions: %w", err)
	}

	return promotions, nil
}
