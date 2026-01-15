package service

import (
	"context"
	"fmt"

	"hauslet/internal/modules/messaging/domain"
	"hauslet/internal/platform/events"

	"github.com/google/uuid"
)

// SetTypingIndicator broadcasts a typing indicator event to all participants.
// This is ephemeral and not persisted to the database.
func (s *messagingServiceImpl) SetTypingIndicator(ctx context.Context, conversationID, userID uuid.UUID, isTyping bool) error {
	if conversationID == uuid.Nil || userID == uuid.Nil {
		return fmt.Errorf("conversation_id and user_id are required")
	}

	// Verify user has access to this conversation
	if _, err := s.GetConversation(ctx, conversationID, userID); err != nil {
		return err
	}

	// Create typing indicator
	indicator := domain.NewTypingIndicator(conversationID, userID, isTyping)

	// Publish typing indicator event
	if s.eventPublisher != nil {
		actorID := userID.String()
		_ = s.eventPublisher.Publish(
			ctx,
			events.ChannelMessages,
			events.EventTypingIndicator,
			conversationID.String(),
			indicator,
			&events.PublishOptions{
				ActorID:      &actorID,
				FailSilently: true,
				Metadata: map[string]string{
					"conversation_id": conversationID.String(),
					"user_id":         userID.String(),
				},
			},
		)
	}

	return nil
}
