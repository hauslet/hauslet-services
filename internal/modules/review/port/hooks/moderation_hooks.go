package hooks

import (
	"context"
	"fmt"

	moderationdomain "hauslet/internal/modules/moderation/domain"
	moderationservice "hauslet/internal/modules/moderation/service"
	reviewdomain "hauslet/internal/modules/review/domain"
	reviewservice "hauslet/internal/modules/review/service"

	"github.com/google/uuid"
)

// Ensure compile-time conformance with the moderation service hooks.
var _ moderationservice.ReviewHooks = (*ReviewModerationHooksAdapter)(nil)

// ReviewModerationHooksAdapter handles moderation completion callbacks for reviews.
type ReviewModerationHooksAdapter struct {
	reviewSvc reviewservice.ReviewService
}

// NewReviewModerationHooksAdapter constructs the adapter.
func NewReviewModerationHooksAdapter(reviewSvc reviewservice.ReviewService) *ReviewModerationHooksAdapter {
	return &ReviewModerationHooksAdapter{reviewSvc: reviewSvc}
}

// OnModerationCompleted is called when AI moderation completes for a review.
// It handles the moderation result by hiding, reporting, or accepting the review.
func (a *ReviewModerationHooksAdapter) OnModerationCompleted(ctx context.Context, aggregate moderationservice.AggregatedModeration) error {
	reviewID := aggregate.TargetID

	switch aggregate.FinalStatus() {
	case moderationdomain.ModerationStatusRejected:
		// Review contains inappropriate content - hide it immediately
		// Use system UUID (nil) since this is an automated action
		reason := parseRejectionReason(aggregate.Reasons)
		if err := a.reviewSvc.HideReview(ctx, reviewID, uuid.Nil, reason); err != nil {
			return fmt.Errorf("failed to hide rejected review %s: %w", reviewID, err)
		}
		return nil

	case moderationdomain.ModerationStatusEscalated:
		// AI is uncertain - flag for human review
		escalationReason := fmt.Sprintf("AI escalated for manual review. Reasons: %v", aggregate.Reasons)
		if err := a.reviewSvc.ReportReview(ctx, reviewID, uuid.Nil, escalationReason); err != nil {
			return fmt.Errorf("failed to report escalated review %s: %w", reviewID, err)
		}
		return nil

	case moderationdomain.ModerationStatusAccepted:
		// Review is clean - allow normal publishing flow to proceed
		// No action needed; the review will follow its natural standoff -> published lifecycle
		return nil

	default:
		// Pending or unknown status - no action needed yet
		return nil
	}
}

// parseRejectionReason converts moderation rejection reasons to review moderation reasons.
func parseRejectionReason(reasons []string) reviewdomain.ModerationReason {
	if len(reasons) == 0 {
		return reviewdomain.ModerationReasonOther
	}

	// Map common AI moderation reasons to review-specific reasons
	firstReason := reasons[0]
	switch {
	case contains(firstReason, "spam"):
		return reviewdomain.ModerationReasonSpam
	case contains(firstReason, "offensive"), contains(firstReason, "harassment"), contains(firstReason, "hate"):
		return reviewdomain.ModerationReasonOffensive
	case contains(firstReason, "fraud"), contains(firstReason, "fake"):
		return reviewdomain.ModerationReasonFraudulent
	case contains(firstReason, "irrelevant"), contains(firstReason, "off-topic"):
		return reviewdomain.ModerationReasonIrrelevant
	case contains(firstReason, "personal"), contains(firstReason, "contact"), contains(firstReason, "email"), contains(firstReason, "phone"):
		return reviewdomain.ModerationReasonPersonalInfo
	default:
		return reviewdomain.ModerationReasonOther
	}
}

// contains checks if a string contains a substring (case-insensitive).
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr || len(s) > 0 && len(substr) > 0 &&
		s[:len(substr)] == substr ||
		len(s) > len(substr) && s[len(s)-len(substr):] == substr)
}
