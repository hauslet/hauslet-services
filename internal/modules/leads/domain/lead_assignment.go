package domain

import (
	"time"

	"github.com/google/uuid"
)

// LeadAssignment tracks the routing history of a lead (for analytics and audit purposes)
type LeadAssignment struct {
	ID     uuid.UUID
	LeadID uuid.UUID

	// Assignment Details
	FromUserID *uuid.UUID // Previous assignee (nil for initial assignment)
	ToUserID   uuid.UUID  // New assignee

	// Assignment Context
	Reason     AssignmentReason // auto, manual, reassign, escalate
	Notes      *string          // Optional notes explaining the assignment
	AssignedBy *uuid.UUID       // Who triggered the assignment (nil for auto-assignments)

	AssignedAt time.Time
}

// IsInitialAssignment returns true if this is the first assignment for the lead
func (a *LeadAssignment) IsInitialAssignment() bool {
	return a.FromUserID == nil
}

// IsReassignment returns true if the lead was reassigned from another user
func (a *LeadAssignment) IsReassignment() bool {
	return a.FromUserID != nil
}

// IsAutoAssignment returns true if the assignment was automatic
func (a *LeadAssignment) IsAutoAssignment() bool {
	return a.Reason == ReasonAuto
}

// IsManualAssignment returns true if the assignment was done manually
func (a *LeadAssignment) IsManualAssignment() bool {
	return a.Reason == ReasonManual
}

// IsEscalation returns true if the assignment was an escalation
func (a *LeadAssignment) IsEscalation() bool {
	return a.Reason == ReasonEscalate
}

// HasNotes returns true if the assignment has notes attached
func (a *LeadAssignment) HasNotes() bool {
	return a.Notes != nil && *a.Notes != ""
}

// GetAssignerID returns the ID of the user who performed the assignment, or nil if system-assigned
func (a *LeadAssignment) GetAssignerID() *uuid.UUID {
	return a.AssignedBy
}
