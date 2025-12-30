package service

import (
	"context"
	"fmt"
	"time"

	"hauslet/internal/modules/review/domain"
	"hauslet/internal/modules/review/notification"
	"hauslet/internal/modules/review/repository"
	"hauslet/internal/modules/review/repository/schema"

	moderationdomain "hauslet/internal/modules/moderation/domain"
	moderationservice "hauslet/internal/modules/moderation/service"

	"github.com/google/uuid"
)

// ================================================================
// REVIEW OPERATIONS
// ================================================================

// CreateReview creates a new review for a booking
func (s *ReviewServiceImpl) CreateReview(ctx context.Context, input CreateReviewInput) (*domain.Review, error) {
	// 1. Validate booking completion
	isCompleted, err := s.bookingQuerier.IsBookingCompleted(ctx, input.BookingID)
	if err != nil {
		return nil, fmt.Errorf("failed to check booking status: %w", err)
	}
	if !isCompleted {
		return nil, domain.ErrReviewNotEditable // Booking must be completed
	}

	// 2. Verify reviewer is part of the booking (authorization)
	guestID, hostID, err := s.bookingQuerier.GetBookingParties(ctx, input.BookingID)
	if err != nil {
		return nil, fmt.Errorf("failed to get booking parties: %w", err)
	}

	if input.ReviewerID != guestID && input.ReviewerID != hostID {
		return nil, domain.ErrUnauthorized
	}

	// 3. Check for duplicate review
	existing, err := s.reviewRepo.GetByBookingAndReviewer(ctx, input.BookingID, input.ReviewerID)
	if err == nil && existing != nil {
		return nil, domain.ErrReviewAlreadyExists
	}

	// 4. Validate rating
	if input.Rating < 1 || input.Rating > 5 {
		return nil, domain.ErrInvalidRating
	}

	// 5. Validate sub-ratings
	if input.SubRatings != nil && !input.SubRatings.IsValid() {
		return nil, domain.ErrInvalidSubRatings
	}

	// 6. Validate review body
	if input.Body == "" {
		return nil, domain.ErrReviewBodyRequired
	}

	// 7. Create domain review
	review := &domain.Review{
		ID:                  uuid.New(),
		BookingID:           input.BookingID,
		TargetType:          input.TargetType,
		TargetID:            input.TargetID,
		ReviewerID:          input.ReviewerID,
		ReviewerCountryCode: input.CountryCode,
		Rating:              input.Rating,
		Title:               input.Title,
		Body:                input.Body,
		SubRatings:          input.SubRatings,
		Language:            input.Language,
		Status:              domain.ReviewStatusStandoff, // Default to standoff
		IsReported:          false,
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}

	// 8. Map to schema and create in repository
	schemaReview := domain.MapReviewToSchema(review)
	if err := s.reviewRepo.Create(ctx, schemaReview); err != nil {
		return nil, fmt.Errorf("failed to create review: %w", err)
	}

	s.log.Logf("[INFO] created review %s for booking %s by reviewer %s", review.ID, input.BookingID, input.ReviewerID)

	// 9. Enqueue AI moderation for review text
	if s.moderationSvc != nil {
		_, err = s.moderationSvc.EnqueueAIModeration(ctx, moderationservice.CreateModerationRequest{
			ContentType: moderationdomain.ContentTypeReviewText,
			TargetID:    review.ID,
			Payload:     review.Body,
		})
		if err != nil {
			s.log.Logf("[WARN] failed to enqueue moderation for review %s: %v", review.ID, err)
			// Don't fail the review creation if moderation fails
		}
	}

	// 10. Check for counterparty review (triggers auto-publish if both exist)
	// Note: This is also handled by repository AfterCreate hook
	if err := s.OnReviewCreated(ctx, review.ID); err != nil {
		s.log.Logf("[WARN] failed to process review creation hook for %s: %v", review.ID, err)
	}

	return review, nil
}

