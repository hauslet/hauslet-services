package domain

import (
	"time"

	"github.com/google/uuid"
)

type Conversation struct {
	ID     uuid.UUID
	Type   ConversationType
	Status string // "active", "archived"

	// Polymorphic Context
	ContextType ConversationContextType
	ContextID   uuid.UUID

	CreatedAt     time.Time
	UpdatedAt     time.Time
	LastMessageAt *time.Time

	// Domain Objects
	UnreadCounts map[uuid.UUID]int
	SupportState *SupportState
	Metadata     map[string]interface{}

	// Relations
	Participants []Participant
	Messages     []Message
}

// CanUserParticipate checks if a user is a member of this conversation
func (c *Conversation) CanUserParticipate(userID uuid.UUID) bool {
	for _, p := range c.Participants {
		if p.UserID == userID {
			return true
		}
	}
	return false
}

// GetParticipant finds a specific participant
func (c *Conversation) GetParticipant(userID uuid.UUID) *Participant {
	for _, p := range c.Participants {
		if p.UserID == userID {
			return &p
		}
	}
	return nil
}

// IsSupport check
func (c *Conversation) IsSupport() bool {
	return c.Type == ConversationTypeSupport
}

// MarkRead resets the unread count for a specific user
func (c *Conversation) MarkRead(userID uuid.UUID) {
	if c.UnreadCounts == nil {
		c.UnreadCounts = make(map[uuid.UUID]int)
	}
	c.UnreadCounts[userID] = 0
}
