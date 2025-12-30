package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"hauslet/internal/modules/review/repository/schema"
)

type StatsRepositoryImpl struct {
	db *gorm.DB
}

func NewStatsRepository(db *gorm.DB) StatsRepository {
	return &StatsRepositoryImpl{db: db}
}

// ----------------------------------------------------------------
// READ OPERATIONS (Fast, Cached Access)
// ----------------------------------------------------------------

func (r *StatsRepositoryImpl) GetListingStats(ctx context.Context, listingID uuid.UUID) (*schema.ListingStats, error) {
	var stats schema.ListingStats

	// We use First() because ListingStats has a primary key of ListingID
	err := r.db.WithContext(ctx).First(&stats, "listing_id = ?", listingID).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		// Crucial UX Pattern:
		// If stats haven't been calculated yet (new listing), return a valid "Zero" object
		// instead of an error. This prevents the UI from crashing or showing 500s.
		return &schema.ListingStats{
			ListingID:          listingID,
			AverageRating:      0,
			ReviewCount:        0,
			SubRatingAverages:  []byte("{}"),
			RatingDistribution: []byte("{}"),
		}, nil
	}

	return &stats, err
}

func (r *StatsRepositoryImpl) GetHostStats(ctx context.Context, hostID uuid.UUID) (*schema.HostStats, error) {
	var stats schema.HostStats
	err := r.db.WithContext(ctx).First(&stats, "host_id = ?", hostID).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &schema.HostStats{
			HostID:              hostID,
			GlobalAverageRating: 0,
			TotalReviewCount:    0,
		}, nil
	}
	return &stats, err
}

// ----------------------------------------------------------------
// WRITE OPERATIONS (Complex Aggregation)
// ----------------------------------------------------------------

// RecalculateStats is the heavy lifter. It aggregates all published reviews
// for a listing and updates the cache table in a single atomic SQL operation.
func (r *StatsRepositoryImpl) RecalculateStats(ctx context.Context, listingID uuid.UUID) error {
	// SQL Breakdown:
	// 1. CTE 'relevant_reviews': Filters down to just the reviews we care about.
	// 2. CTE 'sub_ratings_agg': Expands the JSONB sub-ratings (cleanliness, etc.), averages them by key, and rebuilds a JSON object.
	// 3. CTE 'histogram_agg': Counts how many 5-star, 4-star, etc. reviews exist.
	// 4. INSERT ... ON CONFLICT: Upserts the final result into the stats table.

	query := `
		WITH relevant_reviews AS (
			SELECT rating, sub_ratings
			FROM reviews
			WHERE listing_id = ? 
			  AND status = 'published' 
			  AND deleted_at IS NULL
		),
		sub_ratings_agg AS (
			SELECT 
				jsonb_object_agg(key, ROUND(avg_val, 2)) as data
			FROM (
				SELECT key, AVG(value::numeric) as avg_val
				FROM relevant_reviews, jsonb_each_text(sub_ratings)
				GROUP BY key
			) expanded
		),
		histogram_agg AS (
			SELECT json_build_object(
				'5', COUNT(*) FILTER (WHERE rating = 5),
				'4', COUNT(*) FILTER (WHERE rating = 4),
				'3', COUNT(*) FILTER (WHERE rating = 3),
				'2', COUNT(*) FILTER (WHERE rating = 2),
				'1', COUNT(*) FILTER (WHERE rating = 1)
			) as dist,
			COUNT(*) as total_count,
			COALESCE(AVG(rating), 0) as avg_rating
			FROM relevant_reviews
		)
		INSERT INTO listing_stats (
			listing_id, 
			review_count, 
			average_rating, 
			sub_rating_averages, 
			rating_distribution, 
			last_updated_at
		)
		SELECT 
			?, 
			(SELECT total_count FROM histogram_agg),
			(SELECT avg_rating FROM histogram_agg),
			COALESCE((SELECT data FROM sub_ratings_agg), '{}'::jsonb),
			(SELECT dist FROM histogram_agg),
			NOW()
		ON CONFLICT (listing_id) 
		DO UPDATE SET 
			review_count = EXCLUDED.review_count,
			average_rating = EXCLUDED.average_rating,
			sub_rating_averages = EXCLUDED.sub_rating_averages,
			rating_distribution = EXCLUDED.rating_distribution,
			last_updated_at = EXCLUDED.last_updated_at;
	`

	return r.db.WithContext(ctx).Exec(query, listingID, listingID).Error
}

// RecalculateHostStats aggregates stats for a user (Host).
// NOTE: This implementation aggregates reviews where TargetType = 'host'.
// If your business logic defines "Host Stats" as "Aggregate of all their listings",
// you would need to JOIN the listings table here.
func (r *StatsRepositoryImpl) RecalculateHostStats(ctx context.Context, hostID uuid.UUID) error {
	query := `
		INSERT INTO host_stats (
			host_id, 
			global_average_rating, 
			total_review_count, 
			review_count_last_365_days,
			last_updated_at
		)
		SELECT 
			?, 
			COALESCE(AVG(rating), 0), 
			COUNT(id),
			COUNT(id) FILTER (WHERE created_at > NOW() - INTERVAL '1 year'),
			NOW()
		FROM reviews 
		WHERE target_id = ? 
		  AND target_type = 'host'
		  AND status = 'published' 
		  AND deleted_at IS NULL
		ON CONFLICT (host_id) 
		DO UPDATE SET 
			global_average_rating = EXCLUDED.global_average_rating,
			total_review_count = EXCLUDED.total_review_count,
			review_count_last_365_days = EXCLUDED.review_count_last_365_days,
			last_updated_at = EXCLUDED.last_updated_at;
	`

	return r.db.WithContext(ctx).Exec(query, hostID, hostID).Error
}
