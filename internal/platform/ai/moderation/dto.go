package aimoderation

import (
	"errors"
	"hauslet/internal/modules/moderation/repository/schema"

	"github.com/google/uuid"
)

type ImagePayload struct {
	Key      string `json:"key,omitempty"`
	MimeType string `json:"mime_type"` // e.g. "image/jpeg"
}

type VideoPayload struct {
	Key             string `json:"key"`
	MimeType        string `json:"mime_type"`
	DurationSeconds int    `json:"duration_seconds,omitempty"`
}

type AIModerationInput struct {
	ModerationID uuid.UUID
	ContentType  schema.ContentType
	ContentID    uuid.UUID

	// The Payload (OneOf)
	Text  *string
	Image *ImagePayload
	Video *VideoPayload

	// Context & Metadata
	AttemptNumber int
	MaxAttempts   int
	Language      string // e.g. "en-UK"
}

// Validate ensures we aren't sending empty garbage to the AI (which costs money).
func (i *AIModerationInput) Validate() error {
	hasText := i.Text != nil && *i.Text != ""
	hasImage := i.Image != nil && i.Image.Key != ""
	hasVideo := i.Video != nil && i.Video.Key != ""

	if !hasText && !hasImage && !hasVideo {
		return errors.New("input payload is empty: must provide text, image, or video")
	}
	return nil
}

type AIModerationResult struct {
	Status     schema.ModerationStatus
	Confidence float64 // Changed to value (0.0 default is fine)
	Reason     string

	// RawResponse is useful for debugging or storing the full JSON
	// from the provider if you need to audit why the AI made a decision.
	RawResponse map[string]any
}
