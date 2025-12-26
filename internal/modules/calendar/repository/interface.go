package repository

import (
	"context"
	"hauslet/internal/modules/calendar/repository/schema"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CalendarRepository interface {
	// --- Core Event CRUD ---
	CreateEvent(ctx context.Context, event *schema.CalendarEvent) error
	GetEventByID(ctx context.Context, id uuid.UUID) (*schema.CalendarEvent, error)
	UpdateEvent(ctx context.Context, event *schema.CalendarEvent) error
	DeleteEvent(ctx context.Context, id uuid.UUID) error // Soft delete
	HardDeleteEvent(ctx context.Context, id uuid.UUID) error

	// --- Event Queries ---
	// GetEventsForListing retrieves all events for a listing within a date range
	GetEventsForListing(ctx context.Context, listingID uuid.UUID, startTime, endTime time.Time, eventTypes []schema.EventType) ([]*schema.CalendarEvent, error)

	// GetUpcomingEvents retrieves upcoming events for a listing
	GetUpcomingEvents(ctx context.Context, listingID uuid.UUID, limit int) ([]*schema.CalendarEvent, error)

	// GetEventsByStatus retrieves events by status for a listing
	GetEventsByStatus(ctx context.Context, listingID uuid.UUID, status schema.EventStatus, limit, offset int) ([]*schema.CalendarEvent, error)

	// GetEventsByType retrieves events by type for a listing
	GetEventsByType(ctx context.Context, listingID uuid.UUID, eventType schema.EventType, limit, offset int) ([]*schema.CalendarEvent, error)

	// GetEventsForOwner retrieves all events for properties owned by a user
	GetEventsForOwner(ctx context.Context, ownerID uuid.UUID, startTime, endTime time.Time) ([]*schema.CalendarEvent, error)

	// --- Availability Checks ---
	// CheckAvailability checks if a listing is available for the given time range
	CheckAvailability(ctx context.Context, listingID uuid.UUID, startTime, endTime time.Time) (bool, error)

	// GetConflictingEvents returns events that conflict with the given time range
	GetConflictingEvents(ctx context.Context, listingID uuid.UUID, startTime, endTime time.Time, excludeEventID *uuid.UUID) ([]*schema.CalendarEvent, error)

	// HasConflictingBooking checks if there's a conflicting confirmed booking
	HasConflictingBooking(ctx context.Context, listingID uuid.UUID, startTime, endTime time.Time, excludeEventID *uuid.UUID) (bool, error)

	// --- Event Status Management ---
	// UpdateEventStatus updates the status of an event with optimistic locking
	UpdateEventStatus(ctx context.Context, eventID uuid.UUID, newStatus schema.EventStatus, version int) error

	// MarkEventCompleted marks an event as completed
	MarkEventCompleted(ctx context.Context, eventID uuid.UUID) error

	// MarkEventsCompletedByEndTime marks events as completed if their end time has passed
	MarkEventsCompletedByEndTime(ctx context.Context, beforeTime time.Time) (int64, error)

	// --- Calendar Config ---
	CreateCalendarConfig(ctx context.Context, config *schema.CalendarConfig) error
	GetCalendarConfig(ctx context.Context, listingID uuid.UUID) (*schema.CalendarConfig, error)
	UpdateCalendarConfig(ctx context.Context, config *schema.CalendarConfig) error
	DeleteCalendarConfig(ctx context.Context, listingID uuid.UUID) error
	ConfigExists(ctx context.Context, listingID uuid.UUID) (bool, error)

	// --- Recurring Event Patterns ---
	CreateRecurringPattern(ctx context.Context, pattern *schema.RecurringEventPattern) error
	GetRecurringPattern(ctx context.Context, id uuid.UUID) (*schema.RecurringEventPattern, error)
	GetRecurringPatternsForListing(ctx context.Context, listingID uuid.UUID) ([]*schema.RecurringEventPattern, error)
	UpdateRecurringPattern(ctx context.Context, pattern *schema.RecurringEventPattern) error
	DeleteRecurringPattern(ctx context.Context, id uuid.UUID) error // Soft delete

	// --- Archival ---
	// ArchiveCompletedEvents soft deletes events completed before the given date
	ArchiveCompletedEvents(ctx context.Context, completedBefore time.Time) (int64, error)

	// --- Transactions ---
	Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error
}

type CalendarRepositoryImpl struct {
	db *gorm.DB
}

func NewCalendarRepository(db *gorm.DB) CalendarRepository {
	return &CalendarRepositoryImpl{db: db}
}
