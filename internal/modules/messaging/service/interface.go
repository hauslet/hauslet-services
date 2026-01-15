package service

import (
	"context"
	"time"

	"hauslet/internal/modules/messaging/domain"

	"github.com/google/uuid"
)

type MessagingService interface {
	// --- Conversation Factory Methods ---

	// GetOrCreateInquiryConversation initiates (or retrieves) a chat for a Lead (Sale/Rent/Shortlet Inquiry).
	// It automatically resolves the Lead's prospect and owner/agent using LeadHooks.
	GetOrCreateInquiryConversation(ctx context.Context, leadID uuid.UUID, requesterID uuid.UUID) (*domain.Conversation, error)

	// GetOrCreateTransactionConversation initiates (or retrieves) a chat for an active transaction.
	// ContextType must be a valid polymorphic type (e.g., "booking").
	GetOrCreateTransactionConversation(ctx context.Context, contextType domain.ConversationContextType, contextID uuid.UUID, requesterID uuid.UUID) (*domain.Conversation, error)

	// GetOrCreateSupportConversation retrieves the persistent "Hauslet Support" chat for a user.
	// If it doesn't exist, it creates one with the AI Agent as the initial responder.
	GetOrCreateSupportConversation(ctx context.Context, userID uuid.UUID) (*domain.Conversation, error)

	// --- Attachment Operations ---
	// GenerateUploadURL returns a signed URL (R2) where the client can PUT the attachment.
	GenerateUploadURL(ctx context.Context, input AttachmentUploadRequest) (*AttachmentUploadResult, error)

	// --- Core Messaging Operations ---

	// SendMessage processes a new message.
	// It handles: Authorization, Persistence, Event Publishing, and auto-triggering AI for support chats.
	SendMessage(ctx context.Context, input SendMessageInput) (*SendMessageResult, error)

	// GetConversation retrieves metadata and verifies the user has access.
	GetConversation(ctx context.Context, conversationID, userID uuid.UUID) (*domain.Conversation, error)

	// GetMessages fetches paginated message history for a conversation.
	// It verifies access before returning data.
	GetMessages(ctx context.Context, input GetMessagesInput) ([]*domain.Message, error)

	// MarkAsRead updates the unread count for the specific user in the conversation.
	MarkAsRead(ctx context.Context, conversationID, userID uuid.UUID) error

	// ListConversations returns the user's inbox, sorted by latest activity.
	ListConversations(ctx context.Context, input ListConversationsInput) ([]*domain.Conversation, error)

	// --- Support & Admin Operations ---

	// RequestSupport escalates a conversation (usually AI-handled) to human agents.
	// Updates the SupportState and notifies the support team.
	RequestSupport(ctx context.Context, conversationID, userID uuid.UUID) (*domain.Conversation, error)

	// AssignSupportAgent (Admin only) assigns a specific human agent to a support ticket.
	AssignSupportAgent(ctx context.Context, conversationID, agentID uuid.UUID) error
}

// AttachmentUploadRequest carries metadata needed to generate the presigned link.
type AttachmentUploadRequest struct {
	ConversationID uuid.UUID
	AttachmentID   uuid.UUID
	ContentType    string
	Filename       string
	SizeBytes      int64
	ExpiresIn      time.Duration // Optional: leave zero for service default (e.g., 15m)
}

// AttachmentUploadResult returns the URL and storage key the client should POST the file to.
type AttachmentUploadResult struct {
	URL string
	Key string
}

// SendMessageInput aggregates everything the service needs to construct a message.
type SendMessageInput struct {
	ConversationID uuid.UUID

	// Sender metadata
	SenderID   uuid.UUID
	SenderType domain.ParticipantType

	// Message payload
	Type    domain.MessageType
	Content string

	// Optional metadata helpers
	ClientID    *string        // Idempotency token supplied by caller
	InReplyTo   *uuid.UUID     // References a parent message
	Metadata    map[string]any // Arbitrary metadata (delivery hints, source, etc.)
	Attachments []MessageAttachmentInput

	// AI hints (support conversations only)
	AIContext *domain.AIMessageContext
}

// SendMessageResult captures the message and optional conversation state produced by SendMessage.
type SendMessageResult struct {
	Message      *domain.Message
	Conversation *domain.Conversation
}

// Pagination represents basic limit/offset parameters shared across messaging queries.
type Pagination struct {
	Limit  int
	Offset int
}

// Normalize enforces sane boundaries on pagination values.
func (p Pagination) Normalize() Pagination {
	if p.Limit <= 0 || p.Limit > 100 {
		p.Limit = 20
	}
	if p.Offset < 0 {
		p.Offset = 0
	}
	return p
}

// ListConversationsInput centralizes pagination for inbox listing.
type ListConversationsInput struct {
	UserID uuid.UUID
	Page   Pagination
}

// GetMessagesInput centralizes pagination and access info for message retrieval.
type GetMessagesInput struct {
	ConversationID uuid.UUID
	UserID         uuid.UUID
	Page           Pagination
}

// MessageAttachmentInput expresses an attachment to be saved alongside a message.
type MessageAttachmentInput struct {
	ID       *uuid.UUID        // Optional ID for explicit lookups
	URL      string            // Publicly accessible location
	Type     string            // e.g., "image", "file", "system", matching MessageType if needed
	Filename string            // Original filename for downloads
	Metadata map[string]string // Attachment-specific metadata (mime type, size, sha)
}
