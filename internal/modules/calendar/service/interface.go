package service

import (
	"context"
	"hauslet/internal/modules/calendar/domain"
	"hauslet/internal/modules/calendar/repository"
	"hauslet/internal/platform/redis"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

type CalendarService interface {
	// --- Event Lifecycle ---
	CreateEvent(ctx context.Context, event *domain.CalendarEvent) (*domain.CalendarEvent, error)
	GetEvent(ctx context.Context, eventID uuid.UUID, requestorID uuid.UUID) (*domain.CalendarEvent, error)
	UpdateEvent(ctx context.Context, event *domain.CalendarEvent, requestorID uuid.UUID) (*domain.CalendarEvent, error)
	CancelEvent(ctx context.Context, eventID uuid.UUID, requestorID uuid.UUID) error
	DeleteEvent(ctx context.Context, eventID uuid.UUID, requestorID uuid.UUID) error

	// --- Availability Checks ---
	CheckAvailability(ctx context.Context, listingID uuid.UUID, startTime, endTime time.Time) (*domain.AvailabilityResult, error)
	GetConflictingEvents(ctx context.Context, listingID uuid.UUID, startTime, endTime time.Time) ([]*domain.CalendarEvent, error)

	// --- Event Queries ---
	GetEventsForListing(ctx context.Context, listingID uuid.UUID, startTime, endTime time.Time, eventTypes []domain.EventType) ([]*domain.CalendarEvent, error)
	GetUpcomingEvents(ctx context.Context, listingID uuid.UUID, limit int) ([]*domain.CalendarEvent, error)
	GetEventsByStatus(ctx context.Context, listingID uuid.UUID, status domain.EventStatus, limit, offset int) ([]*domain.CalendarEvent, error)
	GetEventsForOwner(ctx context.Context, ownerID uuid.UUID, startTime, endTime time.Time) ([]*domain.CalendarEvent, error)

	// --- Showing Management ---
	ScheduleShowing(ctx context.Context, listingID uuid.UUID, showingTime time.Time, duration time.Duration, details *domain.ShowingDetail) (*domain.CalendarEvent, error)
	UpdateShowing(ctx context.Context, eventID uuid.UUID, updates *domain.ShowingDetail) (*domain.CalendarEvent, error)
	MarkShowingAttendance(ctx context.Context, eventID uuid.UUID, attended bool, feedback *string) error

	// --- Maintenance Management ---
	ScheduleMaintenance(ctx context.Context, listingID uuid.UUID, startTime, endTime time.Time, details *domain.MaintenanceDetail) (*domain.CalendarEvent, error)
	CompleteMaintenance(ctx context.Context, eventID uuid.UUID, actualCost *float64, notes *string) error

	// --- Block Management ---
	BlockDates(ctx context.Context, listingID uuid.UUID, startTime, endTime time.Time, reason string, ownerID uuid.UUID, ownerStay bool) (*domain.CalendarEvent, error)
	UnblockDates(ctx context.Context, eventID uuid.UUID, ownerID uuid.UUID) error

	// --- Open House Management ---
	CreateOpenHouse(ctx context.Context, listingID uuid.UUID, startTime, endTime time.Time, details *domain.OpenHouseDetail) (*domain.CalendarEvent, error)
	RegisterAttendee(ctx context.Context, eventID uuid.UUID, attendee domain.Attendee) error
	MarkAttendeePresence(ctx context.Context, eventID uuid.UUID, attendeeID uuid.UUID, attended bool) error

	// --- Calendar Configuration ---
	InitializeCalendar(ctx context.Context, listingID uuid.UUID, config *domain.CalendarConfig) (*domain.CalendarConfig, error)
	GetCalendarConfig(ctx context.Context, listingID uuid.UUID) (*domain.CalendarConfig, error)
	UpdateCalendarConfig(ctx context.Context, config *domain.CalendarConfig) (*domain.CalendarConfig, error)

	// --- Recurring Events ---
	CreateRecurringPattern(ctx context.Context, pattern *domain.RecurringEventPattern) (*domain.RecurringEventPattern, error)
	GetRecurringPatternsForListing(ctx context.Context, listingID uuid.UUID) ([]*domain.RecurringEventPattern, error)
	UpdateRecurringPattern(ctx context.Context, pattern *domain.RecurringEventPattern) (*domain.RecurringEventPattern, error)
	DeleteRecurringPattern(ctx context.Context, patternID uuid.UUID, ownerID uuid.UUID) error

	// --- Maintenance Tasks ---
	// MarkCompletedEvents marks events as completed if their end time has passed
	MarkCompletedEvents(ctx context.Context) (int64, error)

	// ArchiveOldEvents soft-deletes events completed before the given date
	ArchiveOldEvents(ctx context.Context, completedBefore time.Time) (int64, error)
}

// ListingHooks interface for property module integration
type ListingHooks interface {
	// CanUseCalendar checks if a listing has calendar enabled
	CanUseCalendar(ctx context.Context, listingID uuid.UUID) (bool, error)

	// GetListingOwner returns the owner ID for a listing
	GetListingOwner(ctx context.Context, listingID uuid.UUID) (uuid.UUID, error)

	// GetListingConstraints returns booking constraints for a listing (min nights, max guests, etc.)
	GetListingConstraints(ctx context.Context, listingID uuid.UUID) (*ListingConstraints, error)

	// MarkCalendarEnabled updates the has_calendar flag in the listing
	MarkCalendarEnabled(ctx context.Context, listingID uuid.UUID, enabled bool) error
}

// ListingConstraints represents constraints from the listing/property
type ListingConstraints struct {
	ListingID    uuid.UUID
	MinNights    int
	MaxNights    *int
	MaxGuests    int
	CheckInTime  *string
	CheckOutTime *string
	Currency     string
	Timezone     string
}

type CalendarServiceImpl struct {
	repo         repository.CalendarRepository
	cache        redis.RedisClient
	listingHooks ListingHooks
	log          *slog.Logger
}

func NewCalendarService(
	repo repository.CalendarRepository,
	cache redis.RedisClient,
	listingHooks ListingHooks,
	log *slog.Logger,
) CalendarService {
	return &CalendarServiceImpl{
		repo:         repo,
		cache:        cache,
		listingHooks: listingHooks,
		log:          log,
	}
}
