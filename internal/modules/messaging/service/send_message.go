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

	// 7. Return Result
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

// Helpers retained from your original code
func copyMetadata(src map[string]any) map[string]any {
	if len(src) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

func marshalJSON(payload map[string]any) ([]byte, error) {
	if len(payload) == 0 {
		return nil, nil
	}
	return json.Marshal(payload)
}

func sanitizeAttachments(inputs []MessageAttachmentInput) ([]map[string]any, error) {
	out := make([]map[string]any, 0, len(inputs))
	for _, att := range inputs {
		if strings.TrimSpace(att.URL) == "" {
			return nil, fmt.Errorf("attachment url required")
		}
		entry := map[string]any{
			"url":      att.URL,
			"filename": att.Filename,
			"type":     att.Type,
		}
		if att.Metadata != nil {
			entry["meta"] = att.Metadata
		}
		if att.ID != nil {
			entry["id"] = att.ID.String()
		}
		out = append(out, entry)
	}
	return out, nil
}
