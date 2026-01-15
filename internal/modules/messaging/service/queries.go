package service

import (
	"context"
	"fmt"

	"hauslet/internal/modules/messaging/domain"

	"github.com/google/uuid"
)

func (s *messagingServiceImpl) GetConversation(ctx context.Context, conversationID, userID uuid.UUID) (*domain.Conversation, error) {
	if conversationID == uuid.Nil || userID == uuid.Nil {
		return nil, fmt.Errorf("conversation_id and user_id are required")
	}

	conversation, err := s.convRepo.GetByID(ctx, conversationID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch conversation: %w", err)
	}

	return s.ensureConversationAccess(conversation, userID)
}

func (s *messagingServiceImpl) GetMessages(ctx context.Context, input GetMessagesInput) ([]*domain.Message, error) {
	if input.ConversationID == uuid.Nil || input.UserID == uuid.Nil {
		return nil, fmt.Errorf("conversation_id and user_id are required")
	}

	if _, err := s.GetConversation(ctx, input.ConversationID, input.UserID); err != nil {
		return nil, err
	}

	page := input.Page.Normalize()
	schemaMessages, err := s.messageRepo.ListByConversation(ctx, input.ConversationID, page.Limit, page.Offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list messages: %w", err)
	}

	messages := make([]*domain.Message, 0, len(schemaMessages))
	for _, msg := range schemaMessages {
		mapped, err := domain.MapMessageToDomain(msg)
		if err != nil {
			return nil, fmt.Errorf("failed to map message: %w", err)
		}
		messages = append(messages, mapped)
	}

	return messages, nil
}

func (s *messagingServiceImpl) ListConversations(ctx context.Context, input ListConversationsInput) ([]*domain.Conversation, error) {
	if input.UserID == uuid.Nil {
		return nil, fmt.Errorf("user_id is required")
	}

	page := input.Page.Normalize()
	schemaConversations, err := s.convRepo.ListForUser(ctx, input.UserID, page.Limit, page.Offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list conversations: %w", err)
	}

	conversations := make([]*domain.Conversation, 0, len(schemaConversations))
	for _, conv := range schemaConversations {
		mapped, err := domain.MapConversationToDomain(conv)
		if err != nil {
			return nil, fmt.Errorf("failed to map conversation: %w", err)
		}
		conversations = append(conversations, mapped)
	}

	return conversations, nil
}

func (s *messagingServiceImpl) MarkAsRead(ctx context.Context, conversationID, userID uuid.UUID) error {
	if conversationID == uuid.Nil || userID == uuid.Nil {
		return fmt.Errorf("conversation_id and user_id are required")
	}

	if _, err := s.GetConversation(ctx, conversationID, userID); err != nil {
		return err
	}

	if err := s.convRepo.MarkAsRead(ctx, conversationID, userID); err != nil {
		return fmt.Errorf("failed to mark conversation as read: %w", err)
	}

	return nil
}
