package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"hauslet/internal/modules/messaging/repository/schema"
)

type participantRepository struct {
	db *gorm.DB
}

func NewParticipantRepository(db *gorm.DB) ParticipantRepository {
	return &participantRepository{db: db}
}

// WithTx returns a new instance of the repository using the transaction
func (r *participantRepository) WithTx(tx *gorm.DB) ParticipantRepository {
	return &participantRepository{db: tx}
}

func (r *participantRepository) Add(ctx context.Context, participant *schema.Participant) error {
	return r.db.WithContext(ctx).Create(participant).Error
}

func (r *participantRepository) Remove(ctx context.Context, conversationID, userID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("conversation_id = ? AND user_id = ?", conversationID, userID).
		Delete(&schema.Participant{}).Error
}

func (r *participantRepository) Get(ctx context.Context, conversationID, userID uuid.UUID) (*schema.Participant, error) {
	var p schema.Participant
	err := r.db.WithContext(ctx).
		Where("conversation_id = ? AND user_id = ?", conversationID, userID).
		First(&p).Error
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *participantRepository) ListByConversation(ctx context.Context, conversationID uuid.UUID) ([]*schema.Participant, error) {
	var participants []*schema.Participant
	err := r.db.WithContext(ctx).
		Where("conversation_id = ?", conversationID).
		Find(&participants).Error
	return participants, err
}

func (r *participantRepository) UpdateLastRead(ctx context.Context, conversationID, userID uuid.UUID, readAt time.Time) error {
	return r.db.WithContext(ctx).Model(&schema.Participant{}).
		Where("conversation_id = ? AND user_id = ?", conversationID, userID).
		Update("last_read_at", readAt).Error
}
