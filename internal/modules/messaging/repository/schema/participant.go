package schema

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Participant struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CreatedAt time.Time      `gorm:"not null"`
	UpdatedAt time.Time      `gorm:"not null"`
	DeletedAt gorm.DeletedAt `gorm:"index"`

	// Foreign Keys
	ConversationID uuid.UUID `gorm:"type:uuid;not null;index:idx_conv_user"`
	UserID         uuid.UUID `gorm:"type:uuid;not null;index:idx_conv_user"`

	// Role & State
	Type string `gorm:"type:varchar(50);not null"` // owner, inquirer, support_agent, ai_agent

	// Activity Tracking
	JoinedAt   time.Time  `gorm:"not null;default:now()"`
	LeftAt     *time.Time `gorm:""`
	LastReadAt *time.Time `gorm:""`

	// Settings
	IsMuted   bool `gorm:"default:false"`
	IsVisible bool `gorm:"default:true"` // Hidden for "shadow" support agents or AI monitoring
}

func (Participant) TableName() string {
	return "conversation_participants"
}
