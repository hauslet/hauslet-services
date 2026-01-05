package schema

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// VerificationSessionSchema is the GORM schema for verification_sessions table
type VerificationSessionSchema struct {
	ID      uuid.UUID `gorm:"type:uuid;primaryKey"`
	UserID  uuid.UUID `gorm:"type:uuid;not null;index:idx_user_verification"`
	Type    string    `gorm:"type:varchar(20);not null;index"` // identity, phone, address, business
	Tier    string    `gorm:"type:varchar(20);not null"`
	Status  string    `gorm:"type:varchar(20);not null;index"`
	Country string    `gorm:"type:char(2);not null;index"` // ISO 3166-1 alpha-2

	// Type-specific verification data (JSONB - union type)
	VerificationData JSONBMap `gorm:"type:jsonb;not null"`

	// Attempts tracking
	MaxAttempts   int        `gorm:"not null;default:3"`
	AttemptsUsed  int        `gorm:"not null;default:0"`
	LastAttemptAt *time.Time `gorm:"index"`

	// Resolution
	ApprovedAt      *time.Time
	RejectedAt      *time.Time
	RejectionReason *string `gorm:"type:varchar(50)"`
	RejectionNotes  *string `gorm:"type:text"`

	// Metadata
	CreatedAt   time.Time
	UpdatedAt   time.Time
	ExpiresAt   time.Time      `gorm:"not null;index"`
	CompletedAt *time.Time     `gorm:"index"`
	DeletedAt   gorm.DeletedAt `gorm:"index"`

	// Client context
	IPAddress *string `gorm:"type:inet"`
	UserAgent *string `gorm:"type:text"`
}

// TableName specifies the table name
func (VerificationSessionSchema) TableName() string {
	return "verification_sessions"
}

// BeforeCreate sets the ID if not already set
func (s *VerificationSessionSchema) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}
