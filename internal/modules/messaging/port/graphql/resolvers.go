package graphql

import (
	"context"
	"encoding/json"
	"log/slog"

	"hauslet/internal/modules/messaging/domain"
	"hauslet/internal/modules/messaging/service"
	"hauslet/internal/platform/events"
	"hauslet/internal/transport/graph/model"
	"hauslet/internal/transport/graph/viewer"

	"github.com/google/uuid"
)

// Resolver provides the GraphQL handlers for messaging operations.
type Resolver struct {
	messagingSvc    service.MessagingService
	eventSubscriber *events.Subscriber
	log             *slog.Logger
}

// NewResolver constructs a messaging GraphQL resolver.
func NewResolver(messagingSvc service.MessagingService, eventSubscriber *events.Subscriber, log *slog.Logger) *Resolver {
	return &Resolver{
		messagingSvc:    messagingSvc,
		eventSubscriber: eventSubscriber,
		log:             log,
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

// HausletSupport returns or creates the persistent Hauslet Support conversation for the user.
func (r *Resolver) HausletSupport(ctx context.Context) (*domain.Conversation, error) {
	userID, err := viewer.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	conversation, err := r.messagingSvc.GetOrCreateSupportConversation(ctx, userID)
	if err != nil {
		r.log.Error("failed to get/create support conversation", "user_id", userID, "error", err)
		return nil, err
	}

	return conversation, nil
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

// ============================================================================
// Subscription Resolvers
// ============================================================================

// MessageReceived subscribes to new messages in a specific conversation.
// The user must be a participant in the conversation to subscribe.
func (r *Resolver) MessageReceived(ctx context.Context, conversationID uuid.UUID) (<-chan *domain.Message, error) {
	userID, err := viewer.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	// Verify the user has access to this conversation before subscribing
	_, err = r.messagingSvc.GetConversation(ctx, conversationID, userID)
	if err != nil {
		r.log.Error("subscription auth failed", "conversation_id", conversationID, "user_id", userID, "error", err)
		return nil, err
	}

	r.messagingSvc.MarkParticipantActive(conversationID, userID)

	// Subscribe to message events with filters
	eventCh, err := r.eventSubscriber.SubscribeWithFilter(
		ctx,
		events.ChannelMessages,
		events.FilterByType(events.EventMessageSent),
		events.FilterByMetadata("conversation_id", conversationID.String()),
	)
	if err != nil {
		r.log.Error("failed to subscribe to message events", "conversation_id", conversationID, "error", err)
		return nil, err
	}

	// Create output channel
	outCh := make(chan *domain.Message, 10)

	// Process events in background
	go func() {
		defer func() {
			r.messagingSvc.MarkParticipantInactive(conversationID, userID)
			close(outCh)
		}()

		for {
			select {
			case <-ctx.Done():
				return

			case event, ok := <-eventCh:
				if !ok {
					return
				}

				// Parse message from event payload
				var msg domain.Message
				if err := json.Unmarshal(event.Payload, &msg); err != nil {
					r.log.Error("failed to unmarshal message event", "error", err)
					continue
				}

				// Send to subscriber
				select {
				case outCh <- &msg:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return outCh, nil
}

// ConversationUpdated subscribes to updates for a specific conversation.
// The user must be a participant in the conversation to subscribe.
func (r *Resolver) ConversationUpdated(ctx context.Context, conversationID uuid.UUID) (<-chan *domain.Conversation, error) {
	userID, err := viewer.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	// Verify the user has access to this conversation before subscribing
	_, err = r.messagingSvc.GetConversation(ctx, conversationID, userID)
	if err != nil {
		r.log.Error("subscription auth failed", "conversation_id", conversationID, "user_id", userID, "error", err)
		return nil, err
	}

	// Subscribe to conversation update events
	eventCh, err := r.eventSubscriber.SubscribeWithFilter(
		ctx,
		events.ChannelMessages,
		events.FilterByType(events.EventConversationUpdated, events.EventMessageRead),
		events.FilterByMetadata("conversation_id", conversationID.String()),
	)
	if err != nil {
		r.log.Error("failed to subscribe to conversation events", "conversation_id", conversationID, "error", err)
		return nil, err
	}

	// Create output channel
	outCh := make(chan *domain.Conversation, 10)

	// Process events in background
	go func() {
		defer close(outCh)

		for {
			select {
			case <-ctx.Done():
				return

			case event, ok := <-eventCh:
				if !ok {
					return
				}

				// Parse conversation from event payload
				var conv domain.Conversation
				if err := json.Unmarshal(event.Payload, &conv); err != nil {
					r.log.Error("failed to unmarshal conversation event", "error", err)
					continue
				}

				// Send to subscriber
				select {
				case outCh <- &conv:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return outCh, nil
}

// MyConversationsUpdated subscribes to updates across all of the user's conversations.
// Useful for inbox-level notifications.
func (r *Resolver) MyConversationsUpdated(ctx context.Context) (<-chan *domain.Conversation, error) {
	userID, err := viewer.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	// Subscribe to all conversation events (we'll filter by user participation)
	eventCh, err := r.eventSubscriber.SubscribeWithFilter(
		ctx,
		events.ChannelMessages,
		events.FilterByType(events.EventMessageSent, events.EventConversationUpdated, events.EventConversationCreated),
	)
	if err != nil {
		r.log.Error("failed to subscribe to conversation events", "user_id", userID, "error", err)
		return nil, err
	}

	// Create output channel
	outCh := make(chan *domain.Conversation, 10)

	// Process events in background
	go func() {
		defer close(outCh)

		for {
			select {
			case <-ctx.Done():
				return

			case event, ok := <-eventCh:
				if !ok {
					return
				}

				// Extract conversation ID from metadata
				convIDStr, ok := event.Metadata["conversation_id"]
				if !ok {
					continue
				}

				convID, err := uuid.Parse(convIDStr)
				if err != nil {
					continue
				}

				// Verify the user is a participant in this conversation
				// This ensures we only deliver events for conversations the user can access
				conv, err := r.messagingSvc.GetConversation(ctx, convID, userID)
				if err != nil {
					// User is not a participant or conversation doesn't exist
					continue
				}

				// Send to subscriber
				select {
				case outCh <- conv:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return outCh, nil
}

// ============================================================================
// Field Resolvers
// ============================================================================

// MessageReadBy resolves the readBy field on Message, converting UUID keys to strings.
func (r *Resolver) MessageReadBy(ctx context.Context, message *domain.Message) (map[string]any, error) {
	if message == nil || len(message.ReadBy) == 0 {
		return nil, nil
	}

	result := make(map[string]any, len(message.ReadBy))
	for userID, readAt := range message.ReadBy {
		result[userID.String()] = readAt
	}

	return result, nil
}

// ============================================================================
// Typing Indicator Resolvers
// ============================================================================

// SetTypingIndicator broadcasts a typing indicator to all participants.
func (r *Resolver) SetTypingIndicator(ctx context.Context, conversationID uuid.UUID, isTyping bool) (bool, error) {
	userID, err := viewer.GetUserIDFromContext(ctx)
	if err != nil {
		return false, err
	}

	if err := r.messagingSvc.SetTypingIndicator(ctx, conversationID, userID, isTyping); err != nil {
		r.log.Error("failed to set typing indicator", "conversation_id", conversationID, "user_id", userID, "error", err)
		return false, err
	}

	return true, nil
}

// TypingIndicator subscribes to typing indicators in a specific conversation.
func (r *Resolver) TypingIndicator(ctx context.Context, conversationID uuid.UUID) (<-chan *domain.TypingIndicator, error) {
	userID, err := viewer.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	// Verify the user has access to this conversation before subscribing
	_, err = r.messagingSvc.GetConversation(ctx, conversationID, userID)
	if err != nil {
		r.log.Error("subscription auth failed", "conversation_id", conversationID, "user_id", userID, "error", err)
		return nil, err
	}

	// Subscribe to typing indicator events
	eventCh, err := r.eventSubscriber.SubscribeWithFilter(
		ctx,
		events.ChannelMessages,
		events.FilterByType(events.EventTypingIndicator),
		events.FilterByMetadata("conversation_id", conversationID.String()),
	)
	if err != nil {
		r.log.Error("failed to subscribe to typing events", "conversation_id", conversationID, "error", err)
		return nil, err
	}

	// Create output channel
	outCh := make(chan *domain.TypingIndicator, 10)

	// Process events in background
	go func() {
		defer close(outCh)

		for {
			select {
			case <-ctx.Done():
				return

			case event, ok := <-eventCh:
				if !ok {
					return
				}

				// Parse typing indicator from event payload
				var indicator domain.TypingIndicator
				if err := json.Unmarshal(event.Payload, &indicator); err != nil {
					r.log.Error("failed to unmarshal typing indicator event", "error", err)
					continue
				}

				// Don't send the user's own typing events back to them
				if indicator.UserID == userID {
					continue
				}

				// Send to subscriber
				select {
				case outCh <- &indicator:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return outCh, nil
}
