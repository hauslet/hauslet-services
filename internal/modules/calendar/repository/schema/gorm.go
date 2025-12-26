package schema

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// --- Validation Errors ---
var (
	ErrInvalidEventType       = errors.New("invalid event type")
	ErrInvalidEventStatus     = errors.New("invalid event status")
	ErrInvalidMaintenanceType = errors.New("invalid maintenance type")
	ErrEventEndBeforeStart    = errors.New("event end time must be after start time")
	ErrListingIDRequired      = errors.New("listing ID is required")
	ErrInvalidMaxAttendees    = errors.New("max attendees must be positive")
	ErrInvalidBufferHours     = errors.New("buffer hours must be non-negative")
)

// CalendarEvent represents any event in a property's calendar
type CalendarEvent struct {
	ID        uuid.UUID   `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	ListingID uuid.UUID   `gorm:"type:uuid;not null;index"`
	EventType EventType   `gorm:"type:varchar(50);not null;index"`
	Status    EventStatus `gorm:"type:varchar(50);not null;default:'pending';index"`
	BookingID *uuid.UUID  `gorm:"type:uuid"`

	// Core time fields (stored in UTC)
	StartTime time.Time `gorm:"not null;index"`
	EndTime   time.Time `gorm:"not null;index"`

	// Polymorphic details based on EventType
	ShowingDetails     *ShowingDetail     `gorm:"type:jsonb;serializer:json"`
	MaintenanceDetails *MaintenanceDetail `gorm:"type:jsonb;serializer:json"`
	BlockDetails       *BlockDetail       `gorm:"type:jsonb;serializer:json"`
	OpenHouseDetails   *OpenHouseDetail   `gorm:"type:jsonb;serializer:json"`

	// Audit fields
	CreatedBy *uuid.UUID `gorm:"type:uuid"`
	UpdatedBy *uuid.UUID `gorm:"type:uuid"`

	// Lifecycle tracking
	CompletedAt *time.Time
	ArchivedAt  *time.Time

	// Optimistic locking
	Version int `gorm:"default:0"`

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// ShowingDetail contains showing-specific information
type ShowingDetail struct {
	ProspectID    *uuid.UUID `json:"prospect_id,omitempty"`
	ProspectName  string     `json:"prospect_name"`
	ProspectEmail string     `json:"prospect_email"`
	ProspectPhone *string    `json:"prospect_phone,omitempty"`

	AgentID   *uuid.UUID `json:"agent_id,omitempty"`
	AgentName *string    `json:"agent_name,omitempty"`

	Notes    *string `json:"notes,omitempty"`
	Attended *bool   `json:"attended,omitempty"`
	Feedback *string `json:"feedback,omitempty"`
}

// MaintenanceDetail contains maintenance-specific information
type MaintenanceDetail struct {
	MaintenanceType MaintenanceType `json:"maintenance_type"`
	Title           string          `json:"title"`
	Description     *string         `json:"description,omitempty"`

	VendorID      *uuid.UUID `json:"vendor_id,omitempty"`
	VendorName    *string    `json:"vendor_name,omitempty"`
	VendorContact *string    `json:"vendor_contact,omitempty"`

	EstimatedCost *float64 `json:"estimated_cost,omitempty"`
	ActualCost    *float64 `json:"actual_cost,omitempty"`

	// If true, blocks bookings when property is vacant
	Disruptive bool `json:"disruptive"`

	// If true, requires guest notification
	RequiresGuestNotification bool `json:"requires_guest_notification"`

	WorkCompleted   *bool   `json:"work_completed,omitempty"`
	CompletionNotes *string `json:"completion_notes,omitempty"`
}

// BlockDetail contains block-specific information (owner personal use)
type BlockDetail struct {
	Reason    string  `json:"reason"`
	Notes     *string `json:"notes,omitempty"`
	OwnerStay bool    `json:"owner_stay"` // True if owner is using the property
}

// OpenHouseDetail contains open house-specific information
type OpenHouseDetail struct {
	Title       string  `json:"title"`
	Description *string `json:"description,omitempty"`

	MaxAttendees int `json:"max_attendees"`

	AgentID   *uuid.UUID `json:"agent_id,omitempty"`
	AgentName *string    `json:"agent_name,omitempty"`

	// Attendee tracking
	Attendees []Attendee `json:"attendees,omitempty"`
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
	ID        uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	ListingID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex"`

	// Buffer time between bookings (in hours)
	BufferHours int `gorm:"default:2"`

	// Lead time required before booking (in hours)
	LeadTimeHours int `gorm:"default:24"`

	// Timezone for the listing (IANA timezone, e.g., "Africa/Lagos")
	Timezone string `gorm:"type:varchar(100);default:'Africa/Lagos'"`

	// How far ahead guests can book (in months)
	BookingWindowMonths int `gorm:"default:12"`

	// Allow instant booking
	InstantBooking bool `gorm:"default:false"`

	// Allow same-day bookings
	SameDayBooking bool `gorm:"default:false"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

// RecurringEventPattern stores recurring event configurations
type RecurringEventPattern struct {
	ID        uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	ListingID uuid.UUID `gorm:"type:uuid;not null;index"`
	EventType EventType `gorm:"type:varchar(50);not null"`

	// Pattern configuration
	Frequency  string  `gorm:"type:varchar(50);not null"` // "daily", "weekly", "monthly"
	Interval   int     `gorm:"default:1"`                 // Every N days/weeks/months
	DaysOfWeek *string `gorm:"type:varchar(100)"`         // Comma-separated: "1,3,5" for Mon,Wed,Fri
	DayOfMonth *int    `gorm:"type:int"`                  // Day of month for monthly patterns

	// Time of day
	StartTimeOfDay string `gorm:"type:varchar(10);not null"` // "09:00"
	EndTimeOfDay   string `gorm:"type:varchar(10);not null"` // "17:00"

	// Pattern validity
	StartDate time.Time `gorm:"not null"`
	EndDate   *time.Time

	// Exceptions (dates to skip)
	Exceptions []time.Time `gorm:"type:jsonb;serializer:json"`

	// Event details (polymorphic based on EventType)
	MaintenanceDetails *MaintenanceDetail `gorm:"type:jsonb;serializer:json"`
	OpenHouseDetails   *OpenHouseDetail   `gorm:"type:jsonb;serializer:json"`

	Active bool `gorm:"default:true"`

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// --- GORM Hooks ---

// BeforeSave validates CalendarEvent before saving
func (e *CalendarEvent) BeforeSave(tx *gorm.DB) error {
	changed := func(field string) bool {
		return tx != nil && tx.Statement != nil && tx.Statement.Changed(field)
	}

	// 1. Validate EventType
	if changed("event_type") {
		switch e.EventType {
		case EventTypeBooking, EventTypeShowing, EventTypeMaintenance, EventTypeBlock, EventTypeOpenHouse:
		default:
			return ErrInvalidEventType
		}
	}

	// 2. Validate EventStatus
	if changed("status") {
		switch e.Status {
		case EventStatusPending, EventStatusConfirmed, EventStatusInProgress,
			EventStatusCompleted, EventStatusCancelled, EventStatusNoShow:
		default:
			return ErrInvalidEventStatus
		}
	}

	// 3. Validate time range
	if changed("start_time") || changed("end_time") {
		if !e.EndTime.After(e.StartTime) {
			return ErrEventEndBeforeStart
		}
	}

	// 4. Validate ListingID
	if e.ListingID == uuid.Nil {
		return ErrListingIDRequired
	}

	// 5. Validate polymorphic details based on EventType
	if e.MaintenanceDetails != nil {
		switch e.MaintenanceDetails.MaintenanceType {
		case MaintenanceRoutine, MaintenanceEmergency, MaintenanceRepair, MaintenanceRenovation:
		default:
			return ErrInvalidMaintenanceType
		}
	}

	if e.OpenHouseDetails != nil {
		if e.OpenHouseDetails.MaxAttendees <= 0 {
			return ErrInvalidMaxAttendees
		}
	}

	return nil
}

// BeforeSave validates CalendarConfig before saving
func (c *CalendarConfig) BeforeSave(tx *gorm.DB) error {
	if c.BufferHours < 0 {
		return ErrInvalidBufferHours
	}

	if c.LeadTimeHours < 0 {
		return errors.New("lead time hours must be non-negative")
	}

	if c.BookingWindowMonths <= 0 {
		return errors.New("booking window months must be positive")
	}

	if c.ListingID == uuid.Nil {
		return ErrListingIDRequired
	}

	return nil
}

// BeforeSave validates RecurringEventPattern before saving
func (r *RecurringEventPattern) BeforeSave(tx *gorm.DB) error {
	changed := func(field string) bool {
		return tx != nil && tx.Statement != nil && tx.Statement.Changed(field)
	}

	// Validate EventType
	if changed("event_type") {
		switch r.EventType {
		case EventTypeMaintenance, EventTypeOpenHouse, EventTypeBlock:
			// Only these types support recurring patterns
		default:
			return fmt.Errorf("event type %s does not support recurring patterns", r.EventType)
		}
	}

	// Validate Frequency
	if changed("frequency") {
		switch r.Frequency {
		case "daily", "weekly", "monthly":
		default:
			return fmt.Errorf("invalid frequency: %s", r.Frequency)
		}
	}

	// Validate Interval
	if r.Interval <= 0 {
		return errors.New("interval must be positive")
	}

	if r.ListingID == uuid.Nil {
		return ErrListingIDRequired
	}

	return nil
}
