package domain

import (
	"time"

	"github.com/google/uuid"
)

type SupportState struct {
	Status          SupportStatus `json:"status"`
	AssignedAgentID *uuid.UUID    `json:"assigned_agent_id,omitempty"`
	TicketID        string        `json:"ticket_id,omitempty"` // External ticket ref
	Priority        string        `json:"priority"`            // low, medium, high, urgent

	AISessionID    string    `json:"ai_session_id,omitempty"`
	LastAIResponse time.Time `json:"last_ai_response,omitempty"`
}

// NewSupportState creates a default state for new support conversations
func NewSupportState() *SupportState {
	return &SupportState{
		Status:   SupportStatusAIActive,
		Priority: "medium",
	}
}
