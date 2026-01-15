package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"hauslet/internal/modules/messaging/repository/schema"
)

type messageRepository struct {
	db *gorm.DB
}

func NewMessageRepository(db *gorm.DB) MessageRepository {
	return &messageRepository{db: db}
}

// WithTx returns a new instance of the repository using the transaction
func (r *messageRepository) WithTx(tx *gorm.DB) MessageRepository {
	return &messageRepository{db: tx}
}

func (r *messageRepository) Create(ctx context.Context, message *schema.Message) error {
	return r.db.WithContext(ctx).Create(message).Error
}

func (r *messageRepository) GetByID(ctx context.Context, id uuid.UUID) (*schema.Message, error) {
	var msg schema.Message
	if err := r.db.WithContext(ctx).First(&msg, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &msg, nil
}

func (r *messageRepository) ListByConversation(ctx context.Context, conversationID uuid.UUID, limit, offset int) ([]*schema.Message, error) {
	var msgs []*schema.Message
	err := r.db.WithContext(ctx).
		Where("conversation_id = ?", conversationID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&msgs).Error
	return msgs, err
}

func (r *messageRepository) ListByConversationSince(ctx context.Context, conversationID uuid.UUID, since time.Time) ([]*schema.Message, error) {
	var msgs []*schema.Message
	err := r.db.WithContext(ctx).
		Where("conversation_id = ? AND created_at > ?", conversationID, since).
		Order("created_at ASC").
		Find(&msgs).Error
	return msgs, err
}

// MarkAsReadByUser updates the read_by JSONB field for all messages in a conversation
// that were not sent by the user. Uses PostgreSQL's jsonb_set to atomically add the read receipt.
func (r *messageRepository) MarkAsReadByUser(ctx context.Context, conversationID, userID uuid.UUID, readAt time.Time) error {
	// Update all messages in conversation not sent by this user
	// Uses JSONB concatenation to add/update the user's read timestamp
	return r.db.WithContext(ctx).
		Model(&schema.Message{}).
		Where("conversation_id = ? AND sender_id != ?", conversationID, userID).
		Update("read_by", gorm.Expr(
			"COALESCE(read_by, '{}'::jsonb) || jsonb_build_object(?, ?::text)",
			userID.String(),
			readAt.Format(time.RFC3339),
		)).Error
}
