package domain

import (
	"time"

	"github.com/google/uuid"
)

type Participant struct {
	ID             uuid.UUID
	ConversationID uuid.UUID
	UserID         uuid.UUID

	Type ParticipantType

	JoinedAt   time.Time
	LeftAt     *time.Time
	LastReadAt *time.Time

	IsMuted   bool
	IsVisible bool // False for "shadow" support agents monitoring a chat
}

func NewParticipant(conversationID, userID uuid.UUID, pType ParticipantType) *Participant {
	return &Participant{
		ID:             uuid.New(),
		ConversationID: conversationID,
		UserID:         userID,
		Type:           pType,
		JoinedAt:       time.Now(),
		IsVisible:      true,
		IsMuted:        false,
	}
}
