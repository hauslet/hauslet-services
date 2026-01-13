package repository

import (
	"context"
	"errors"
	"hauslet/internal/modules/review/repository/schema"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ReviewRepositoryImpl struct {
	db *gorm.DB
}

// NewReviewRepository creates a new instance of the repository.
func NewReviewRepository(db *gorm.DB) ReviewRepository {
	return &ReviewRepositoryImpl{db: db}
}

func (r *ReviewRepositoryImpl) Create(ctx context.Context, review *schema.Review) error {
	return r.db.WithContext(ctx).Create(review).Error
}

func (r *ReviewRepositoryImpl) GetByID(ctx context.Context, id uuid.UUID) (*schema.Review, error) {
	var review schema.Review
	if err := r.db.WithContext(ctx).First(&review, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // Or return a custom ErrNotFound
		}
		return nil, err
	}
	return &review, nil
}

func (r *ReviewRepositoryImpl) Update(ctx context.Context, review *schema.Review) error {
	// Safety: Only allow updating specific fields to prevent users from hacking
	// the 'Status' or 'BookingID' via an update payload.
	return r.db.WithContext(ctx).Model(review).
		Updates(map[string]any{
			"title":       review.Title,
			"body":        review.Body,
			"rating":      review.Rating,
			"sub_ratings": review.SubRatings,
			"updated_at":  time.Now(),
		}).Error
}

func (r *ReviewRepositoryImpl) Delete(ctx context.Context, id uuid.UUID) error {
	// Soft Delete
	return r.db.WithContext(ctx).Delete(&schema.Review{}, "id = ?", id).Error
}

func (r *ReviewRepositoryImpl) List(ctx context.Context, filter ReviewFilter) ([]schema.Review, int64, error) {
	var reviews []schema.Review
	var total int64

	// Start building the query
	query := r.db.WithContext(ctx).Model(&schema.Review{})

	// 1. Target Filtering (Listings, Hosts, etc.)
	if filter.TargetID != uuid.Nil {
		query = query.Where("target_id = ?", filter.TargetID)
	}
	if filter.TargetType != "" {
		query = query.Where("target_type = ?", filter.TargetType)
	}

	// 2. Status Filtering
	if filter.OnlyVisible {
		// Only show published reviews.
		// Note: We intentionally exclude 'standoff', 'hidden', and 'archived'.
		query = query.Where("status = ?", schema.ReviewStatusPublished)
	}

	// 2.a. Optional eager loading
	if filter.PreloadResponse {
		query = query.Preload("Response")
	}

	// 3. Content Filtering
	if filter.Rating != nil {
		query = query.Where("rating = ?", *filter.Rating)
	}
	if filter.Language != nil {
		query = query.Where("language = ?", *filter.Language)
	}

	// 4. Get Total Count (before pagination)
	// Efficient counting for pagination metadata
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 6. Sorting
	switch filter.SortBy {
	case "highest_rated":
		query = query.Order("rating DESC, created_at DESC")
	case "lowest_rated":
		query = query.Order("rating ASC, created_at DESC")
	case "oldest":
		query = query.Order("created_at ASC")
	case "newest":
		fallthrough
	default:
		query = query.Order("created_at DESC")
	}

	// 7. Pagination
	if filter.Limit > 0 {
		query = query.Limit(filter.Limit)
	}
	if filter.Offset > 0 {
		query = query.Offset(filter.Offset)
	}

	if err := query.Find(&reviews).Error; err != nil {
		return nil, 0, err
	}

	return reviews, total, nil
}

func (r *ReviewRepositoryImpl) GetByBookingAndReviewer(ctx context.Context, bookingID, reviewerID uuid.UUID) (*schema.Review, error) {
	var review schema.Review
	err := r.db.WithContext(ctx).
		Where("booking_id = ? AND reviewer_id = ?", bookingID, reviewerID).
		First(&review).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &review, err
}

func (r *ReviewRepositoryImpl) GetCounterpartReview(ctx context.Context, bookingID, currentReviewerID uuid.UUID) (*schema.Review, error) {
	var review schema.Review
	// We want a review for the same booking, but NOT by the current user.
	err := r.db.WithContext(ctx).
		Where("booking_id = ? AND reviewer_id != ?", bookingID, currentReviewerID).
		First(&review).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &review, err
}

// PublishExpiredStandoffs is the Cron Job engine.
// It finds reviews that have been hidden ("standoff") for longer than 14 days and force-publishes them.
func (r *ReviewRepositoryImpl) PublishExpiredStandoffs(ctx context.Context, olderThan time.Time) ([]schema.Review, error) {
	var reviews []schema.Review
	query := r.db.WithContext(ctx).
		Model(&schema.Review{}).
		Where("status = ? AND created_at < ?", schema.ReviewStatusStandoff, olderThan)

	if err := query.Find(&reviews).Error; err != nil {
		return nil, err
	}

	if len(reviews) == 0 {
		return reviews, nil
	}

	now := time.Now()
	updateQuery := r.db.WithContext(ctx).
		Model(&schema.Review{}).
		Where("status = ? AND created_at < ?", schema.ReviewStatusStandoff, olderThan)

	if err := updateQuery.Updates(map[string]any{
		"status":       schema.ReviewStatusPublished,
		"published_at": now,
	}).Error; err != nil {
		return nil, err
	}

	for i := range reviews {
		reviews[i].Status = schema.ReviewStatusPublished
		reviews[i].PublishedAt = &now
		reviews[i].UpdatedAt = now
	}

	return reviews, nil
}

func (r *ReviewRepositoryImpl) MarkAsHidden(ctx context.Context, reviewID uuid.UUID, reason schema.ModerationReason) error {
	// We update the status and store the reason.
	// We assume 'ModerationReason' is a string or compatible enum in the DB.
	return r.db.WithContext(ctx).Model(&schema.Review{}).
		Where("id = ?", reviewID).
		Updates(map[string]any{
			"status":            schema.ReviewStatusHidden,
			"moderation_reason": reason,
			"updated_at":        time.Now(),
		}).Error
}
