package service

import (
	"context"
	"fmt"
	"time"

	"hauslet/internal/modules/review/domain"
	"hauslet/internal/modules/review/notification"

	"github.com/google/uuid"
)

// ================================================================
// RESPONSE OPERATIONS
// ================================================================

// CreateResponse creates a host's response to a review
func (s *ReviewServiceImpl) CreateResponse(ctx context.Context, reviewID, authorID uuid.UUID, body string) (*domain.ReviewResponse, error) {
	// 1. Validate body
	if body == "" {
		return nil, domain.ErrResponseBodyRequired
	}

	// 2. Get review
	schemaReview, err := s.reviewRepo.GetByID(ctx, reviewID)
	if err != nil {
		return nil, fmt.Errorf("failed to get review: %w", err)
	}
	if schemaReview == nil {
		return nil, domain.ErrReviewNotFound
	}

	review := domain.MapReviewFromSchema(schemaReview)

	// 3. Check if review is published
	if !review.IsPublished() {
		return nil, domain.ErrCannotRespondToUnpublished
	}

	// 4. Authorization: only the target (host) can respond
	// TODO: Verify authorID is the host/target of the review
	// For now, we trust the caller

	// 5. Check for existing response
	existingResponse, err := s.responseRepo.GetByReviewID(ctx, reviewID)
	if err == nil && existingResponse != nil {
		return nil, domain.ErrResponseAlreadyExists
	}

	// 6. Create response
	response := &domain.ReviewResponse{
		ID:        uuid.New(),
		ReviewID:  reviewID,
		AuthorID:  authorID,
		Body:      body,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// 7. Save to repository
	schemaResponse := domain.MapReviewResponseToSchema(response)
	if err := s.responseRepo.Create(ctx, schemaResponse); err != nil {
		return nil, fmt.Errorf("failed to create response: %w", err)
	}

	s.log.Info("created response", "response_id", response.ID, "review_id", reviewID)

	// 8. Send notification to guest (reviewer)
	guestName, guestEmail, err := s.userQuerier.GetUserContact(ctx, review.ReviewerID)
	if err != nil {
		s.log.Warn("failed to get guest contact for response notification", "error", err)
	} else if guestEmail != "" {
		guestContact := notification.ContactInfo{
			ID:    review.ReviewerID,
			Name:  guestName,
			Email: guestEmail,
		}
		// TODO: Get listing title - needs ListingQuerier interface
		// For now using empty string, notification service can handle it
		s.notificationSvc.SendResponseCreated(ctx, review, response, guestContact, "")
	}

	return response, nil
}

// GetResponse retrieves a response by review ID
func (s *ReviewServiceImpl) GetResponse(ctx context.Context, reviewID uuid.UUID) (*domain.ReviewResponse, error) {
	schemaResponse, err := s.responseRepo.GetByReviewID(ctx, reviewID)
	if err != nil {
		return nil, fmt.Errorf("failed to get response: %w", err)
	}
	if schemaResponse == nil {
		return nil, domain.ErrResponseNotFound
	}

	return domain.MapReviewResponseFromSchema(schemaResponse), nil
}

// UpdateResponse updates an existing response
func (s *ReviewServiceImpl) UpdateResponse(ctx context.Context, responseID, authorID uuid.UUID, body string) (*domain.ReviewResponse, error) {
	// 1. Validate body
	if body == "" {
		return nil, domain.ErrResponseBodyRequired
	}

	// 2. Get existing response
	schemaResponse, err := s.responseRepo.GetByReviewID(ctx, responseID)
	if err != nil {
		return nil, fmt.Errorf("failed to get response: %w", err)
	}
	if schemaResponse == nil {
		return nil, domain.ErrResponseNotFound
	}

	response := domain.MapReviewResponseFromSchema(schemaResponse)

	// 3. Authorization: only author can update
	if response.AuthorID != authorID {
		return nil, domain.ErrUnauthorized
	}

	// 4. Update
	response.Body = body
	response.UpdatedAt = time.Now()

	// 5. Save
	schemaResponse = domain.MapReviewResponseToSchema(response)
	if err := s.responseRepo.Update(ctx, schemaResponse); err != nil {
		return nil, fmt.Errorf("failed to update response: %w", err)
	}

	s.log.Info("updated response", "response_id", responseID)

	return response, nil
}

// DeleteResponse deletes a response
func (s *ReviewServiceImpl) DeleteResponse(ctx context.Context, responseID, actorID uuid.UUID) error {
	// 1. Get existing response
	schemaResponse, err := s.responseRepo.GetByReviewID(ctx, responseID)
	if err != nil {
		return fmt.Errorf("failed to get response: %w", err)
	}
	if schemaResponse == nil {
		return domain.ErrResponseNotFound
	}

	response := domain.MapReviewResponseFromSchema(schemaResponse)

	// 2. Authorization: only author can delete
	// TODO: Add admin check
	if response.AuthorID != actorID {
		return domain.ErrUnauthorized
	}

	// 3. Delete
	if err := s.responseRepo.Delete(ctx, responseID); err != nil {
		return fmt.Errorf("failed to delete response: %w", err)
	}

	s.log.Info("deleted response", "response_id", responseID)

	return nil
}
