package schema

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

// VerificationAttemptSchema is the GORM schema for verification_attempts table
type VerificationAttemptSchema struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey"`
	SessionID    uuid.UUID `gorm:"type:uuid;not null;index:idx_session_attempts"`
	ProviderName string    `gorm:"type:varchar(50);not null;index"`
	Status       string    `gorm:"type:varchar(20);not null;index"`

	// Result (JSONB)
	Result JSONBMap `gorm:"type:jsonb"`

	// Evidence references
	EvidenceIDs pq.StringArray `gorm:"type:uuid[]"`

	// Timing
	SubmittedAt    time.Time  `gorm:"index"`
	CompletedAt    *time.Time `gorm:"index"`
	ProcessingTime *int64     `gorm:"type:bigint"` // Duration in nanoseconds

	// Provider tracking
	ProviderSessionID *string    `gorm:"type:varchar(255);index"`
	WebhookReceived   bool       `gorm:"default:false"`
	WebhookReceivedAt *time.Time

	// Metadata
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// TableName specifies the table name
func (VerificationAttemptSchema) TableName() string {
	return "verification_attempts"
}

// BeforeCreate sets the ID if not already set
func (a *VerificationAttemptSchema) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}
