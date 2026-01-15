package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"

	"hauslet/internal/modules/messaging/repository/schema"
)

// --- Conversation Mappers ---

func MapConversationToDomain(s *schema.Conversation) (*Conversation, error) {
	if s == nil {
		return nil, nil
	}

	// 1. Map JSON Fields
	var unreadCounts map[uuid.UUID]int
	if len(s.UnreadCounts) > 0 && string(s.UnreadCounts) != "null" {
		if err := json.Unmarshal(s.UnreadCounts, &unreadCounts); err != nil {
			return nil, err
		}
	} else {
		unreadCounts = make(map[uuid.UUID]int)
	}

	var supportState *SupportState
	if len(s.SupportState) > 0 && string(s.SupportState) != "null" {
		if err := json.Unmarshal(s.SupportState, &supportState); err != nil {
			return nil, err
		}
	}

	var metadata map[string]any
	if len(s.Metadata) > 0 && string(s.Metadata) != "null" {
		if err := json.Unmarshal(s.Metadata, &metadata); err != nil {
			return nil, err
		}
	}

	// 2. Map Relations
	participants := make([]Participant, len(s.Participants))
	for i, p := range s.Participants {
		mapped, err := MapParticipantToDomain(&p)
		if err != nil {
			return nil, err
		}
		participants[i] = *mapped
	}

	messages := make([]Message, len(s.Messages))
	for i, m := range s.Messages {
		mapped, err := MapMessageToDomain(&m)
		if err != nil {
			return nil, err
		}
		messages[i] = *mapped
	}

	// 3. Construct Entity
	return &Conversation{
		ID:            s.ID,
		Type:          ConversationType(s.Type),
		Status:        s.Status,
		ContextType:   ConversationContextType(s.ContextType),
		ContextID:     s.ContextID,
		CreatedAt:     s.CreatedAt,
		UpdatedAt:     s.UpdatedAt,
		LastMessageAt: s.LastMessageAt,
		UnreadCounts:  unreadCounts,
		SupportState:  supportState,
		Metadata:      metadata,
		Participants:  participants,
		Messages:      messages,
	}, nil
}

func MapConversationToSchema(d *Conversation) (*schema.Conversation, error) {
	if d == nil {
		return nil, nil
	}

	// Marshal JSON fields
	// Note: We ignore errors here for empty maps -> empty json, but explicit handling is safer
	unreadJSON, _ := json.Marshal(d.UnreadCounts)
	if d.UnreadCounts == nil {
		unreadJSON = []byte("{}")
	}

	var stateJSON []byte
	if d.SupportState != nil {
		stateJSON, _ = json.Marshal(d.SupportState)
	}

	var metaJSON []byte
	if d.Metadata != nil {
		metaJSON, _ = json.Marshal(d.Metadata)
	}

	return &schema.Conversation{
		ID:            d.ID,
		Type:          string(d.Type),
		Status:        d.Status,
		ContextType:   string(d.ContextType),
		ContextID:     d.ContextID,
		CreatedAt:     d.CreatedAt,
		UpdatedAt:     d.UpdatedAt,
		LastMessageAt: d.LastMessageAt,
		UnreadCounts:  datatypes.JSON(unreadJSON),
		SupportState:  datatypes.JSON(stateJSON),
		Metadata:      datatypes.JSON(metaJSON),
	}, nil
}

// --- Message Mappers ---

func MapMessageToDomain(s *schema.Message) (*Message, error) {
	if s == nil {
		return nil, nil
	}

	var metadata map[string]any
	if len(s.Metadata) > 0 && string(s.Metadata) != "null" {
		if err := json.Unmarshal(s.Metadata, &metadata); err != nil {
			return nil, err
		}
	}

	var readBy map[uuid.UUID]time.Time // Note: time.Time in JSON needs specific handling if format differs
	if len(s.ReadBy) > 0 && string(s.ReadBy) != "null" {
		if err := json.Unmarshal(s.ReadBy, &readBy); err != nil {
			return nil, err
		}
	}

	var aiContext *AIMessageContext
	if len(s.AIContext) > 0 && string(s.AIContext) != "null" {
		if err := json.Unmarshal(s.AIContext, &aiContext); err != nil {
			return nil, err
		}
	}

	return &Message{
		ID:             s.ID,
		ConversationID: s.ConversationID,
		SenderID:       s.SenderID,
		SenderType:     ParticipantType(s.SenderType),
		Type:           MessageType(s.MessageType),
		Content:        s.Content,
		CreatedAt:      s.CreatedAt,
		UpdatedAt:      s.UpdatedAt,
		Metadata:       metadata,
		ReadBy:         readBy,
		AIContext:      aiContext,
	}, nil
}

func MapMessageToSchema(d *Message) (*schema.Message, error) {
	if d == nil {
		return nil, nil
	}

	var metaJSON []byte
	if d.Metadata != nil {
		metaJSON, _ = json.Marshal(d.Metadata)
	}

	var readByJSON []byte
	if d.ReadBy != nil {
		readByJSON, _ = json.Marshal(d.ReadBy)
	}

	var aiJSON []byte
	if d.AIContext != nil {
		aiJSON, _ = json.Marshal(d.AIContext)
	}

	return &schema.Message{
		ID:             d.ID,
		ConversationID: d.ConversationID,
		SenderID:       d.SenderID,
		SenderType:     string(d.SenderType),
		MessageType:    string(d.Type),
		Content:        d.Content,
		CreatedAt:      d.CreatedAt,
		UpdatedAt:      d.UpdatedAt,
		Metadata:       datatypes.JSON(metaJSON),
		ReadBy:         datatypes.JSON(readByJSON),
		AIContext:      datatypes.JSON(aiJSON),
	}, nil
}

// --- Participant Mappers ---

func MapParticipantToDomain(s *schema.Participant) (*Participant, error) {
	if s == nil {
		return nil, nil
	}

	return &Participant{
		ID:             s.ID,
		ConversationID: s.ConversationID,
		UserID:         s.UserID,
		Type:           ParticipantType(s.Type),
		JoinedAt:       s.JoinedAt,
		LeftAt:         s.LeftAt,
		LastReadAt:     s.LastReadAt,
		IsMuted:        s.IsMuted,
		IsVisible:      s.IsVisible,
	}, nil
}

func MapParticipantToSchema(d *Participant) (*schema.Participant, error) {
	if d == nil {
		return nil, nil
	}

	return &schema.Participant{
		ID:             d.ID,
		ConversationID: d.ConversationID,
		UserID:         d.UserID,
		Type:           string(d.Type),
		JoinedAt:       d.JoinedAt,
		LeftAt:         d.LeftAt,
		LastReadAt:     d.LastReadAt,
		IsMuted:        d.IsMuted,
		IsVisible:      d.IsVisible,
	}, nil
}