// GetReview retrieves a review by ID
func (s *ReviewServiceImpl) GetReview(ctx context.Context, reviewID, requestorID uuid.UUID) (*domain.Review, error) {
	schemaReview, err := s.reviewRepo.GetByID(ctx, reviewID)
	if err != nil {
		return nil, fmt.Errorf("failed to get review: %w", err)
	}
	if schemaReview == nil {
		return nil, domain.ErrReviewNotFound
	}

	review := domain.MapReviewFromSchema(schemaReview)

	// Authorization: only show published reviews to non-authors
	if review.ReviewerID != requestorID && !review.IsPublished() {
		return nil, domain.ErrUnauthorized
	}

	return review, nil
}

// UpdateReview updates an existing review
func (s *ReviewServiceImpl) UpdateReview(ctx context.Context, reviewID, actorID uuid.UUID, input UpdateReviewInput) (*domain.Review, error) {
	// 1. Get existing review
	schemaReview, err := s.reviewRepo.GetByID(ctx, reviewID)
	if err != nil {
		return nil, fmt.Errorf("failed to get review: %w", err)
	}
	if schemaReview == nil {
		return nil, domain.ErrReviewNotFound
	}

	review := domain.MapReviewFromSchema(schemaReview)

	// 2. Authorization: only author can update
	if review.ReviewerID != actorID {
		return nil, domain.ErrNotReviewAuthor
	}

	// 3. Validate: only unpublished reviews can be edited
	if review.IsPublished() {
		return nil, domain.ErrReviewNotEditable
	}

	// 4. Track original content for moderation check
	originalTitle := review.Title
	originalBody := review.Body
	contentChanged := false

	// 5. Apply updates
	if input.Rating != nil {
		if *input.Rating < 1 || *input.Rating > 5 {
			return nil, domain.ErrInvalidRating
		}
		review.Rating = *input.Rating
	}

	if input.Title != nil {
		if *input.Title != originalTitle {
			contentChanged = true
		}
		review.Title = *input.Title
	}

	if input.Body != nil {
		if *input.Body == "" {
			return nil, domain.ErrReviewBodyRequired
		}
		if *input.Body != originalBody {
			contentChanged = true
		}
		review.Body = *input.Body
	}

	if input.SubRatings != nil {
		if !input.SubRatings.IsValid() {
			return nil, domain.ErrInvalidSubRatings
		}
		review.SubRatings = input.SubRatings
	}

	review.UpdatedAt = time.Now()

	// 6. Re-trigger moderation if content changed
	// This prevents users from bypassing moderation by editing after approval
	if contentChanged && s.moderationSvc != nil {
		_, err = s.moderationSvc.EnqueueAIModeration(ctx, moderationservice.CreateModerationRequest{
			ContentType: moderationdomain.ContentTypeReviewText,
			TargetID:    review.ID,
			Payload:     review.Body,
		})
		if err != nil {
			s.log.Logf("[WARN] failed to re-enqueue moderation for updated review %s: %v", reviewID, err)
			// Don't fail the update if moderation enqueueing fails
		} else {
			s.log.Logf("[INFO] re-enqueued moderation for updated review %s (content changed)", reviewID)
		}
	}

	// 7. Save to repository
	schemaReview = domain.MapReviewToSchema(review)
	if err := s.reviewRepo.Update(ctx, schemaReview); err != nil {
		return nil, fmt.Errorf("failed to update review: %w", err)
	}

	s.log.Logf("[INFO] updated review %s", reviewID)

	return review, nil
}

// DeleteReview soft-deletes a review
func (s *ReviewServiceImpl) DeleteReview(ctx context.Context, reviewID, actorID uuid.UUID) error {
	// 1. Get existing review
	schemaReview, err := s.reviewRepo.GetByID(ctx, reviewID)
	if err != nil {
		return fmt.Errorf("failed to get review: %w", err)
	}
	if schemaReview == nil {
		return domain.ErrReviewNotFound
	}

	review := domain.MapReviewFromSchema(schemaReview)

	// 2. Authorization: only author can delete
	// TODO: Add admin check if needed
	if review.ReviewerID != actorID {
		return domain.ErrNotReviewAuthor
	}

	// 3. Delete
	if err := s.reviewRepo.Delete(ctx, reviewID); err != nil {
		return fmt.Errorf("failed to delete review: %w", err)
	}

	s.log.Logf("[INFO] deleted review %s", reviewID)

	return nil
}

