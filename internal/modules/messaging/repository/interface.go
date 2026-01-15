package repository

import (
	"context"
	"hauslet/internal/modules/messaging/repository/schema"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ConversationRepository handles the persistence of Conversation records
type ConversationRepository interface {
	// Core Lifecycle
	Create(ctx context.Context, conversation *schema.Conversation) error
	GetByID(ctx context.Context, id uuid.UUID) (*schema.Conversation, error)
	Update(ctx context.Context, conversation *schema.Conversation) error

	// Context Lookups
	// GetByContext finds the existing chat for a specific Lead, Booking, etc.
	// contextType is a string matching the polymorphic schema field (e.g., "lead", "booking")
	GetByContext(ctx context.Context, contextType string, contextID uuid.UUID) (*schema.Conversation, error)

	// GetSupportConversation finds the persistent "Hauslet Support" chat for a user
	GetSupportConversation(ctx context.Context, userID uuid.UUID) (*schema.Conversation, error)

	// Lists
	// ListForUser returns paginated conversations for a user's inbox
	ListForUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*schema.Conversation, error)

	// ListSupportQueue returns active support chats (filtered by agent assignment if provided)
	ListSupportQueue(ctx context.Context, assignedAgentID *uuid.UUID, limit, offset int) ([]*schema.Conversation, error)

	// Atomic Updates (Performance optimizations)
	// UpdateLastMessageTimestamp is efficient for bringing threads to top of inbox
	UpdateLastMessageTimestamp(ctx context.Context, conversationID uuid.UUID, sentAt time.Time) error

	// IncrementUnreadCounts atomically increments unread counters for recipients
	IncrementUnreadCounts(ctx context.Context, conversationID uuid.UUID, recipientIDs []uuid.UUID) error

	// MarkAsRead atomically resets unread count for a user
	MarkAsRead(ctx context.Context, conversationID uuid.UUID, userID uuid.UUID) error

	// WithTx returns a new repository instance using the provided transaction
	WithTx(tx *gorm.DB) ConversationRepository
}

// MessageRepository handles individual message storage and retrieval
type MessageRepository interface {
	Create(ctx context.Context, message *schema.Message) error
	GetByID(ctx context.Context, id uuid.UUID) (*schema.Message, error)

	// Fetch messages for a specific chat (usually reverse chronological)
	ListByConversation(ctx context.Context, conversationID uuid.UUID, limit, offset int) ([]*schema.Message, error)

	// Fetch delta since a timestamp (useful for client reconnects/sync)
	ListByConversationSince(ctx context.Context, conversationID uuid.UUID, since time.Time) ([]*schema.Message, error)

	// WithTx returns a new repository instance using the provided transaction
	WithTx(tx *gorm.DB) MessageRepository
}

// ParticipantRepository handles user membership within conversations
type ParticipantRepository interface {
	Add(ctx context.Context, participant *schema.Participant) error
	Remove(ctx context.Context, conversationID, userID uuid.UUID) error

	Get(ctx context.Context, conversationID, userID uuid.UUID) (*schema.Participant, error)
	ListByConversation(ctx context.Context, conversationID uuid.UUID) ([]*schema.Participant, error)

	// UpdateLastRead updates the 'last_read_at' timestamp for read receipts
	UpdateLastRead(ctx context.Context, conversationID, userID uuid.UUID, readAt time.Time) error

	// WithTx returns a new repository instance using the provided transaction
	WithTx(tx *gorm.DB) ParticipantRepository
}
