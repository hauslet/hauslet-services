package domain

import (
	"time"

	"github.com/google/uuid"
)

// Review represents a guest or host review in the domain
type Review struct {
	ID uuid.UUID

	// Booking Context
	BookingID uuid.UUID

	// Polymorphic Relations (Who is being reviewed?)
	TargetType ReviewTargetType
	TargetID   uuid.UUID

	// The Author
	ReviewerID          uuid.UUID
	ReviewerCountryCode string

	// Content
	Rating   int // 1-5 star rating
	Title    string
	Body     string
	Language string // e.g., 'en-US'

	// Sub-ratings for detailed feedback
	SubRatings *SubRatings

	// State Management
	Status ReviewStatus

	// Moderation
	IsReported       bool
	ModerationReason *ModerationReason

	// Response Mechanism (Has the host replied?)
	ResponseID *uuid.UUID

	// Timestamps
	PublishedAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time
}

// SubRatings represents detailed category ratings for a review
type SubRatings struct {
	Cleanliness   float64 // Rating for property cleanliness (1.0-5.0)
	Accuracy      float64 // How accurate the listing description was (1.0-5.0)
	CheckIn       float64 // Check-in experience rating (1.0-5.0)
	Communication float64 // Host communication quality (1.0-5.0)
	Location      float64 // Location rating (1.0-5.0)
	Value         float64 // Value for money rating (1.0-5.0)
}

// IsPublished returns true if the review is visible to the public
func (r *Review) IsPublished() bool {
	return r.Status == ReviewStatusPublished
}

// IsHidden returns true if the review has been hidden by moderation
func (r *Review) IsHidden() bool {
	return r.Status == ReviewStatusHidden
}

// IsStandoff returns true if the review is waiting for counterparty
func (r *Review) IsStandoff() bool {
	return r.Status == ReviewStatusStandoff
}

// Publish marks the review as published
func (r *Review) Publish() {
	now := time.Now()
	r.Status = ReviewStatusPublished
	r.PublishedAt = &now
	r.UpdatedAt = now
}

// Hide marks the review as hidden with a moderation reason
func (r *Review) Hide(reason ModerationReason) {
	r.Status = ReviewStatusHidden
	r.ModerationReason = &reason
	r.UpdatedAt = time.Now()
}

// MarkAsReported marks the review as reported for moderation
func (r *Review) MarkAsReported() {
	r.IsReported = true
	r.UpdatedAt = time.Now()
}

// Average returns the average of all sub-ratings (0.0 if no sub-ratings)
func (s *SubRatings) Average() float64 {
	if s == nil {
		return 0.0
	}

	sum := s.Cleanliness + s.Accuracy + s.CheckIn + s.Communication + s.Location + s.Value
	return sum / 6.0
}

// IsValid validates that all sub-rating values are within 1.0-5.0 range
func (s *SubRatings) IsValid() bool {
	if s == nil {
		return true // nil is considered valid (optional sub-ratings)
	}

	ratings := []float64{s.Cleanliness, s.Accuracy, s.CheckIn, s.Communication, s.Location, s.Value}
	for _, rating := range ratings {
		if rating < 1.0 || rating > 5.0 {
			return false
		}
	}
	return true
}
