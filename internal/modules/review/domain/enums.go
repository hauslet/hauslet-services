package domain

// ReviewTargetType defines what entity is being reviewed
type ReviewTargetType string

const (
	ReviewTargetListing    ReviewTargetType = "listing"
	ReviewTargetHost       ReviewTargetType = "host"
	ReviewTargetExperience ReviewTargetType = "experience"
)

// ReviewStatus defines the current state of a review
type ReviewStatus string

const (
	ReviewStatusPending   ReviewStatus = "pending"   // Initial state, awaiting approval
	ReviewStatusPublished ReviewStatus = "published" // Visible to public
	ReviewStatusHidden    ReviewStatus = "hidden"    // Hidden by admin
	ReviewStatusArchived  ReviewStatus = "archived"  // Soft delete logic
	ReviewStatusStandoff  ReviewStatus = "standoff"  // Waiting for counterparty review
)

// ModerationReason defines why a review was moderated
type ModerationReason string

const (
	ModerationReasonSpam            ModerationReason = "spam"
	ModerationReasonOffensive       ModerationReason = "offensive"
	ModerationReasonFraudulent      ModerationReason = "fraudulent"
	ModerationReasonIrrelevant      ModerationReason = "irrelevant"
	ModerationReasonPersonalInfo    ModerationReason = "personal_info"
	ModerationReasonDuplicateReview ModerationReason = "duplicate_review"
	ModerationReasonOther           ModerationReason = "other"
)

// String methods for enums
func (r ReviewTargetType) String() string {
	return string(r)
}

func (r ReviewStatus) String() string {
	return string(r)
}

func (m ModerationReason) String() string {
	return string(m)
}
