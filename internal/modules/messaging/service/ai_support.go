package service

import (
	"context"
	"encoding/json"
	"time"

	"hauslet/internal/modules/messaging/domain"
	"hauslet/internal/modules/messaging/repository/schema"
	"hauslet/internal/platform/events"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// AISupportService defines the minimal interface required by messaging when the AI agent is involved.
type AISupportService interface {
	ProcessUserMessage(ctx context.Context, conversation *domain.Conversation, message *domain.Message) (*domain.Message, error)
	ShouldEscalateToHuman(ctx context.Context, conversation *domain.Conversation, aiResponse *domain.Message) (bool, string, error)
	CreateSession(ctx context.Context, userID uuid.UUID) (string, error)
	CloseSession(ctx context.Context, sessionID string) error
}

// tryTriggerAI attempts to generate an AI response in the background.
// It creates a detached context to ensure the process survives the HTTP request lifecycle.
func (s *messagingServiceImpl) tryTriggerAI(conv *domain.Conversation, userMsg *domain.Message) {
	if s.aiSupport == nil {
		return
	}

	// Create a detached context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 1. Process with AI Service
	responseMsg, err := s.aiSupport.ProcessUserMessage(ctx, conv, userMsg)
	if err != nil {
		s.log.Error("ai_agent_failed", "conversation_id", conv.ID, "error", err)
		return
	}
	if responseMsg == nil {
		// AI decided not to respond (maybe low confidence or handled silently)
		return
	}

	// 2. Save AI Response
	// Note: We use the logic similar to SendMessage but simplified for internal use
	aiSchemaMsg := &schema.Message{
		ID:             responseMsg.ID,
		ConversationID: conv.ID,
		SenderID:       responseMsg.SenderID,
		SenderType:     string(domain.ParticipantTypeAIAgent),
		MessageType:    string(domain.MessageTypeText),
		Content:        responseMsg.Content,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		Metadata:       datatypes.JSON([]byte("{}")),
	}

	if responseMsg.AIContext != nil {
		if b, err := json.Marshal(responseMsg.AIContext); err == nil {
			aiSchemaMsg.AIContext = datatypes.JSON(b)
		}
	}

	// 3. Persist
	if err := s.messageRepo.Create(ctx, aiSchemaMsg); err != nil {
		s.log.Error("failed to save ai message", "error", err)
		return
	}

	// 4. Update Conversation
	_ = s.convRepo.UpdateLastMessageTimestamp(ctx, conv.ID, aiSchemaMsg.CreatedAt)

	// 5. Publish Event
	if s.eventPublisher != nil {
		actorID := responseMsg.SenderID.String()
		_ = s.eventPublisher.Publish(ctx, events.ChannelMessages, events.EventMessageSent, responseMsg.ID.String(), responseMsg, &events.PublishOptions{
			ActorID: &actorID,
			Metadata: map[string]string{
				"conversation_id": conv.ID.String(),
			},
		})
	}
}
