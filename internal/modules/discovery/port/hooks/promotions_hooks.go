package hooks

import (
	"context"
	"fmt"

	discoveryservice "hauslet/internal/modules/discovery/service"
	promotiondomain "hauslet/internal/modules/promotions/domain"
	promotionservice "hauslet/internal/modules/promotions/service"

	"github.com/google/uuid"
)

// PromotionDiscoveryAdapter wraps PromotionService to implement PromotionDiscoveryHooks
type PromotionDiscoveryAdapter struct {
	promotionSvc promotionservice.PromotionService
}

// NewPromotionDiscoveryAdapter creates a new PromotionDiscoveryAdapter
func NewPromotionDiscoveryAdapter(promotionSvc promotionservice.PromotionService) discoveryservice.PromotionDiscoveryHooks {
	return &PromotionDiscoveryAdapter{
		promotionSvc: promotionSvc,
	}
}

// GetActivePromotionForListing gets the active promotion for a single listing
func (a *PromotionDiscoveryAdapter) GetActivePromotionForListing(ctx context.Context, listingID uuid.UUID) (*discoveryservice.PromotionInfo, error) {
	promo, err := a.promotionSvc.GetActivePromotion(ctx, listingID)
	if err != nil {
		return nil, err
	}
	if promo == nil {
		return nil, nil
	}

	return &discoveryservice.PromotionInfo{
		PromotionID:     promo.ID,
		PromotionType:   string(promo.Type),
		BoostMultiplier: promo.BoostMultiplier,
		ExpiresAt:       *promo.ExpiresAt,
	}, nil
}

// GetActivePromotionForListings batch fetches active promotions for multiple listings
func (a *PromotionDiscoveryAdapter) GetActivePromotionForListings(ctx context.Context, listingIDs []uuid.UUID) (map[uuid.UUID]*discoveryservice.PromotionInfo, error) {
	if len(listingIDs) == 0 {
		return make(map[uuid.UUID]*discoveryservice.PromotionInfo), nil
	}

	// Use batch method to avoid N+1 queries
	promos, err := a.promotionSvc.GetActivePromotions(ctx, listingIDs)
	if err != nil {
		return nil, err
	}

	result := make(map[uuid.UUID]*discoveryservice.PromotionInfo, len(promos))
	for listingID, promo := range promos {
		if promo == nil || promo.ExpiresAt == nil {
			continue
		}
		result[listingID] = &discoveryservice.PromotionInfo{
			PromotionID:     promo.ID,
			PromotionType:   string(promo.Type),
			BoostMultiplier: promo.BoostMultiplier,
			ExpiresAt:       *promo.ExpiresAt,
		}
	}

	return result, nil
}

// GetFeaturedListings gets listing IDs for currently featured promotions
func (a *PromotionDiscoveryAdapter) GetFeaturedListings(ctx context.Context, limit int) ([]uuid.UUID, error) {
	promos, err := a.promotionSvc.GetFeaturedListings(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get featured listings: %w", err)
	}

	listingIDs := make([]uuid.UUID, 0, len(promos))
	for _, promo := range promos {
		// Only include if promotion is still active
		if promo.Status == promotiondomain.PromotionStatusActive {
			listingIDs = append(listingIDs, promo.ListingID)
		}
	}

	return listingIDs, nil
}

// GetPremiumListings gets listing IDs for currently premium promotions
func (a *PromotionDiscoveryAdapter) GetPremiumListings(ctx context.Context, limit int) ([]uuid.UUID, error) {
	promos, err := a.promotionSvc.GetPremiumListings(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get premium listings: %w", err)
	}

	listingIDs := make([]uuid.UUID, 0, len(promos))
	for _, promo := range promos {
		// Only include if promotion is still active
		if promo.Status == promotiondomain.PromotionStatusActive {
			listingIDs = append(listingIDs, promo.ListingID)
		}
	}

	return listingIDs, nil
}
