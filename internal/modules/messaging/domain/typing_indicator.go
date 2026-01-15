package domain

import (
	"time"

	"github.com/google/uuid"
)

// TypingIndicator represents a user's typing status in a conversation.
type TypingIndicator struct {
	ConversationID uuid.UUID `json:"conversation_id"`
	UserID         uuid.UUID `json:"user_id"`
	IsTyping       bool      `json:"is_typing"`
	Timestamp      time.Time `json:"timestamp"`
}

// NewTypingIndicator creates a new typing indicator.
func NewTypingIndicator(conversationID, userID uuid.UUID, isTyping bool) *TypingIndicator {
	return &TypingIndicator{
		ConversationID: conversationID,
		UserID:         userID,
		IsTyping:       isTyping,
		Timestamp:      time.Now(),
	}
}
