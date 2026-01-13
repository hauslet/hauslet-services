package hooks

import (
	"context"
	"fmt"
	"strings"

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
	// 1. Define a mapping of keywords to reasons.
	// This makes it easy to add new rules without touching the logic.
	// Order matters: put specific keywords above generic ones if needed.
	rules := []struct {
		keyword string
		reason  reviewdomain.ModerationReason
	}{
		{"spam", reviewdomain.ModerationReasonSpam},
		{"offensive", reviewdomain.ModerationReasonOffensive},
		{"harassment", reviewdomain.ModerationReasonOffensive},
		{"hate", reviewdomain.ModerationReasonOffensive},
		{"fraud", reviewdomain.ModerationReasonFraudulent},
		{"fake", reviewdomain.ModerationReasonFraudulent},
		{"irrelevant", reviewdomain.ModerationReasonIrrelevant},
		{"off-topic", reviewdomain.ModerationReasonIrrelevant},
		{"personal", reviewdomain.ModerationReasonPersonalInfo},
		{"contact", reviewdomain.ModerationReasonPersonalInfo},
		{"email", reviewdomain.ModerationReasonPersonalInfo},
		{"phone", reviewdomain.ModerationReasonPersonalInfo},
	}

	// 2. Iterate through ALL reasons provided, not just the first one.
	for _, reason := range reasons {
		// Normalize the input to lowercase to ensure matching works
		normalized := strings.ToLower(reason)

		// Check against our rules
		for _, rule := range rules {
			if strings.Contains(normalized, rule.keyword) {
				return rule.reason
			}
		}
	}

	// 3. Fallback if no keywords matched
	return reviewdomain.ModerationReasonOther
}
