package graphql

import (
	"context"
	"fmt"
	"hauslet/internal/modules/review/domain"
	"hauslet/internal/modules/review/service"
	"hauslet/internal/transport/graph/viewer"

	"github.com/google/uuid"
)

// ============================================================================
// Review Query Resolvers
// ============================================================================

// Review retrieves a review by ID
func (r *Resolver) Review(ctx context.Context, id string) (*domain.Review, error) {
	reviewID, err := uuid.Parse(id)
	if err != nil {
		r.log.Error("invalid review ID", "review_id", id, "error", err)
		return nil, fmt.Errorf("invalid review ID")
	}

	userID, err := viewer.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	review, err := r.reviewService.GetReview(ctx, reviewID, userID)
	if err != nil {
		if err == domain.ErrReviewNotFound {
			return nil, nil
		}
		r.log.Error("failed to get review", "review_id", id, "error", err)
		return nil, err
	}

	return review, nil
}

// ReviewForBooking gets the current user's review for a booking
func (r *Resolver) ReviewForBooking(ctx context.Context, bookingID string) (*domain.Review, error) {
	bid, err := uuid.Parse(bookingID)
	if err != nil {
		r.log.Error("invalid booking ID", "booking_id", bookingID, "error", err)
		return nil, fmt.Errorf("invalid booking ID")
	}

	userID, err := viewer.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	review, err := r.reviewService.GetReviewForBooking(ctx, bid, userID)
	if err != nil {
		if err == domain.ErrReviewNotFound {
			return nil, nil
		}
		r.log.Error("failed to get review for booking", "booking_id", bookingID, "error", err)
		return nil, err
	}

	return review, nil
}

// Reviews lists reviews for a target (listing or host)
func (r *Resolver) Reviews(
	ctx context.Context,
	targetType domain.ReviewTargetType,
	targetID string,
	filter *ReviewFilterInput,
) ([]*domain.Review, error) {
	tid, err := uuid.Parse(targetID)
	if err != nil {
		r.log.Error("invalid target ID", "target_id", targetID, "error", err)
		return nil, fmt.Errorf("invalid target ID")
	}

	// Build filter from input
	svcFilter := service.ReviewFilter{
		Limit:       50, // default
		Offset:      0,
		OnlyVisible: true, // default to visible only
	}

	if filter != nil {
		if filter.Limit != nil && *filter.Limit > 0 {
			svcFilter.Limit = *filter.Limit
		}
		if filter.Offset != nil && *filter.Offset > 0 {
			svcFilter.Offset = *filter.Offset
		}
		if filter.Rating != nil {
			svcFilter.Rating = filter.Rating
		}
		if filter.Language != nil {
			svcFilter.Language = filter.Language
		}
		if filter.OnlyVisible != nil {
			svcFilter.OnlyVisible = *filter.OnlyVisible
		}
		if filter.SortBy != nil {
			svcFilter.SortBy = *filter.SortBy
		}
	}

	reviews, _, err := r.reviewService.ListReviewsForTarget(ctx, targetType, tid, svcFilter)
	if err != nil {
		r.log.Error("failed to list reviews for target", "target_type", targetType, "target_id", targetID, "error", err)
		return nil, err
	}

	return reviews, nil
}

// UserReviews lists reviews written by a specific user
func (r *Resolver) UserReviews(ctx context.Context, userID string, limit, offset *int) ([]*domain.Review, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		r.log.Error("invalid user ID", "user_id", userID, "error", err)
		return nil, fmt.Errorf("invalid user ID")
	}

	l := 50 // default limit
	if limit != nil && *limit > 0 {
		l = *limit
	}

	o := 0 // default offset
	if offset != nil && *offset > 0 {
		o = *offset
	}

	reviews, err := r.reviewService.ListUserReviews(ctx, uid, l, o)
	if err != nil {
		r.log.Error("failed to list reviews for user", "user_id", userID, "error", err)
		return nil, err
	}

	return reviews, nil
}

// ============================================================================
// Review Mutation Resolvers
// ============================================================================

// CreateReviewInput represents the input for creating a review
type CreateReviewInput struct {
	BookingID     string
	TargetType    domain.ReviewTargetType
	TargetID      string
	Rating        int
	Title         string
	Body          string
	Cleanliness   *int
	Accuracy      *int
	Communication *int
	Location      *int
	Checkin       *int
	Value         *int
	Language      *string
	CountryCode   *string
}

