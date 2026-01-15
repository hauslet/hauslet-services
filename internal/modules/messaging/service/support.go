package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"hauslet/internal/modules/messaging/domain"
	"hauslet/internal/modules/messaging/repository/schema"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func (s *messagingServiceImpl) RequestSupport(ctx context.Context, conversationID, userID uuid.UUID) (*domain.Conversation, error) {
	if conversationID == uuid.Nil || userID == uuid.Nil {
		return nil, fmt.Errorf("conversation_id and user_id are required")
	}

	conversation, err := s.convRepo.GetByID(ctx, conversationID)
	if err != nil {
		return nil, fmt.Errorf("failed to load conversation: %w", err)
	}

	domainConv, err := s.ensureConversationAccess(conversation, userID)
	if err != nil {
		return nil, err
	}

	if !domainConv.IsSupport() {
		return nil, fmt.Errorf("request support is only valid for support conversations")
	}

	supportState := domainConv.SupportState
	if supportState == nil {
		supportState = domain.NewSupportState()
	}
	supportState.Status = domain.SupportStatusEscalated
	supportState.AssignedAgentID = nil

	if err := s.persistSupportState(ctx, conversation, supportState); err != nil {
		return nil, err
	}

	return s.reloadConversation(ctx, conversationID)
}

func (s *messagingServiceImpl) AssignSupportAgent(ctx context.Context, conversationID, agentID uuid.UUID) error {
	if conversationID == uuid.Nil || agentID == uuid.Nil {
		return fmt.Errorf("conversation_id and agent_id are required")
	}

	conversation, err := s.convRepo.GetByID(ctx, conversationID)
	if err != nil {
		return fmt.Errorf("failed to load conversation: %w", err)
	}

	domainConv, err := domain.MapConversationToDomain(conversation)
	if err != nil {
		return fmt.Errorf("failed to map conversation: %w", err)
	}
	if domainConv == nil {
		return domain.ErrConversationNotFound
	}
	if !domainConv.IsSupport() {
		return fmt.Errorf("assigning agents is only valid for support conversations")
	}

	supportState := domainConv.SupportState
	if supportState == nil {
		supportState = domain.NewSupportState()
	}
	supportState.Status = domain.SupportStatusAgentActive
	supportState.AssignedAgentID = &agentID

	if err := s.persistSupportState(ctx, conversation, supportState); err != nil {
		return err
	}

	if err := s.ensureSupportAgentParticipant(ctx, conversationID, agentID); err != nil {
		return err
	}

	return nil
}

func (s *messagingServiceImpl) persistSupportState(ctx context.Context, conversation *schema.Conversation, state *domain.SupportState) error {
	bytes, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal support state: %w", err)
	}
	conversation.SupportState = datatypes.JSON(bytes)
	if err := s.convRepo.Update(ctx, conversation); err != nil {
		return fmt.Errorf("failed to persist support state: %w", err)
	}
	return nil
}

func (s *messagingServiceImpl) reloadConversation(ctx context.Context, conversationID uuid.UUID) (*domain.Conversation, error) {
	updated, err := s.convRepo.GetByID(ctx, conversationID)
	if err != nil {
		return nil, fmt.Errorf("failed to reload conversation: %w", err)
	}
	mapped, err := domain.MapConversationToDomain(updated)
	if err != nil {
		return nil, fmt.Errorf("failed to map conversation: %w", err)
	}
	if mapped == nil {
		return nil, domain.ErrConversationNotFound
	}
	return mapped, nil
}

func (s *messagingServiceImpl) ensureSupportAgentParticipant(ctx context.Context, conversationID, agentID uuid.UUID) error {
	if s.participantRepo == nil {
		return fmt.Errorf("participant repository not configured")
	}

	_, err := s.participantRepo.Get(ctx, conversationID, agentID)
	if err == nil {
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("failed to check support agent participant: %w", err)
	}

	participant := &schema.Participant{
		ID:             uuid.New(),
		ConversationID: conversationID,
		UserID:         agentID,
		Type:           string(domain.ParticipantTypeSupportAgent),
		JoinedAt:       time.Now(),
		IsVisible:      true,
	}

	if err := s.participantRepo.Add(ctx, participant); err != nil {
		return fmt.Errorf("failed to add support agent participant: %w", err)
	}
	return nil
}
