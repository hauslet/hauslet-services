package domain

import (
	"time"

	"github.com/google/uuid"
)

// CalendarEvent represents a calendar event in the domain model
type CalendarEvent struct {
	ID        uuid.UUID   `json:"id"`
	ListingID uuid.UUID   `json:"listing_id"`
	EventType EventType   `json:"event_type"`
	Status    EventStatus `json:"status"`
	BookingID *uuid.UUID  `json:"booking_id,omitempty"`

	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`

	// Polymorphic details based on EventType
	ShowingDetails     *ShowingDetail     `json:"showing_details,omitempty"`
	MaintenanceDetails *MaintenanceDetail `json:"maintenance_details,omitempty"`
	BlockDetails       *BlockDetail       `json:"block_details,omitempty"`
	OpenHouseDetails   *OpenHouseDetail   `json:"open_house_details,omitempty"`

	CreatedBy   *uuid.UUID `json:"created_by,omitempty"`
	UpdatedBy   *uuid.UUID `json:"updated_by,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	ArchivedAt  *time.Time `json:"archived_at,omitempty"`

	Version   int        `json:"version"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

// ShowingDetail contains showing-specific information
type ShowingDetail struct {
	ProspectID    *uuid.UUID `json:"prospect_id,omitempty"`
	ProspectName  string     `json:"prospect_name"`
	ProspectEmail string     `json:"prospect_email"`
	ProspectPhone *string    `json:"prospect_phone,omitempty"`
	AgentID       *uuid.UUID `json:"agent_id,omitempty"`
	AgentName     *string    `json:"agent_name,omitempty"`
	Notes         *string    `json:"notes,omitempty"`
	Attended      *bool      `json:"attended,omitempty"`
	Feedback      *string    `json:"feedback,omitempty"`
}

// MaintenanceDetail contains maintenance-specific information
type MaintenanceDetail struct {
	MaintenanceType           MaintenanceType `json:"maintenance_type"`
	Title                     string          `json:"title"`
	Description               *string         `json:"description,omitempty"`
	VendorID                  *uuid.UUID      `json:"vendor_id,omitempty"`
	VendorName                *string         `json:"vendor_name,omitempty"`
	VendorContact             *string         `json:"vendor_contact,omitempty"`
	EstimatedCost             *float64        `json:"estimated_cost,omitempty"`
	ActualCost                *float64        `json:"actual_cost,omitempty"`
	Disruptive                bool            `json:"disruptive"`
	RequiresGuestNotification bool            `json:"requires_guest_notification"`
	WorkCompleted             *bool           `json:"work_completed,omitempty"`
	CompletionNotes           *string         `json:"completion_notes,omitempty"`
}

// BlockDetail contains block-specific information
type BlockDetail struct {
	Reason    string  `json:"reason"`
	Notes     *string `json:"notes,omitempty"`
	OwnerStay bool    `json:"owner_stay"`
}

// OpenHouseDetail contains open house-specific information
type OpenHouseDetail struct {
	Title        string     `json:"title"`
	Description  *string    `json:"description,omitempty"`
	MaxAttendees int        `json:"max_attendees"`
	AgentID      *uuid.UUID `json:"agent_id,omitempty"`
	AgentName    *string    `json:"agent_name,omitempty"`
	Attendees    []Attendee `json:"attendees,omitempty"`
}

// Attendee represents someone who registered for an open house
type Attendee struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	Phone        *string   `json:"phone,omitempty"`
	RegisteredAt time.Time `json:"registered_at"`
	Attended     *bool     `json:"attended,omitempty"`
}

// CalendarConfig stores calendar configuration for a listing
type CalendarConfig struct {
	ID                  uuid.UUID `json:"id"`
	ListingID           uuid.UUID `json:"listing_id"`
	BufferHours         int       `json:"buffer_hours"`
	LeadTimeHours       int       `json:"lead_time_hours"`
	Timezone            string    `json:"timezone"`
	BookingWindowMonths int       `json:"booking_window_months"`
	InstantBooking      bool      `json:"instant_booking"`
	SameDayBooking      bool      `json:"same_day_booking"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// RecurringEventPattern stores recurring event configurations
type RecurringEventPattern struct {
	ID                 uuid.UUID          `json:"id"`
	ListingID          uuid.UUID          `json:"listing_id"`
	EventType          EventType          `json:"event_type"`
	Frequency          string             `json:"frequency"`
	Interval           int                `json:"interval"`
	DaysOfWeek         *string            `json:"days_of_week,omitempty"`
	DayOfMonth         *int               `json:"day_of_month,omitempty"`
	StartTimeOfDay     string             `json:"start_time_of_day"`
	EndTimeOfDay       string             `json:"end_time_of_day"`
	StartDate          time.Time          `json:"start_date"`
	EndDate            *time.Time         `json:"end_date,omitempty"`
	Exceptions         []time.Time        `json:"exceptions,omitempty"`
	MaintenanceDetails *MaintenanceDetail `json:"maintenance_details,omitempty"`
	OpenHouseDetails   *OpenHouseDetail   `json:"open_house_details,omitempty"`
	Active             bool               `json:"active"`
	CreatedAt          time.Time          `json:"created_at"`
	UpdatedAt          time.Time          `json:"updated_at"`
	DeletedAt          *time.Time         `json:"deleted_at,omitempty"`
}

// AvailabilityResult represents the result of an availability check
type AvailabilityResult struct {
	Available bool             `json:"available"`
	Conflicts []*CalendarEvent `json:"conflicts,omitempty"`
	Reason    string           `json:"reason,omitempty"`
}

// --- DOMAIN METHODS ---

// IsActive checks if the event is active (not deleted or cancelled)
func (e *CalendarEvent) IsActive() bool {
	return e.DeletedAt == nil && e.Status != EventStatusCancelled
}

// IsConfirmed checks if the event is confirmed
func (e *CalendarEvent) IsConfirmed() bool {
	return e.Status == EventStatusConfirmed || e.Status == EventStatusInProgress
}

// IsPending checks if the event is pending confirmation
func (e *CalendarEvent) IsPending() bool {
	return e.Status == EventStatusPending
}

// IsCompleted checks if the event is completed
func (e *CalendarEvent) IsCompleted() bool {
	return e.Status == EventStatusCompleted || e.CompletedAt != nil
}

// IsCancelled checks if the event is cancelled
func (e *CalendarEvent) IsCancelled() bool {
	return e.Status == EventStatusCancelled
}

// IsBooking checks if the event is a booking
func (e *CalendarEvent) IsBooking() bool {
	return e.EventType == EventTypeBooking
}

// IsShowing checks if the event is a showing
func (e *CalendarEvent) IsShowing() bool {
	return e.EventType == EventTypeShowing
}

// IsMaintenance checks if the event is maintenance
func (e *CalendarEvent) IsMaintenance() bool {
	return e.EventType == EventTypeMaintenance
}

// IsBlock checks if the event is a block
func (e *CalendarEvent) IsBlock() bool {
	return e.EventType == EventTypeBlock
}

// IsOpenHouse checks if the event is an open house
func (e *CalendarEvent) IsOpenHouse() bool {
	return e.EventType == EventTypeOpenHouse
}

// BlocksBookings returns true if this event should prevent new bookings
func (e *CalendarEvent) BlocksBookings() bool {
	switch e.EventType {
	case EventTypeBooking:
		// Booking holds block the timeline unless explicitly cancelled.
		return e.Status != EventStatusCancelled
	case EventTypeBlock:
		// Blocks always prevent bookings
		return true
	case EventTypeMaintenance:
		// Only disruptive maintenance blocks bookings
		return e.MaintenanceDetails != nil && e.MaintenanceDetails.Disruptive
	case EventTypeShowing, EventTypeOpenHouse:
		// Showings and open houses don't block bookings
		return false
	default:
		return false
	}
}

// Duration returns the duration of the event
func (e *CalendarEvent) Duration() time.Duration {
	return e.EndTime.Sub(e.StartTime)
}

// DurationHours returns the duration in hours
func (e *CalendarEvent) DurationHours() float64 {
	return e.Duration().Hours()
}

// DurationDays returns the duration in days (rounded up)
func (e *CalendarEvent) DurationDays() int {
	hours := e.DurationHours()
	days := int(hours / 24)
	if hours > float64(days*24) {
		days++
	}
	return days
}

// OverlapsWith checks if this event overlaps with another event
func (e *CalendarEvent) OverlapsWith(other *CalendarEvent) bool {
	// Events don't overlap if they're in different listings
	if e.ListingID != other.ListingID {
		return false
	}

	// Check time overlap: event1.start < event2.end AND event1.end > event2.start
	return e.StartTime.Before(other.EndTime) && e.EndTime.After(other.StartTime)
}

// ConflictsWith checks if this event conflicts with another event (considering business rules)
func (e *CalendarEvent) ConflictsWith(other *CalendarEvent) bool {
	// No conflict if they don't overlap
	if !e.OverlapsWith(other) {
		return false
	}

	// No conflict if either event is cancelled or deleted
	if !e.IsActive() || !other.IsActive() {
		return false
	}

	// Check if either event blocks bookings
	return e.BlocksBookings() || other.BlocksBookings()
}

// CanBeCancelled checks if the event can be cancelled
func (e *CalendarEvent) CanBeCancelled() bool {
	// Already cancelled or completed
	if e.IsCancelled() || e.IsCompleted() {
		return false
	}

	// Can't cancel past events
	if e.EndTime.Before(time.Now()) {
		return false
	}

	return true
}

// CanBeModified checks if the event can be modified
func (e *CalendarEvent) CanBeModified() bool {
	// Can't modify cancelled or completed events
	if e.IsCancelled() || e.IsCompleted() {
		return false
	}

	// Can't modify past events
	if e.StartTime.Before(time.Now()) {
		return false
	}

	return true
}

// GetTotalAttendees returns the current attendee count for open houses
func (e *CalendarEvent) GetTotalAttendees() int {
	if e.OpenHouseDetails == nil {
		return 0
	}
	return len(e.OpenHouseDetails.Attendees)
}

// HasCapacity checks if an open house has capacity for more attendees
func (e *CalendarEvent) HasCapacity() bool {
	if e.EventType != EventTypeOpenHouse || e.OpenHouseDetails == nil {
		return false
	}
	return e.GetTotalAttendees() < e.OpenHouseDetails.MaxAttendees
}

// CanAddAttendee checks if a new attendee can be added to an open house
func (e *CalendarEvent) CanAddAttendee() bool {
	return e.IsOpenHouse() && e.HasCapacity() && !e.IsCancelled() && !e.IsCompleted()
}

// IsRecurring checks if this event is part of a recurring pattern
func (p *RecurringEventPattern) IsRecurring() bool {
	return p.Frequency != "" && p.Interval > 0
}

// IsActive checks if the recurring pattern is currently active
func (p *RecurringEventPattern) IsActive() bool {
	return p.Active && p.DeletedAt == nil
}

// IsInDateRange checks if a given date falls within the pattern's validity period
func (p *RecurringEventPattern) IsInDateRange(date time.Time) bool {
	if date.Before(p.StartDate) {
		return false
	}
	if p.EndDate != nil && date.After(*p.EndDate) {
		return false
	}
	return true
}

// IsException checks if a given date is in the exceptions list
func (p *RecurringEventPattern) IsException(date time.Time) bool {
	for _, exception := range p.Exceptions {
		if date.Year() == exception.Year() &&
			date.Month() == exception.Month() &&
			date.Day() == exception.Day() {
			return true
		}
	}
	return false
}
