package hooks

import (
	"context"

	discoveryservice "hauslet/internal/modules/discovery/service"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AnalyticsDiscoveryAdapter queries cross-module analytics tables directly.
// This adapter uses *gorm.DB because the queries span module boundaries
// (interaction_aggregates from interactions module, listing_stats from review module).
type AnalyticsDiscoveryAdapter struct {
	db *gorm.DB
}

// NewAnalyticsDiscoveryAdapter creates a new analytics discovery adapter.
func NewAnalyticsDiscoveryAdapter(db *gorm.DB) discoveryservice.AnalyticsDiscoveryHooks {
	return &AnalyticsDiscoveryAdapter{db: db}
}

// GetTrendingListingIDs returns listing IDs ordered by engagement score
// from the interaction_aggregates table (weekly period).
func (a *AnalyticsDiscoveryAdapter) GetTrendingListingIDs(ctx context.Context, limit int) ([]uuid.UUID, error) {
	var ids []uuid.UUID

	err := a.db.WithContext(ctx).
		Table("interaction_aggregates").
		Select("entity_id").
		Where("entity_type = ?", "listing").
		Where("period_type = ?", "week").
		Where("views_total > ?", 0).
		Order("engagement_score DESC, views_total DESC").
		Limit(limit).
		Pluck("entity_id", &ids).Error

	if err != nil {
		return nil, err
	}

	return ids, nil
}

// GetTopRatedListingIDs returns listing IDs with the highest average ratings,
// requiring at least 3 reviews for statistical significance.
func (a *AnalyticsDiscoveryAdapter) GetTopRatedListingIDs(ctx context.Context, limit int) ([]uuid.UUID, error) {
	const minReviews = 3

	var ids []uuid.UUID

	err := a.db.WithContext(ctx).
		Table("listing_stats").
		Select("listing_id").
		Where("review_count >= ?", minReviews).
		Where("average_rating >= ?", 4.0).
		Order("average_rating DESC, review_count DESC").
		Limit(limit).
		Pluck("listing_id", &ids).Error

	if err != nil {
		return nil, err
	}

	return ids, nil
}