// ListReviewsForTarget retrieves reviews for a specific target
func (s *ReviewServiceImpl) ListReviewsForTarget(ctx context.Context, targetType domain.ReviewTargetType, targetID uuid.UUID, filter ReviewFilter) ([]*domain.Review, int64, error) {
	// Convert service filter to repository filter
	repoFilter := repository.ReviewFilter{
		TargetID:    targetID,
		TargetType:  schema.ReviewTargetType(targetType),
		Limit:       filter.Limit,
		Offset:      filter.Offset,
		SortBy:      filter.SortBy,
		Rating:      filter.Rating,
		Language:    filter.Language,
		OnlyVisible: filter.OnlyVisible,
	}

	schemaReviews, total, err := s.reviewRepo.List(ctx, repoFilter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list reviews: %w", err)
	}

	// Map to domain
	reviews := make([]*domain.Review, 0, len(schemaReviews))
	for _, sr := range schemaReviews {
		reviews = append(reviews, domain.MapReviewFromSchema(&sr))
	}

	return reviews, total, nil
}

// ListUserReviews retrieves all reviews written by a user
func (s *ReviewServiceImpl) ListUserReviews(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.Review, error) {
	filter := repository.ReviewFilter{
		Limit:  limit,
		Offset: offset,
		SortBy: "newest",
	}

	schemaReviews, _, err := s.reviewRepo.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list user reviews: %w", err)
	}

	// Filter by reviewer ID (TODO: add to repository filter)
	reviews := make([]*domain.Review, 0)
	for _, sr := range schemaReviews {
		if sr.ReviewerID == userID {
			reviews = append(reviews, domain.MapReviewFromSchema(&sr))
		}
	}

	return reviews, nil
}

// GetReviewForBooking retrieves the review for a specific booking and reviewer
func (s *ReviewServiceImpl) GetReviewForBooking(ctx context.Context, bookingID, reviewerID uuid.UUID) (*domain.Review, error) {
	schemaReview, err := s.reviewRepo.GetByBookingAndReviewer(ctx, bookingID, reviewerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get review for booking: %w", err)
	}
	if schemaReview == nil {
		return nil, domain.ErrReviewNotFound
	}

	return domain.MapReviewFromSchema(schemaReview), nil
}

// PublishReview immediately publishes a review (admin action)
func (s *ReviewServiceImpl) PublishReview(ctx context.Context, reviewID, adminID uuid.UUID) error {
	// 1. Get existing review
	schemaReview, err := s.reviewRepo.GetByID(ctx, reviewID)
	if err != nil {
		return fmt.Errorf("failed to get review: %w", err)
	}
	if schemaReview == nil {
		return domain.ErrReviewNotFound
	}

	review := domain.MapReviewFromSchema(schemaReview)

	// 2. Check if already published
	if review.IsPublished() {
		return domain.ErrReviewAlreadyPublished
	}

	// 3. Check if hidden (cannot publish hidden reviews)
	if review.IsHidden() {
		return domain.ErrCannotPublishHiddenReview
	}

	// 4. Publish
	review.Publish()

	// 5. Save
	schemaReview = domain.MapReviewToSchema(review)
	if err := s.reviewRepo.Update(ctx, schemaReview); err != nil {
		return fmt.Errorf("failed to publish review: %w", err)
	}

	s.log.Logf("[INFO] admin %s published review %s", adminID, reviewID)

	// 6. Update booking review timestamp
	if s.bookingHooks != nil {
		reviewerType := s.determineReviewerType(ctx, review)
		if err := s.bookingHooks.OnReviewPublished(ctx, review.BookingID, review.ReviewerID, reviewerType); err != nil {
			s.log.Logf("[WARN] failed to update booking review timestamp for review %s: %v", reviewID, err)
			// Don't fail the publication
		}
	}

	// 7. Trigger stats recalculation
	if err := s.RecalculateListingStats(ctx, review.TargetID); err != nil {
		s.log.Logf("[WARN] failed to recalculate stats after publishing review %s: %v", reviewID, err)
	}

	return nil
}

