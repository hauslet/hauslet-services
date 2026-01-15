package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"hauslet/internal/modules/messaging/repository/schema"
)

type conversationRepository struct {
	db *gorm.DB
}

func NewConversationRepository(db *gorm.DB) ConversationRepository {
	return &conversationRepository{db: db}
}

// WithTx returns a new instance of the repository using the transaction
func (r *conversationRepository) WithTx(tx *gorm.DB) ConversationRepository {
	return &conversationRepository{db: tx}
}

func (r *conversationRepository) Create(ctx context.Context, conversation *schema.Conversation) error {
	return r.db.WithContext(ctx).Create(conversation).Error
}

func (r *conversationRepository) GetByID(ctx context.Context, id uuid.UUID) (*schema.Conversation, error) {
	var conv schema.Conversation
	if err := r.db.WithContext(ctx).First(&conv, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &conv, nil
}

func (r *conversationRepository) Update(ctx context.Context, conversation *schema.Conversation) error {
	return r.db.WithContext(ctx).Save(conversation).Error
}

func (r *conversationRepository) GetByContext(ctx context.Context, contextType string, contextID uuid.UUID) (*schema.Conversation, error) {
	var conv schema.Conversation
	err := r.db.WithContext(ctx).
		Where("context_type = ? AND context_id = ?", contextType, contextID).
		First(&conv).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &conv, nil
}

func (r *conversationRepository) GetSupportConversation(ctx context.Context, userID uuid.UUID) (*schema.Conversation, error) {
	var conv schema.Conversation
	// Finds conversation of type 'support' where user is a participant
	err := r.db.WithContext(ctx).
		Joins("JOIN conversation_participants p ON p.conversation_id = conversations.id").
		Where("conversations.type = ? AND p.user_id = ?", "support", userID).
		First(&conv).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &conv, nil
}

func (r *conversationRepository) ListForUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*schema.Conversation, error) {
	var convs []*schema.Conversation
	err := r.db.WithContext(ctx).
		Distinct("conversations.*").
		Joins("JOIN conversation_participants p ON p.conversation_id = conversations.id").
		Where("p.user_id = ? AND p.is_visible = true", userID).
		Order("conversations.last_message_at DESC NULLS LAST").
		Limit(limit).
		Offset(offset).
		Find(&convs).Error

	return convs, err
}

func (r *conversationRepository) ListSupportQueue(ctx context.Context, assignedAgentID *uuid.UUID, limit, offset int) ([]*schema.Conversation, error) {
	query := r.db.WithContext(ctx).
		Where("type = ?", "support").
		Order("last_message_at ASC")

	if assignedAgentID != nil {
		query = query.Where("support_state->>'assigned_agent_id' = ?", assignedAgentID.String())
	} else {
		query = query.Where("support_state->>'assigned_agent_id' IS NULL")
	}

	var convs []*schema.Conversation
	err := query.Limit(limit).Offset(offset).Find(&convs).Error
	return convs, err
}

func (r *conversationRepository) UpdateLastMessageTimestamp(ctx context.Context, conversationID uuid.UUID, sentAt time.Time) error {
	return r.db.WithContext(ctx).Model(&schema.Conversation{}).
		Where("id = ?", conversationID).
		Update("last_message_at", sentAt).Error
}

func (r *conversationRepository) IncrementUnreadCounts(ctx context.Context, conversationID uuid.UUID, recipientIDs []uuid.UUID) error {
	if len(recipientIDs) == 0 {
		return nil
	}

	sql := "UPDATE conversations SET unread_counts = unread_counts"
	params := []any{}

	for _, uid := range recipientIDs {
		sql += " || jsonb_build_object(?, (COALESCE(unread_counts->>?,'0')::int + 1))"
		params = append(params, uid.String(), uid.String())
	}

	sql += " WHERE id = ?"
	params = append(params, conversationID)

	return r.db.WithContext(ctx).Exec(sql, params...).Error
}

func (r *conversationRepository) MarkAsRead(ctx context.Context, conversationID uuid.UUID, userID uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&schema.Conversation{}).
		Where("id = ?", conversationID).
		UpdateColumn("unread_counts", gorm.Expr("unread_counts || jsonb_build_object(?, 0)", userID.String())).
		Error
}
