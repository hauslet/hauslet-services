package domain

import (
	"time"

	"github.com/google/uuid"
)

// ReviewResponse represents a host's reply to a review
type ReviewResponse struct {
	ID uuid.UUID

	// The review being responded to (One-to-One relationship)
	ReviewID uuid.UUID

	// The response author (usually the host)
	AuthorID uuid.UUID
	Body     string

	// Timestamps
	CreatedAt time.Time
	UpdatedAt time.Time
}

// IsEmpty returns true if the response body is empty
func (r *ReviewResponse) IsEmpty() bool {
	return r.Body == ""
}
