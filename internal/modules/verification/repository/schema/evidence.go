package schema

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// VerificationEvidenceSchema is the GORM schema for verification_evidence table
type VerificationEvidenceSchema struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	SessionID uuid.UUID `gorm:"type:uuid;not null;index:idx_session_evidence"`
	AttemptID *uuid.UUID `gorm:"type:uuid;index"`
	Type      string    `gorm:"type:varchar(50);not null"`

	// Evidence metadata (JSONB) - contains URL, hash, size, mimeType, etc.
	Metadata JSONBMap `gorm:"type:jsonb;not null"`

	// Verification
	Verified   bool       `gorm:"default:false"`
	VerifiedAt *time.Time

	// Lifecycle
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time `gorm:"index"` // Soft delete for compliance
}

// TableName specifies the table name
func (VerificationEvidenceSchema) TableName() string {
	return "verification_evidence"
}

// BeforeCreate sets the ID if not already set
func (e *VerificationEvidenceSchema) BeforeCreate(tx *gorm.DB) error {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	return nil
}
