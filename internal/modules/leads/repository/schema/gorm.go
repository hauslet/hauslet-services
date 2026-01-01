package schema

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Lead represents the GORM schema for leads table
type Lead struct {
	ID         uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	ListingID  uuid.UUID  `gorm:"type:uuid;not null;index:idx_leads_listing"`
	BusinessID *uuid.UUID `gorm:"type:uuid;index:idx_leads_business"`

	// Contact Information
	Name        string  `gorm:"type:varchar(255);not null"`
	Email       string  `gorm:"type:varchar(255);not null;index:idx_leads_email"`
	PhoneNumber *string `gorm:"type:varchar(50)"`

	// Lead Details
	Message string `gorm:"type:text;not null"`
	Source  string `gorm:"type:varchar(50);not null;index:idx_leads_source"`
	Status  string `gorm:"type:varchar(50);not null;default:'new';index:idx_leads_status"`

	// Spam Detection
	SpamScore float64 `gorm:"type:decimal(3,2);default:0.0"`
	IsSpam    bool    `gorm:"default:false;index:idx_leads_spam"`

	// Assignment
	AssignedTo   *uuid.UUID `gorm:"type:uuid;index:idx_leads_assigned_to"`
	AssignedAt   *time.Time
	AutoAssigned bool `gorm:"default:false"`

	// Metadata (JSONB)
	UserAgent      *string           `gorm:"type:text"`
	IPAddress      *string           `gorm:"type:varchar(45);index:idx_leads_ip"` // IPv6 compatible
	ReferrerURL    *string           `gorm:"type:text"`
	UTMParams      map[string]string `gorm:"type:jsonb;serializer:json"`
	CustomMetadata map[string]any    `gorm:"type:jsonb;serializer:json"`

	// Response Tracking
	FirstResponseAt *time.Time
	ResponseTime    *int64 // seconds
	ResponseCount   int    `gorm:"default:0"`

	// Timestamps
	CreatedAt time.Time      `gorm:"autoCreateTime;index:idx_leads_created"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// TableName overrides the table name for GORM
func (Lead) TableName() string {
	return "leads"
}

// BeforeSave is a GORM hook that runs before saving a lead
func (l *Lead) BeforeSave(tx *gorm.DB) error {
	// Ensure spam score is within valid range
	if l.SpamScore < 0.0 {
		l.SpamScore = 0.0
	}
	if l.SpamScore > 1.0 {
		l.SpamScore = 1.0
	}

	// Auto-mark as spam if spam score is high
	if l.SpamScore >= 0.7 && !l.IsSpam {
		l.IsSpam = true
		if l.Status == "new" {
			l.Status = "spam"
		}
	}

	return nil
}

// LeadEvent represents the GORM schema for lead_events table (audit trail)
type LeadEvent struct {
	ID     uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	LeadID uuid.UUID  `gorm:"type:uuid;not null;index:idx_lead_events_lead"`

	// Event Details
	EventType string     `gorm:"type:varchar(50);not null"`
	ActorID   *uuid.UUID `gorm:"type:uuid"`
	ActorType string     `gorm:"type:varchar(50);not null"`

	// Status Transition
	OldStatus *string `gorm:"type:varchar(50)"`
	NewStatus *string `gorm:"type:varchar(50)"`

	// Change Details (JSONB)
	Changes map[string]any `gorm:"type:jsonb;serializer:json"`
	Notes   *string        `gorm:"type:text"`

	CreatedAt time.Time `gorm:"autoCreateTime;index:idx_lead_events_created"`
}

// TableName overrides the table name for GORM
func (LeadEvent) TableName() string {
	return "lead_events"
}

// LeadAssignment represents the GORM schema for lead_assignments table (routing history)
type LeadAssignment struct {
	ID     uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	LeadID uuid.UUID  `gorm:"type:uuid;not null;index:idx_lead_assignments_lead"`

	// Assignment Details
	FromUserID *uuid.UUID `gorm:"type:uuid"`
	ToUserID   uuid.UUID  `gorm:"type:uuid;not null;index:idx_lead_assignments_user"`

	// Assignment Context
	Reason     string     `gorm:"type:varchar(50);not null"`
	Notes      *string    `gorm:"type:text"`
	AssignedBy *uuid.UUID `gorm:"type:uuid"`

	AssignedAt time.Time `gorm:"autoCreateTime"`
}

// TableName overrides the table name for GORM
func (LeadAssignment) TableName() string {
	return "lead_assignments"
}
