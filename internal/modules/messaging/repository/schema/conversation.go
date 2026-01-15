package schema

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Conversation struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CreatedAt time.Time      `gorm:"not null"`
	UpdatedAt time.Time      `gorm:"not null"`
	DeletedAt gorm.DeletedAt `gorm:"index"`

	// Core Fields
	Type   string `gorm:"type:varchar(50);not null;index"` // inquiry, transaction, support
	Status string `gorm:"type:varchar(50);not null;default:'active';index"`

	// Polymorphic Context (links to Lead, Booking, etc.)
	// Unique index ensures only one active conversation per context
	ContextType string    `gorm:"type:varchar(50);index:idx_context"`
	ContextID   uuid.UUID `gorm:"type:uuid;index:idx_context"`

	// Sorting & Preview
	LastMessageAt *time.Time `gorm:"index;sort:desc"`

	// JSONB Fields for fast access without joins
	// Map[UserID]int - Tracks unread messages per user
	UnreadCounts datatypes.JSON `gorm:"type:jsonb;default:'{}'"`

	// Support workflow state (AI session, escalation status, etc.)
	SupportState datatypes.JSON `gorm:"type:jsonb"`

	// Generic metadata
	Metadata datatypes.JSON `gorm:"type:jsonb"`

	// Relations
	Messages     []Message     `gorm:"foreignKey:ConversationID;constraint:OnDelete:CASCADE"`
	Participants []Participant `gorm:"foreignKey:ConversationID;constraint:OnDelete:CASCADE"`
}

func (Conversation) TableName() string {
	return "conversations"
}
