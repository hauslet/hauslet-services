package service

import (
	"context"
	"hauslet/internal/modules/calendar/domain"
	"hauslet/internal/modules/calendar/notification"
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
	GetBusyListings(ctx context.Context, startTime, endTime time.Time) ([]uuid.UUID, error)

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
	CreateOpenHouse(ctx context.Context, listingID uuid.UUID, startTime, endTime time.Time, details *domain.OpenHouseDetail, createdBy uuid.UUID) (*domain.CalendarEvent, error)
	RegisterAttendee(ctx context.Context, eventID uuid.UUID, attendee domain.Attendee) error
	MarkAttendeePresence(ctx context.Context, eventID uuid.UUID, attendeeID uuid.UUID, attended bool) error

	// --- Viewing Events (New) ---
	RequestShowing(ctx context.Context, listingID uuid.UUID, startTime, endTime time.Time, details *domain.ShowingDetail, requestorID uuid.UUID) (*domain.CalendarEvent, error)
	ConfirmShowing(ctx context.Context, eventID uuid.UUID, confirmerID uuid.UUID) error
	CancelShowing(ctx context.Context, eventID uuid.UUID, cancelReason string, cancellerID uuid.UUID) error
	RescheduleViewing(ctx context.Context, eventID uuid.UUID, newStartTime, newEndTime time.Time, reason string, requestorID uuid.UUID) (*domain.CalendarEvent, error)
	RegisterOpenHouseAttendee(ctx context.Context, eventID uuid.UUID, attendee domain.Attendee) error
	RemoveOpenHouseAttendee(ctx context.Context, eventID uuid.UUID, attendeeID uuid.UUID, removerID uuid.UUID) error

	// --- Profile Integration ---
	GetUserProfile(ctx context.Context, userID uuid.UUID) (*UserProfile, error)

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

	// GetShowingAvailability returns showing availability windows for rent/sale listings
	GetShowingAvailability(ctx context.Context, listingID uuid.UUID) ([]ShowingAvailability, error)

	// GetListingType returns the listing type (sale, rent, shortlet)
	GetListingType(ctx context.Context, listingID uuid.UUID) (string, error)
}

// ProfileHooks interface for user profile integration
type ProfileHooks interface {
	// GetUserProfile returns profile data for a user
	GetUserProfile(ctx context.Context, userID uuid.UUID) (*UserProfile, error)
}

// UserProfile represents minimal user profile data needed for calendar operations
type UserProfile struct {
	UserID       uuid.UUID
	FullName     string
	Email        string
	Phone        *string
	IsIDVerified bool
}

// CustomFee represents a fee associated with a listing
type CustomFee struct {
	Name         string
	Amount       float64
	Frequency    string // "one_time", "per_night", "per_month", "per_year"
	Category     string // "legal", "agency", "service", "caution", "other"
	IsRefundable bool
	IsOptional   bool
}

// Discount represents a price reduction
type Discount struct {
	Name       string
	Type       string // "flat" or "length_of_stay"
	Percentage float64
	MinNights  *int
	Active     bool
}

// BookingSettings defines booking configuration
type BookingSettings struct {
	ApprovalMethod       string // "instant" or "request"
	VerifiedID           bool
	PositiveReviewsOnly  bool
	ProfilePhotoRequired bool
	PreBookingMessage    string
}

// AdvanceBooking defines advance booking settings
type AdvanceBooking struct {
	MonthsAhead    int
	MinNoticeHours int
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

	// New pricing and constraint fields
	Fees            []CustomFee
	Discounts       []Discount
	BookingSettings *BookingSettings
	AdvanceBooking  *AdvanceBooking
}

// ShowingAvailability defines when viewings can be scheduled
type ShowingAvailability struct {
	DayOfWeek string // "monday", "tuesday", etc.
	StartTime string // HH:MM format (24h)
	EndTime   string // HH:MM format (24h)
	Timezone  string // IANA timezone (e.g., "Africa/Lagos")
}

type CalendarServiceImpl struct {
	repo         repository.CalendarRepository
	cache        redis.RedisClient
	listingHooks ListingHooks
	profileHooks ProfileHooks
	notifier     *notification.NotificationService
	log          *slog.Logger
}

func NewCalendarService(
	repo repository.CalendarRepository,
	cache redis.RedisClient,
	listingHooks ListingHooks,
	profileHooks ProfileHooks,
	notifier *notification.NotificationService,
	log *slog.Logger,
) CalendarService {
	return &CalendarServiceImpl{
		repo:         repo,
		cache:        cache,
		listingHooks: listingHooks,
		profileHooks: profileHooks,
		notifier:     notifier,
		log:          log,
	}
}
