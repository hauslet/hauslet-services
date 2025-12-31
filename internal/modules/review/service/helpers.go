package service

import (
	"context"

	"hauslet/internal/modules/review/domain"
)

// determineReviewerType checks if the reviewer is the guest or host of the booking
func (s *ReviewServiceImpl) determineReviewerType(ctx context.Context, review *domain.Review) string {
	guestID, hostID, err := s.bookingQuerier.GetBookingParties(ctx, review.BookingID)
	if err != nil {
		s.log.Warn("failed to get booking parties for review", "review_id", review.ID, "error", err)
		return "unknown"
	}

	if review.ReviewerID == guestID {
		return "guest"
	} else if review.ReviewerID == hostID {
		return "host"
	}

	return "unknown"
}
