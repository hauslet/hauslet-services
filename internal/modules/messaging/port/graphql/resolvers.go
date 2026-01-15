package graphql

import (
	"context"
	"log/slog"

	"hauslet/internal/modules/messaging/domain"
	"hauslet/internal/modules/messaging/service"
	"hauslet/internal/transport/graph/model"
	"hauslet/internal/transport/graph/viewer"

	"github.com/google/uuid"
)

// Resolver provides the GraphQL handlers for messaging operations.
type Resolver struct {
	messagingSvc service.MessagingService
	log          *slog.Logger
}

// NewResolver constructs a messaging GraphQL resolver.
func NewResolver(messagingSvc service.MessagingService, log *slog.Logger) *Resolver {
	return &Resolver{
		messagingSvc: messagingSvc,
		log:          log,
	}
}

// Conversation fetches a single conversation for the authenticated user.
func (r *Resolver) Conversation(ctx context.Context, id uuid.UUID) (*domain.Conversation, error) {
	userID, err := viewer.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	conversation, err := r.messagingSvc.GetConversation(ctx, id, userID)
	if err != nil {
		r.log.Error("failed to fetch conversation", "conversation_id", id, "error", err)
		return nil, err
	}

	return conversation, nil
}

// MyConversations returns the requesting user's inbox.
func (r *Resolver) MyConversations(ctx context.Context, limit *int, offset *int) ([]*domain.Conversation, error) {
	userID, err := viewer.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	page := r.buildPagination(limit, offset)
	conversations, err := r.messagingSvc.ListConversations(ctx, service.ListConversationsInput{
		UserID: userID,
		Page:   page,
	})
	if err != nil {
		r.log.Error("failed to list conversations", "user_id", userID, "error", err)
		return nil, err
	}

	return conversations, nil
}

// StartInquiryConversation returns or creates the inbox for a lead inquiry.
func (r *Resolver) StartInquiryConversation(ctx context.Context, leadID uuid.UUID) (*domain.Conversation, error) {
	userID, err := viewer.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	conversation, err := r.messagingSvc.GetOrCreateInquiryConversation(ctx, leadID, userID)
	if err != nil {
		r.log.Error("failed to start inquiry conversation", "lead_id", leadID, "error", err)
		return nil, err
	}

	return conversation, nil
}

// StartTransactionConversation returns or creates the inbox for a booking transaction.
func (r *Resolver) StartTransactionConversation(ctx context.Context, contextType domain.ConversationContextType, contextID uuid.UUID) (*domain.Conversation, error) {
	userID, err := viewer.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	conversation, err := r.messagingSvc.GetOrCreateTransactionConversation(ctx, contextType, contextID, userID)
	if err != nil {
		r.log.Error("failed to start transaction conversation", "context_type", contextType, "context_id", contextID, "error", err)
		return nil, err
	}

	return conversation, nil
}

// SendMessage proxies the mutation to the messaging service for the authenticated user.
// SendMessage proxies the mutation to the messaging service for the authenticated user.
func (r *Resolver) SendMessage(
	ctx context.Context,
	input model.SendMessageInput,
) (*domain.Message, error) {
	userID, err := viewer.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	attachments := make([]service.MessageAttachmentInput, 0, len(input.Attachments))

	for _, att := range input.Attachments {
		if att == nil {
			continue
		}

		metadata := make(map[string]string)
		for k, v := range att.Metadata {
			if str, ok := v.(string); ok {
				metadata[k] = str
			}
		}

		a := service.MessageAttachmentInput{
			ID:       att.ID,
			URL:      att.URL,
			Metadata: metadata,
		}

		if att.Type != nil {
			a.Type = *att.Type
		}
		if att.Filename != nil {
			a.Filename = *att.Filename
		}

		attachments = append(attachments, a)
	}

	serviceInput := service.SendMessageInput{
		ConversationID: input.ConversationID,
		SenderID:       userID,
		Type:           input.Type,
		Content:        input.Content,
		Metadata:       input.Metadata,
		Attachments:    attachments,
	}

	result, err := r.messagingSvc.SendMessage(ctx, serviceInput)
	if err != nil {
		r.log.Error(
			"failed to send message",
			"user_id", userID,
			"conversation_id", input.ConversationID,
			"error", err,
		)
		return nil, err
	}

	return result.Message, nil
}

// MarkConversationAsRead acknowledges the conversation for the viewer.
func (r *Resolver) MarkConversationAsRead(ctx context.Context, conversationID uuid.UUID) (bool, error) {
	userID, err := viewer.GetUserIDFromContext(ctx)
	if err != nil {
		return false, err
	}

	if err := r.messagingSvc.MarkAsRead(ctx, conversationID, userID); err != nil {
		r.log.Error("failed to mark conversation as read", "conversation_id", conversationID, "user_id", userID, "error", err)
		return false, err
	}

	return true, nil
}

// ConversationMessages resolves paginated messages for the conversation.
func (r *Resolver) ConversationMessages(ctx context.Context, conversation *domain.Conversation, limit *int, offset *int) ([]*domain.Message, error) {
	if conversation == nil {
		return nil, nil
	}

	userID, err := viewer.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	page := r.buildPagination(limit, offset)
	messages, err := r.messagingSvc.GetMessages(ctx, service.GetMessagesInput{
		ConversationID: conversation.ID,
		UserID:         userID,
		Page:           page,
	})
	if err != nil {
		r.log.Error("failed to fetch messages", "conversation_id", conversation.ID, "error", err)
		return nil, err
	}

	return messages, nil
}

// ConversationUnreadCounts maps UUID keys to string-based map for GraphQL.
func (r *Resolver) ConversationUnreadCounts(ctx context.Context, conversation *domain.Conversation) (map[string]any, error) {
	if conversation == nil || len(conversation.UnreadCounts) == 0 {
		return nil, nil
	}

	result := make(map[string]any, len(conversation.UnreadCounts))
	for userID, count := range conversation.UnreadCounts {
		result[userID.String()] = count
	}

	return result, nil
}

func (r *Resolver) buildPagination(limit *int, offset *int) service.Pagination {
	page := service.Pagination{}
	if limit != nil {
		page.Limit = *limit
	}
	if offset != nil {
		page.Offset = *offset
	}
	return page.Normalize()
}
