package domain

import (
	"time"

	"github.com/google/uuid"
)

type ContentType string
type ModerationStatus string
type ReviewerType string

const (
	ReviewerTypeAI        ReviewerType = "ai"
	ReviewerTypeModerator ReviewerType = "moderator"
)

const (
	ContentTypeListingText     ContentType = "listing_text"
	ContentTypeListingImage    ContentType = "listing_image"
	ContentTypeListingVideo    ContentType = "listing_video"
	ContentTypeProfileBio      ContentType = "profile_bio"
	ContentTypeProfileImage    ContentType = "profile_image"
	ContentTypeTravelCompImage ContentType = "tc_image"
	ContentTypeReviewText      ContentType = "review_text"
	ContentTypeReviewImage     ContentType = "review_image"
)

const (
	ModerationStatusPending   ModerationStatus = "pending"
	ModerationStatusAccepted  ModerationStatus = "accepted"
	ModerationStatusEscalated ModerationStatus = "escalated"
	ModerationStatusRejected  ModerationStatus = "rejected"
)

type Moderation struct {
	ID             uuid.UUID        `json:"id"`
	ContentType    ContentType      `json:"content_type"`
	TargetID       uuid.UUID        `json:"target_id"`
	AttemptCount   int              `json:"attempt_count"`
	CurrentAttempt int              `json:"current_attempt"`
	Status         ModerationStatus `json:"status"`
	ReviewerType   ReviewerType     `json:"reviewer_type"`
	ReviewerID     *uuid.UUID       `json:"reviewer_id,omitempty"`
	AIConfidence   *float64         `json:"ai_confidence,omitempty"`
	Reason         string           `json:"reason,omitempty"`
	CreatedAt      time.Time        `json:"created_at"`
	UpdatedAt      time.Time        `json:"updated_at"`
}

func (ct ContentType) IsListingContent() bool {
	return ct == ContentTypeListingText || ct == ContentTypeListingImage || ct == ContentTypeListingVideo
}

func (ct ContentType) IsProfileContent() bool {
	return ct == ContentTypeProfileBio || ct == ContentTypeProfileImage || ct == ContentTypeTravelCompImage
}

func (ct ContentType) IsReviewContent() bool {
	return ct == ContentTypeReviewText || ct == ContentTypeReviewImage
}

// func (ct ContentType) IsTravelCompContent() bool {
// 	return ct == ContentTypeTravelCompImage
// }

type TargetUserDetails struct {
	Email              string
	Name               string
	ContentDescription string
}
