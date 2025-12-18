package schema

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
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
	ID                uuid.UUID        `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	ContentType       ContentType      `gorm:"type:varchar(50);not null;index:idx_content_type_id" json:"content_type"` // Composite Index
	ContentID         uuid.UUID        `gorm:"type:uuid;not null;index:idx_content_type_id" json:"content_id"`          // Composite Index
	MaxAIAttemptCount int              `gorm:"not null;default:3" json:"max_ai_attempt_count"`
	CurrentAttempt    int              `gorm:"not null;default:0" json:"current_attempt"`
	Status            ModerationStatus `gorm:"type:varchar(50);not null;index" json:"status"`
	ReviewerType      ReviewerType     `gorm:"type:varchar(50);not null" json:"reviewer_type"`
	ReviewerID        *uuid.UUID       `gorm:"type:uuid" json:"reviewer_id,omitempty"`
	AIConfidence      *float64         `gorm:"type:float" json:"ai_confidence,omitempty"`
	Reason            string           `gorm:"type:text" json:"reason"`
	Metadata          string           `gorm:"type:jsonb" json:"metadata,omitempty"` // Additional metadata as JSON
	UpdatedAt         time.Time
	DeletedAt         gorm.DeletedAt `gorm:"index"`
}

// BeforeCreate runs once before inserting a new record.
// Use this for setting defaults and generating UUIDs if not relying purely on DB defaults.
func (m *Moderation) BeforeCreate(tx *gorm.DB) (err error) {
	// 1. Ensure UUID is set (Go-side generation is often safer for immediate usage)
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}

	// 2. Set Default Status if missing
	if m.Status == "" {
		m.Status = ModerationStatusPending
	}

	// 3. Set Default Max Attempts if not specified
	if m.MaxAIAttemptCount == 0 {
		m.MaxAIAttemptCount = 3
	}

	// 4. Default ReviewerType logic (optional)
	if m.ReviewerType == "" {
		// Assume AI by default if not specified? Or return error?
		m.ReviewerType = ReviewerTypeAI
	}

	return nil
}

func (m *Moderation) BeforeSave(tx *gorm.DB) (err error) {
	// 1. Validate Enums (Prevent saving invalid strings)
	if !isValidContentType(m.ContentType) {
		return fmt.Errorf("invalid content_type: %s", m.ContentType)
	}

	if !isValidStatus(m.Status) {
		return fmt.Errorf("invalid moderation_status: %s", m.Status)
	}

	// 2. Business Logic: Completed moderation must have a reason?
	// If status is Rejected, Reason should probably be mandatory.
	if m.Status == ModerationStatusRejected && m.Reason == "" {
		return errors.New("reason is required when status is rejected")
	}

	// 3. Business Logic: AI Consistency
	// If Reviewer is AI, ensure confidence score is set when accepted.
	if m.ReviewerType == ReviewerTypeAI && m.Status == ModerationStatusAccepted {
		if m.AIConfidence == nil {
			return errors.New("AI accepted content must have confidence score")
		}
	}

	return nil
}

// ---------------------------------------------------------
// HELPER VALIDATORS
// ---------------------------------------------------------

func isValidContentType(ct ContentType) bool {
	switch ct {
	case ContentTypeListingText, ContentTypeListingImage, ContentTypeListingVideo,
		ContentTypeProfileBio, ContentTypeProfileImage, ContentTypeTravelCompImage,
		ContentTypeReviewText, ContentTypeReviewImage:
		return true
	}
	return false
}

func isValidStatus(s ModerationStatus) bool {
	switch s {
	case ModerationStatusPending, ModerationStatusAccepted,
		ModerationStatusEscalated, ModerationStatusRejected:
		return true
	}
	return false
}
