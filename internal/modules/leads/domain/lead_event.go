package domain

import (
	"time"

	"github.com/google/uuid"
)

// LeadEvent tracks all state changes and actions performed on a lead (audit trail)
type LeadEvent struct {
	ID     uuid.UUID
	LeadID uuid.UUID

	// Event Details
	EventType LeadEventType
	ActorID   *uuid.UUID // Nil for system events
	ActorType ActorType  // user, system, automation

	// Status Transition
	OldStatus *LeadStatus
	NewStatus *LeadStatus

	// Change Details (JSONB in database)
	Changes map[string]any // Field-level changes for detailed tracking
	Notes   *string        // Optional notes about the event

	CreatedAt time.Time
}

// IsSystemEvent returns true if this event was triggered by the system
func (e *LeadEvent) IsSystemEvent() bool {
	return e.ActorType == ActorSystem
}

// IsUserEvent returns true if this event was triggered by a user
func (e *LeadEvent) IsUserEvent() bool {
	return e.ActorType == ActorUser
}

// IsAutomationEvent returns true if this event was triggered by automation
func (e *LeadEvent) IsAutomationEvent() bool {
	return e.ActorType == ActorAutomation
}

// HasStatusChange returns true if this event involved a status change
func (e *LeadEvent) HasStatusChange() bool {
	return e.OldStatus != nil && e.NewStatus != nil
}

// GetStatusChange returns the status change details
func (e *LeadEvent) GetStatusChange() (oldStatus, newStatus LeadStatus, hasChange bool) {
	if !e.HasStatusChange() {
		return "", "", false
	}
	return *e.OldStatus, *e.NewStatus, true
}

// HasNotes returns true if the event has notes attached
func (e *LeadEvent) HasNotes() bool {
	return e.Notes != nil && *e.Notes != ""
}
