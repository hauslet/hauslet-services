package graphql

import (
	"context"
	"fmt"
	"hauslet/internal/modules/review/domain"
	"hauslet/internal/transport/graph/viewer"

	"github.com/google/uuid"
)

// ============================================================================
// Response Query Resolvers 
// ============================================================================

// ReviewResponse retrieves a response for a review
func (r *Resolver) ReviewResponse(ctx context.Context, reviewID string) (*domain.ReviewResponse, error) {
	rid, err := uuid.Parse(reviewID)
	if err != nil {
		r.log.Error("invalid review ID", "review_id", reviewID, "error", err)
		return nil, fmt.Errorf("invalid review ID")
	}

	response, err := r.reviewService.GetResponse(ctx, rid)
	if err != nil {
		if err == domain.ErrResponseNotFound {
			return nil, nil
		}
		r.log.Error("failed to get response for review", "review_id", reviewID, "error", err)
		return nil, err
	}

	return response, nil
}

// ============================================================================
// Response Mutation Resolvers
// ============================================================================

// CreateResponse creates a response to a review (target owner only)
func (r *Resolver) CreateResponse(ctx context.Context, reviewID string, body string) (*domain.ReviewResponse, error) {
	userID, err := viewer.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	rid, err := uuid.Parse(reviewID)
	if err != nil {
		r.log.Error("invalid review ID", "review_id", reviewID, "error", err)
		return nil, fmt.Errorf("invalid review ID")
	}

	response, err := r.reviewService.CreateResponse(ctx, rid, userID, body)
	if err != nil {
		r.log.Error("failed to create response for review", "review_id", reviewID, "err", err)
		return nil, err
	}

	r.log.Info("review response created", "id", response.ID, "review_id", reviewID, "author_id", userID)

	return response, nil
}

// UpdateResponse updates an existing response
func (r *Resolver) UpdateResponse(ctx context.Context, responseID string, body string) (*domain.ReviewResponse, error) {
	userID, err := viewer.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	respID, err := uuid.Parse(responseID)
	if err != nil {
		r.log.Error("invalid response ID", "response_id", responseID, "error", err)
		return nil, fmt.Errorf("invalid response ID")
	}

	response, err := r.reviewService.UpdateResponse(ctx, respID, userID, body)
	if err != nil {
		r.log.Error("failed to update response", "response_id", responseID, "err", err)
		return nil, err
	}

	r.log.Info("review response updated", "id", responseID, "author_id", userID)

	return response, nil
}

// DeleteResponse deletes a response
func (r *Resolver) DeleteResponse(ctx context.Context, responseID string) (bool, error) {
	userID, err := viewer.GetUserIDFromContext(ctx)
	if err != nil {
		return false, err
	}

	respID, err := uuid.Parse(responseID)
	if err != nil {
		r.log.Error("invalid response ID", "response_id", responseID, "error", err)
		return false, fmt.Errorf("invalid response ID")
	}

	if err := r.reviewService.DeleteResponse(ctx, respID, userID); err != nil {
		r.log.Error("failed to delete response", "response_id", responseID, "err", err)
		return false, err
	}

	r.log.Info("review response deleted", "id", responseID, "author_id", userID)

	return true, nil
}
