package schema

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Message struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CreatedAt time.Time      `gorm:"not null;index"` // Index for pagination
	UpdatedAt time.Time      `gorm:"not null"`
	DeletedAt gorm.DeletedAt `gorm:"index"`

	// Foreign Key
	ConversationID uuid.UUID `gorm:"type:uuid;not null;index"`

	// Sender Info
	SenderID   uuid.UUID `gorm:"type:uuid;not null;index"`
	SenderType string    `gorm:"type:varchar(50);not null"` // user, ai_agent, support_agent

	// Content
	MessageType string `gorm:"type:varchar(50);not null;default:'text'"` // text, image, file, system
	Content     string `gorm:"type:text;not null"`

	// JSONB Fields
	Metadata  datatypes.JSON `gorm:"type:jsonb"` // For file URLs, system event details
	ReadBy    datatypes.JSON `gorm:"type:jsonb"` // Map[UserID]Time - explicit read receipts
	AIContext datatypes.JSON `gorm:"type:jsonb"` // Token usage, confidence score, intent detected
}

func (Message) TableName() string {
	return "messages"
}