// PublishExpiredStandoffs publishes reviews that have been in standoff for too long
func (s *ReviewServiceImpl) PublishExpiredStandoffs(ctx context.Context, olderThan time.Time) (int64, error) {
	count, err := s.reviewRepo.PublishExpiredStandoffs(ctx, olderThan)
	if err != nil {
		return 0, fmt.Errorf("failed to publish expired standoffs: %w", err)
	}

	s.log.Logf("[INFO] published %d expired standoff reviews", count)

	// TODO: Update booking review timestamps for published standoffs
	// This requires fetching all published reviews and calling bookingHooks.OnReviewPublished
	// for each one. For now, this is handled at the repository level or can be implemented
	// when needed by iterating through the published reviews.

	return count, nil
}

// ReportReview marks a review as reported for moderation
func (s *ReviewServiceImpl) ReportReview(ctx context.Context, reviewID, reporterID uuid.UUID, reason string) error {
	// 1. Get existing review
	schemaReview, err := s.reviewRepo.GetByID(ctx, reviewID)
	if err != nil {
		return fmt.Errorf("failed to get review: %w", err)
	}
	if schemaReview == nil {
		return domain.ErrReviewNotFound
	}

	review := domain.MapReviewFromSchema(schemaReview)

	// 2. Mark as reported
	review.MarkAsReported()

	// 3. Save
	schemaReview = domain.MapReviewToSchema(review)
	if err := s.reviewRepo.Update(ctx, schemaReview); err != nil {
		return fmt.Errorf("failed to report review: %w", err)
	}

	s.log.Logf("[INFO] review %s reported by user %s: %s", reviewID, reporterID, reason)

	// TODO: Send notification to admins
	// Requires: AdminQuerier interface with GetAdminEmails() method
	// Create adapter similar to internal/modules/finance/port/hooks/profile_adapter.go
	// Example:
	//   adminEmails, err := s.adminQuerier.GetAdminEmails(ctx)
	//   if err == nil {
	//     s.notificationSvc.SendReviewReportedToAdmins(ctx, review, adminEmails, reason)
	//   }

	return nil
}

// HideReview hides a review from public view (admin action)
func (s *ReviewServiceImpl) HideReview(ctx context.Context, reviewID, adminID uuid.UUID, reason domain.ModerationReason) error {
	// 1. Get existing review
	schemaReview, err := s.reviewRepo.GetByID(ctx, reviewID)
	if err != nil {
		return fmt.Errorf("failed to get review: %w", err)
	}
	if schemaReview == nil {
		return domain.ErrReviewNotFound
	}

	review := domain.MapReviewFromSchema(schemaReview)

	// 2. Check if already hidden
	if review.IsHidden() {
		return domain.ErrReviewAlreadyHidden
	}

	// 3. Hide with reason
	review.Hide(reason)

	// 4. Save
	schemaReview = domain.MapReviewToSchema(review)
	if err := s.reviewRepo.Update(ctx, schemaReview); err != nil {
		return fmt.Errorf("failed to hide review: %w", err)
	}

	s.log.Logf("[INFO] admin %s hid review %s (reason: %s)", adminID, reviewID, reason)

	// 5. Notify reviewer
	reviewerName, reviewerEmail, err := s.userQuerier.GetUserContact(ctx, review.ReviewerID)
	if err != nil {
		s.log.Logf("[WARN] failed to get reviewer contact for review %s: %v", reviewID, err)
	} else if reviewerEmail != "" {
		reviewerContact := notification.ContactInfo{
			ID:    review.ReviewerID,
			Name:  reviewerName,
			Email: reviewerEmail,
		}
		s.notificationSvc.SendReviewModerationRejected(ctx, review, reviewerContact, string(reason))
	}

	// 6. Recalculate stats (hidden review affects averages)
	if err := s.RecalculateListingStats(ctx, review.TargetID); err != nil {
		s.log.Logf("[WARN] failed to recalculate stats after hiding review %s: %v", reviewID, err)
	}

	return nil
}