// CreateReview creates a new review
func (r *Resolver) CreateReview(ctx context.Context, input CreateReviewInput) (*domain.Review, error) {
	userID, err := viewer.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	bookingID, err := uuid.Parse(input.BookingID)
	if err != nil {
		r.log.Error("invalid booking ID", "booking_id", input.BookingID, "error", err)
		return nil, fmt.Errorf("invalid booking ID")
	}

	targetID, err := uuid.Parse(input.TargetID)
	if err != nil {
		r.log.Error("invalid target ID", "target_id", input.TargetID, "error", err)
		return nil, fmt.Errorf("invalid target ID")
	}

	// Build service input
	svcInput := service.CreateReviewInput{
		BookingID:  bookingID,
		ReviewerID: userID,
		TargetType: input.TargetType,
		TargetID:   targetID,
		Rating:     input.Rating,
		Title:      input.Title,
		Body:       input.Body,
	}

	// Add sub-ratings if provided
	if input.Cleanliness != nil || input.Accuracy != nil || input.Communication != nil ||
		input.Location != nil || input.Checkin != nil || input.Value != nil {
		svcInput.SubRatings = &domain.SubRatings{
			Cleanliness:   convertIntPtrToFloat(input.Cleanliness),
			Accuracy:      convertIntPtrToFloat(input.Accuracy),
			Communication: convertIntPtrToFloat(input.Communication),
			Location:      convertIntPtrToFloat(input.Location),
			CheckIn:       convertIntPtrToFloat(input.Checkin),
			Value:         convertIntPtrToFloat(input.Value),
		}
	}

	if input.Language != nil {
		svcInput.Language = *input.Language
	}
	if input.CountryCode != nil {
		svcInput.CountryCode = *input.CountryCode
	}

	review, err := r.reviewService.CreateReview(ctx, svcInput)
	if err != nil {
		r.log.Error("failed to create review", "error", err)
		return nil, err
	}

	r.log.Info("review created", "review_id", review.ID, "booking_id", bookingID, "reviewer_id", userID)

	return review, nil
}

// UpdateReviewInput represents the input for updating a review
type UpdateReviewInput struct {
	Rating        *int
	Title         *string
	Body          *string
	Cleanliness   *int
	Accuracy      *int
	Communication *int
	Location      *int
	Checkin       *int
	Value         *int
}

// UpdateReview updates an existing review
func (r *Resolver) UpdateReview(ctx context.Context, reviewID string, input UpdateReviewInput) (*domain.Review, error) {
	userID, err := viewer.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	rid, err := uuid.Parse(reviewID)
	if err != nil {
		r.log.Error("invalid review ID", "review_id", reviewID, "error", err)
		return nil, fmt.Errorf("invalid review ID")
	}

	// Build service input
	svcInput := service.UpdateReviewInput{
		Rating: input.Rating,
		Title:  input.Title,
		Body:   input.Body,
	}

	// Add sub-ratings if provided
	if input.Cleanliness != nil || input.Accuracy != nil || input.Communication != nil ||
		input.Location != nil || input.Checkin != nil || input.Value != nil {
		svcInput.SubRatings = &domain.SubRatings{
			Cleanliness:   convertIntPtrToFloat(input.Cleanliness),
			Accuracy:      convertIntPtrToFloat(input.Accuracy),
			Communication: convertIntPtrToFloat(input.Communication),
			Location:      convertIntPtrToFloat(input.Location),
			CheckIn:       convertIntPtrToFloat(input.Checkin),
			Value:         convertIntPtrToFloat(input.Value),
		}
	}

	review, err := r.reviewService.UpdateReview(ctx, rid, userID, svcInput)
	if err != nil {
		r.log.Error("failed to update review", "review_id", reviewID, "error", err)
		return nil, err
	}

	r.log.Info("review updated", "review_id", review.ID, "user_id", userID)

	return review, nil
}

// DeleteReview deletes a review (only if unpublished)
func (r *Resolver) DeleteReview(ctx context.Context, reviewID string) (bool, error) {
	userID, err := viewer.GetUserIDFromContext(ctx)
	if err != nil {
		return false, err
	}

	rid, err := uuid.Parse(reviewID)
	if err != nil {
		r.log.Error("invalid review ID", "review_id", reviewID, "error", err)
		return false, fmt.Errorf("invalid review ID")
	}

	if err := r.reviewService.DeleteReview(ctx, rid, userID); err != nil {
		r.log.Error("failed to delete review", "review_id", reviewID, "error", err)
		return false, err
	}

	r.log.Info("review deleted", "review_id", reviewID, "user_id", userID)

	return true, nil
}


// ReportReview reports a review for moderation
func (r *Resolver) ReportReview(ctx context.Context, reviewID string, reason string) (bool, error) {
	userID, err := viewer.GetUserIDFromContext(ctx)
	if err != nil {
		return false, err
	}

	rid, err := uuid.Parse(reviewID)
	if err != nil {
		r.log.Error("invalid review ID", "review_id", reviewID, "error", err)
		return false, fmt.Errorf("invalid review ID")
	}

	if err := r.reviewService.ReportReview(ctx, rid, userID, reason); err != nil {
		r.log.Error("failed to report review", "review_id", reviewID, "error", err)
		return false, err
	}

	r.log.Info("review reported", "review_id", reviewID, "reporter_id", userID, "reason", reason)

	return true, nil
}

// ============================================================================
// GraphQL Schema Types
// ============================================================================

// ReviewFilterInput represents filtering options for reviews
type ReviewFilterInput struct {
	Rating      *int
	Language    *string
	OnlyVisible *bool
	SortBy      *string
	Limit       *int
	Offset      *int
}

// ============================================================================
// Helper Functions
// ============================================================================

// convertIntPtrToFloat converts an int pointer to float64, returns 0 if nil
func convertIntPtrToFloat(val *int) float64 {
	if val == nil {
		return 0
	}
	return float64(*val)
}
