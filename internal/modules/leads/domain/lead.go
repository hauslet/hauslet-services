package domain

import (
	"time"

	"github.com/google/uuid"
)

// Lead represents an inquiry from a potential customer about a listing
type Lead struct {
	ID         uuid.UUID
	ListingID  uuid.UUID
	BusinessID *uuid.UUID // Nullable - direct to owner if nil

	// User Tracking (Hybrid Authentication)
	UserID     *uuid.UUID // Nullable - authenticated users only
	IsVerified bool       // True if created by authenticated user with verified profile

	// Contact Information
	Name        string
	Email       string
	PhoneNumber *string

	// Lead Details
	Message string
	Source  LeadSource
	Status  LeadStatus

	// Spam Detection
	SpamScore float64 // 0.0 = clean, 1.0 = definitely spam
	IsSpam    bool

	// Assignment & Routing
	AssignedTo   *uuid.UUID
	AssignedAt   *time.Time
	AutoAssigned bool

	// Metadata (tracking and context)
	UserAgent      *string
	IPAddress      *string
	ReferrerURL    *string
	UTMParams      map[string]string // UTM tracking parameters
	CustomMetadata map[string]any    // Extensible metadata

	// Response Tracking
	FirstResponseAt *time.Time
	ResponseTime    *int64 // seconds to first response
	ResponseCount   int

	// Timestamps
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

// IsNew returns true if the lead has just been created and not yet contacted
func (l *Lead) IsNew() bool {
	return l.Status == StatusNew && l.FirstResponseAt == nil
}

// IsAuthenticatedUser returns true if the lead was created by an authenticated user
func (l *Lead) IsAuthenticatedUser() bool {
	return l.UserID != nil
}

// IsAnonymous returns true if the lead was created anonymously (no user account)
func (l *Lead) IsAnonymous() bool {
	return l.UserID == nil
}

// IsAssigned returns true if the lead is assigned to an agent
func (l *Lead) IsAssigned() bool {
	return l.AssignedTo != nil && l.AssignedAt != nil
}

// IsActive returns true if the lead is in an active state (not spam, not archived, not lost)
func (l *Lead) IsActive() bool {
	return l.Status != StatusSpam && l.Status != StatusArchived && l.Status != StatusLost
}

// IsConverted returns true if the lead successfully converted
func (l *Lead) IsConverted() bool {
	return l.Status == StatusConverted
}

// MarkAsSpam marks the lead as spam
func (l *Lead) MarkAsSpam() {
	l.IsSpam = true
	l.Status = StatusSpam
	l.UpdatedAt = time.Now()
}

// Assign assigns the lead to a user
func (l *Lead) Assign(userID uuid.UUID, auto bool) {
	now := time.Now()
	l.AssignedTo = &userID
	l.AssignedAt = &now
	l.AutoAssigned = auto

	// Update status to assigned if currently new
	if l.Status == StatusNew {
		l.Status = StatusAssigned
	}

	l.UpdatedAt = now
}

// Unassign removes the assignment from the lead
func (l *Lead) Unassign() {
	l.AssignedTo = nil
	l.AssignedAt = nil
	l.AutoAssigned = false

	// Revert to new status if it was just assigned
	if l.Status == StatusAssigned {
		l.Status = StatusNew
	}

	l.UpdatedAt = time.Now()
}

// RecordResponse records that the lead has been contacted
func (l *Lead) RecordResponse() {
	now := time.Now()

	// Record first response time
	if l.FirstResponseAt == nil {
		l.FirstResponseAt = &now
		responseTime := int64(now.Sub(l.CreatedAt).Seconds())
		l.ResponseTime = &responseTime
	}

	l.ResponseCount++
	l.UpdatedAt = now
}

// UpdateStatus updates the lead status
func (l *Lead) UpdateStatus(newStatus LeadStatus) error {
	if !newStatus.IsValid() {
		return ErrInvalidInput
	}

	l.Status = newStatus
	l.UpdatedAt = time.Now()
	return nil
}

// CanBeContacted returns true if the lead can be contacted (not spam, not archived)
func (l *Lead) CanBeContacted() bool {
	return !l.IsSpam && l.Status != StatusSpam && l.Status != StatusArchived
}

// CanBeAssigned returns true if the lead can be assigned to an agent
func (l *Lead) CanBeAssigned() bool {
	return !l.IsSpam && l.Status != StatusSpam && l.Status != StatusArchived
}

// CanBeReassigned returns true if the lead can be reassigned to a different agent
func (l *Lead) CanBeReassigned() bool {
	return l.IsAssigned() && l.CanBeAssigned()
}

// Archive archives the lead
func (l *Lead) Archive() {
	l.Status = StatusArchived
	l.UpdatedAt = time.Now()
}

// GetResponseTimeInSeconds returns the response time in seconds, or nil if not yet responded
func (l *Lead) GetResponseTimeInSeconds() *int64 {
	return l.ResponseTime
}

// HasResponded returns true if the lead has been responded to at least once
func (l *Lead) HasResponded() bool {
	return l.FirstResponseAt != nil
}
