package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"hauslet/internal/modules/messaging/domain"
	"hauslet/internal/modules/messaging/repository/schema"
	"hauslet/internal/platform/events"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// AIEscalationMarker is the exact string the AI agent returns when it needs to escalate to a human.
// When detected, the message is not saved or sent to the guest.
const AIEscalationMarker = "###ESCALATE_TO_HUMAN###"

func (s *messagingServiceImpl) SendMessage(ctx context.Context, input SendMessageInput) (*SendMessageResult, error) {
	if input.ConversationID == uuid.Nil || input.SenderID == uuid.Nil {
		return nil, fmt.Errorf("conversation_id and sender_id are required")
	}
	if input.Content == "" {
		return nil, domain.ErrMessageEmpty
	}
	if input.Type == "" {
		return nil, fmt.Errorf("message type is required")
	}

	// 1. Fetch and Authorize
	conversation, err := s.convRepo.GetByID(ctx, input.ConversationID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch conversation: %w", err)
	}

	domainConv, err := s.ensureConversationAccess(conversation, input.SenderID)
	if err != nil {
		return nil, err
	}

	// 2. Resolve Sender Type
	senderType := input.SenderType
	if senderType == "" {
		if participant := domainConv.GetParticipant(input.SenderID); participant != nil {
			senderType = participant.Type
		}
	}
	if senderType == "" {
		senderType = domain.ParticipantTypeUser
	}

	// 3. Prepare Data
	metadata := copyMetadata(input.Metadata)
	if len(input.Attachments) > 0 {
		attachments, err := sanitizeAttachments(input.Attachments)
		if err != nil {
			return nil, fmt.Errorf("invalid attachments: %w", err)
		}
		if len(attachments) > 0 {
			metadata["attachments"] = attachments
		}
	}
	metaJSON, err := marshalJSON(metadata)
	if err != nil {
		return nil, fmt.Errorf("invalid metadata: %w", err)
	}

	var aiJSON datatypes.JSON
	if input.AIContext != nil {
		bytes, err := json.Marshal(input.AIContext)
		if err != nil {
			return nil, fmt.Errorf("invalid ai context: %w", err)
		}
		aiJSON = datatypes.JSON(bytes)
	}

	now := time.Now()
	message := &schema.Message{
		ID:             uuid.New(),
		ConversationID: conversation.ID,
		SenderID:       input.SenderID,
		SenderType:     string(senderType),
		MessageType:    string(input.Type),
		Content:        input.Content,
		CreatedAt:      now,
		UpdatedAt:      now,
		Metadata:       datatypes.JSON(metaJSON),
		AIContext:      aiJSON,
	}

	// 4. Execute Transaction (Atomic Save + Update Timestamp + Increment Unread)
	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.messageRepo.WithTx(tx).Create(ctx, message); err != nil {
			return fmt.Errorf("failed to create message: %w", err)
		}

		if err := s.convRepo.WithTx(tx).UpdateLastMessageTimestamp(ctx, conversation.ID, now); err != nil {
			return fmt.Errorf("failed to update timestamp: %w", err)
		}

		// Calculate recipients inside tx to ensure consistency
		var recipients []uuid.UUID
		for _, participant := range domainConv.Participants {
			if participant.UserID == input.SenderID {
				continue
			}
			// Optional: Check if participant is muted here
			recipients = append(recipients, participant.UserID)
		}

		if len(recipients) > 0 {
			if err := s.convRepo.WithTx(tx).IncrementUnreadCounts(ctx, conversation.ID, recipients); err != nil {
				return fmt.Errorf("failed to increment unread counts: %w", err)
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	// 5. Post-Transaction: Domain Mapping & Events
	domainMessage, err := domain.MapMessageToDomain(message)
	if err != nil {
		return nil, fmt.Errorf("failed to map message: %w", err)
	}

	if s.eventPublisher != nil {
		actorID := input.SenderID.String()
		// Fire and forget event
		_ = s.eventPublisher.Publish(ctx, events.ChannelMessages, events.EventMessageSent, domainMessage.ID.String(), domainMessage, &events.PublishOptions{
			ActorID:      &actorID,
			FailSilently: true,
			Metadata: map[string]string{
				"conversation_id": conversation.ID.String(),
			},
		})
	}

	// 6. Async AI Trigger
	// Only trigger if this is a support conversation and the sender is a User (not system/agent)
	if domainConv.IsSupport() && senderType == domain.ParticipantTypeUser {
		go s.tryTriggerAI(domainConv, domainMessage)
	}

	// 7. Async Email Notifications
	// Notify recipients who are not currently connected (offline notifications)
	if s.notificationSvc != nil && s.profileHooks != nil {
		go s.notifyMessageRecipients(domainConv, domainMessage, input.SenderID)
	}

	// 8. Return Result
	// Reload conversation to get fresh unread counts/timestamp
	updatedConv, err := s.convRepo.GetByID(ctx, conversation.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to reload conversation: %w", err)
	}
	domainConversation, err := domain.MapConversationToDomain(updatedConv)
	if err != nil {
		return nil, fmt.Errorf("failed to map updated conversation: %w", err)
	}

	return &SendMessageResult{
		Message:      domainMessage,
		Conversation: domainConversation,
	}, nil
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

	// 2. Check for explicit escalation marker
	// If the AI returns this marker, escalate immediately without saving or sending the message
	if strings.TrimSpace(responseMsg.Content) == AIEscalationMarker {
		s.log.Info("ai_escalation_marker_detected", "conversation_id", conv.ID)
		if _, err := s.RequestSupport(ctx, conv.ID, userMsg.SenderID); err != nil {
			s.log.Error("ai_marker_escalation_failed", "conversation_id", conv.ID, "error", err)
		}
		return
	}

	// 3. Save AI Response
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

	// 4. Persist
	if err := s.messageRepo.Create(ctx, aiSchemaMsg); err != nil {
		s.log.Error("failed to save ai message", "error", err)
		return
	}

	// 5. Update Conversation
	_ = s.convRepo.UpdateLastMessageTimestamp(ctx, conv.ID, aiSchemaMsg.CreatedAt)

	// 6. Publish Event
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

// notifyMessageRecipients sends email notifications to message recipients.
// It runs asynchronously with a detached context.
func (s *messagingServiceImpl) notifyMessageRecipients(conv *domain.Conversation, msg *domain.Message, senderID uuid.UUID) {
	if s.notificationSvc == nil || s.profileHooks == nil {
		return
	}

	// Create a detached context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get sender contact info
	senderContact, err := s.profileHooks.GetUserContact(ctx, senderID)
	if err != nil {
		s.log.Warn("failed to get sender contact for notification", "sender_id", senderID, "error", err)
	}

	// Notify all recipients
	s.notificationSvc.NotifyRecipients(ctx, conv, msg, senderContact, s.profileHooks.GetUserContact)
}
