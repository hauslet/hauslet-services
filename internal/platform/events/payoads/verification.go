package payoads

import (
	"time"

	"github.com/google/uuid"
)

// VerificationCompletedPayload represents the data for verification completion events.
type VerificationCompletedPayload struct {
	SessionID        uuid.UUID  `json:"session_id"`
	UserID           uuid.UUID  `json:"user_id"`
	VerificationType string     `json:"verification_type"`
	VerificationTier string     `json:"verification_tier"`
	Status           string     `json:"status"` // approved or rejected
	TargetID         *uuid.UUID `json:"target_id,omitempty"`
	TargetType       string     `json:"target_type,omitempty"`
	RejectionReason  *string    `json:"rejection_reason,omitempty"`
	CompletedAt      time.Time  `json:"completed_at"`
}