// UnhideReview restores a hidden review to published status (admin action)
func (s *ReviewServiceImpl) UnhideReview(ctx context.Context, reviewID, adminID uuid.UUID) error {
	// 1. Get existing review
	schemaReview, err := s.reviewRepo.GetByID(ctx, reviewID)
	if err != nil {
		return fmt.Errorf("failed to get review: %w", err)
	}
	if schemaReview == nil {
		return domain.ErrReviewNotFound
	}

	review := domain.MapReviewFromSchema(schemaReview)

	// 2. Check if hidden
	if !review.IsHidden() {
		return nil // Already visible, nothing to do
	}

	// 3. Restore to published
	review.Status = domain.ReviewStatusPublished
	review.ModerationReason = nil
	review.UpdatedAt = time.Now()

	// 4. Save
	schemaReview = domain.MapReviewToSchema(review)
	if err := s.reviewRepo.Update(ctx, schemaReview); err != nil {
		return fmt.Errorf("failed to unhide review: %w", err)
	}

	s.log.Logf("[INFO] admin %s restored review %s", adminID, reviewID)

	// 5. Recalculate stats
	if err := s.RecalculateListingStats(ctx, review.TargetID); err != nil {
		s.log.Logf("[WARN] failed to recalculate stats after unhiding review %s: %v", reviewID, err)
	}

	return nil
}

// OnReviewCreated is called after a review is created to check for counterparty
func (s *ReviewServiceImpl) OnReviewCreated(ctx context.Context, reviewID uuid.UUID) error {
	// This is typically handled by repository AfterCreate hook
	// But we provide this method for manual triggering if needed

	// Get the newly created review
	schemaReview, err := s.reviewRepo.GetByID(ctx, reviewID)
	if err != nil {
		return fmt.Errorf("failed to get review: %w", err)
	}
	if schemaReview == nil {
		return domain.ErrReviewNotFound
	}

	review := domain.MapReviewFromSchema(schemaReview)

	// Check for counterparty review
	counterparty, err := s.reviewRepo.GetCounterpartReview(ctx, review.BookingID, review.ReviewerID)
	if err != nil {
		// No counterparty review yet, review stays in standoff
		return nil
	}

	// Both reviews exist - they should auto-publish via repository hook
	// Update booking timestamps and trigger stats recalculation
	if review.IsPublished() && counterparty != nil {
		// Update booking review timestamps for both reviews
		if s.bookingHooks != nil {
			// Update for current review
			reviewerType := s.determineReviewerType(ctx, review)
			if err := s.bookingHooks.OnReviewPublished(ctx, review.BookingID, review.ReviewerID, reviewerType); err != nil {
				s.log.Logf("[WARN] failed to update booking timestamp for review %s: %v", reviewID, err)
			}

			// Update for counterparty review
			counterpartyReview := domain.MapReviewFromSchema(counterparty)
			counterpartyType := s.determineReviewerType(ctx, counterpartyReview)
			if err := s.bookingHooks.OnReviewPublished(ctx, counterpartyReview.BookingID, counterpartyReview.ReviewerID, counterpartyType); err != nil {
				s.log.Logf("[WARN] failed to update booking timestamp for counterparty review %s: %v", counterpartyReview.ID, err)
			}
		}

		// Recalculate stats
		if err := s.RecalculateListingStats(ctx, review.TargetID); err != nil {
			s.log.Logf("[WARN] failed to recalculate stats for review %s: %v", reviewID, err)
		}
	}

	return nil
}
