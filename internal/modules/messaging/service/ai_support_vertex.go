package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"hauslet/config"
	"hauslet/internal/modules/messaging/domain"
	"hauslet/internal/modules/messaging/port/vertexai"
	"hauslet/internal/modules/messaging/repository"

	"log/slog"

	"github.com/google/uuid"
)

// AISupportService defines the minimal interface required by messaging when the AI agent is involved.
type AISupportService interface {
	ProcessUserMessage(ctx context.Context, conversation *domain.Conversation, message *domain.Message) (*domain.Message, error)
	ShouldEscalateToHuman(ctx context.Context, conversation *domain.Conversation, aiResponse *domain.Message) (bool, string, error)
	CreateSession(ctx context.Context, userID uuid.UUID) (string, error)
	CloseSession(ctx context.Context, sessionID string) error
}

type vertexAISupportService struct {
	client      *vertexai.VertexAIClient
	messageRepo repository.MessageRepository
	config      config.MessagingConfig
	logger      *slog.Logger
}

// NewVertexAISupportService builds the default AI support service powered by Vertex AI.
func NewVertexAISupportService(client *vertexai.VertexAIClient, messageRepo repository.MessageRepository, cfg config.MessagingConfig, logger *slog.Logger) AISupportService {
	if client == nil || messageRepo == nil {
		return nil
	}
	if cfg.AIContextMessages <= 0 {
		cfg.AIContextMessages = 10
	}
	if cfg.AIConfidenceThreshold <= 0 {
		cfg.AIConfidenceThreshold = 0.6
	}
	return &vertexAISupportService{
		client:      client,
		messageRepo: messageRepo,
		config:      cfg,
		logger:      logger,
	}
}

func (s *vertexAISupportService) ProcessUserMessage(ctx context.Context, conversation *domain.Conversation, message *domain.Message) (*domain.Message, error) {
	if conversation == nil || message == nil {
		return nil, fmt.Errorf("conversation and message are required for ai processing")
	}

	request := &vertexai.SendMessageRequest{
		SessionID: conversation.ID.String(),
		Query:     strings.TrimSpace(message.Content),
		History:   nil,
		UserContext: &vertexai.UserContext{
			UserID: message.SenderID.String(),
			Metadata: map[string]string{
				"participant_type": string(message.SenderType),
			},
		},
		Metadata:      s.conversationMetadata(conversation),
		MaxHistory:    s.historyLimit(),
		ServingConfig: s.config.AIServingConfig,
	}

	if history, err := s.buildHistory(ctx, conversation.ID, message.ID, request.MaxHistory); err == nil {
		request.History = history
	} else {
		return nil, fmt.Errorf("failed to build ai history: %w", err)
	}

	timeout := s.requestTimeout()
	reqCtx := ctx
	var cancel context.CancelFunc
	if timeout > 0 {
		reqCtx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	resp, err := s.client.SendMessage(reqCtx, request)
	if err != nil {
		return nil, fmt.Errorf("vertex ai conversation failed: %w", err)
	}

	aiMessage := &domain.Message{
		ID:             uuid.New(),
		ConversationID: conversation.ID,
		SenderID:       uuid.New(),
		SenderType:     domain.ParticipantTypeAIAgent,
		Type:           domain.MessageTypeText,
		Content:        resp.Text,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		AIContext: &domain.AIMessageContext{
			Intent:      resp.Intent,
			Confidence:  float64(resp.Confidence),
			TokenUsage:  0,
			IsEscalated: resp.RequiresEscalation,
		},
	}

	if len(resp.Sources) > 0 {
		aiMessage.Metadata = map[string]any{
			"ai_sources": resp.Sources,
		}
	}

	return aiMessage, nil
}

func (s *vertexAISupportService) ShouldEscalateToHuman(ctx context.Context, conversation *domain.Conversation, aiResponse *domain.Message) (bool, string, error) {
	if aiResponse == nil || aiResponse.AIContext == nil {
		return false, "", nil
	}

	threshold := s.config.AIConfidenceThreshold
	if threshold <= 0 {
		threshold = 0.6
	}

	if aiResponse.AIContext.IsEscalated {
		return true, "ai requested escalation", nil
	}
	if aiResponse.AIContext.Confidence < threshold {
		return true, fmt.Sprintf("confidence %.2f below threshold %.2f", aiResponse.AIContext.Confidence, threshold), nil
	}

	return false, "", nil
}

func (s *vertexAISupportService) CreateSession(ctx context.Context, userID uuid.UUID) (string, error) {
	if userID == uuid.Nil {
		return "", fmt.Errorf("user_id is required to create ai session")
	}
	return uuid.NewString(), nil
}

func (s *vertexAISupportService) CloseSession(ctx context.Context, sessionID string) error {
	return nil
}

func (s *vertexAISupportService) historyLimit() int {
	if s.config.AIContextMessages > 0 {
		return s.config.AIContextMessages
	}
	return 10
}

func (s *vertexAISupportService) requestTimeout() time.Duration {
	if s.config.AITimeoutSeconds > 0 {
		return time.Duration(s.config.AITimeoutSeconds) * time.Second
	}
	return 15 * time.Second
}

func (s *vertexAISupportService) buildHistory(ctx context.Context, conversationID, skip uuid.UUID, limit int) ([]vertexai.HistoryEntry, error) {
	if conversationID == uuid.Nil || limit <= 0 {
		return nil, nil
	}

	fetchLimit := limit * 2
	if fetchLimit == 0 {
		fetchLimit = limit
	}

	messages, err := s.messageRepo.ListByConversation(ctx, conversationID, fetchLimit, 0)
	if err != nil {
		return nil, err
	}

	history := make([]vertexai.HistoryEntry, 0, len(messages))
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		if msg == nil || msg.ID == skip {
			continue
		}

		domainMsg, err := domain.MapMessageToDomain(msg)
		if err != nil {
			if s.logger != nil {
				s.logger.Debug("failed to map message for ai history", "conversation_id", conversationID, "error", err)
			}
			continue
		}
		if domainMsg == nil || domainMsg.Type != domain.MessageTypeText {
			continue
		}

		history = append(history, vertexai.HistoryEntry{
			Role:      historyRoleFromParticipant(domainMsg.SenderType),
			Text:      domainMsg.Content,
			CreatedAt: domainMsg.CreatedAt,
		})
		if len(history) >= limit {
			break
		}
	}

	return history, nil
}

func historyRoleFromParticipant(typ domain.ParticipantType) vertexai.HistoryRole {
	if typ == domain.ParticipantTypeAIAgent {
		return vertexai.HistoryRoleAI
	}
	return vertexai.HistoryRoleUser
}

func (s *vertexAISupportService) conversationMetadata(conversation *domain.Conversation) map[string]string {
	if conversation == nil {
		return nil
	}

	metadata := map[string]string{
		"conversation_id":   conversation.ID.String(),
		"conversation_type": string(conversation.Type),
		"context_type":      string(conversation.ContextType),
	}
	if conversation.ContextID != uuid.Nil {
		metadata["context_id"] = conversation.ContextID.String()
	}
	if conversation.SupportState != nil {
		if conversation.SupportState.Status != "" {
			metadata["support_status"] = string(conversation.SupportState.Status)
		}
		if conversation.SupportState.AISessionID != "" {
			metadata["ai_session_id"] = conversation.SupportState.AISessionID
		}
	}

	for key, value := range conversation.Metadata {
		if key == "" || value == nil {
			continue
		}
		metadata["meta_"+sanitizeMetaKey(key)] = fmt.Sprint(value)
	}

	if len(metadata) == 0 {
		return nil
	}
	return metadata
}
