package jobs

import (
	"errors"

	"github.com/google/uuid"
)

type ContentType string

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

const AIModerationJobType = "ai_moderation"

// AIModerationJob represents a request to moderate content using AI.
type AIModerationJob struct {
	ModerationID uuid.UUID `json:"moderation_id"`

	// TargetID is the UUID of the Listing or Profile being checked (for context).
	TargetID uuid.UUID `json:"target_id"`

	// ContentType ensures the worker knows which AI prompt/model to use.
	ContentType ContentType `json:"content_type"`

	// Payload holds the actual data to check.
	// Use string for Text content or a URL for Images.
	Payload string `json:"payload"`
}

func (j *AIModerationJob) Type() string {
	return AIModerationJobType
}

func (j *AIModerationJob) Validate() error {
	if j.ModerationID == uuid.Nil {
		return errors.New("moderation_id is required")
	}
	if j.TargetID == uuid.Nil {
		return errors.New("target_id is required")
	}
	if j.ContentType == "" {
		return errors.New("content_type is required")
	}
	if j.Payload == "" {
		return errors.New("payload content is required")
	}
	return nil
}
