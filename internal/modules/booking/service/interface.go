package service

import (
	"context"
	"hauslet/config"
	"hauslet/internal/modules/booking/domain"
	"hauslet/internal/modules/booking/notification"
	"hauslet/internal/modules/booking/repository"
	calendardomain "hauslet/internal/modules/calendar/domain"
	pricingdomain "hauslet/internal/modules/pricing/domain"
	platformQueue "hauslet/internal/platform/queue"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// FinanceHooks defines callbacks to finance module for booking-related financial events
type FinanceHooks interface {
	OnPaymentSucceeded(ctx context.Context, bookingID, paymentID uuid.UUID, amount int64, currency string) error
	OnRefundProcessed(ctx context.Context, bookingID, paymentID uuid.UUID, amount int64, currency string) error
	OnBookingCompleted(ctx context.Context, bookingID, hostID uuid.UUID) error
}

// ReviewHooks defines callbacks to review module for review-related notifications.
type ReviewHooks interface {
	SendReviewInvites(ctx context.Context, bookingID uuid.UUID) error
}

type BookingService interface {
	// Quote and pricing
	QuoteBooking(ctx context.Context, listingID uuid.UUID, checkIn, checkOut time.Time, guestCount int) (*domain.BookingQuote, error)

	// Booking creation and management
	ReserveBooking(ctx context.Context, listingID uuid.UUID, guestID uuid.UUID, checkIn, checkOut time.Time, guestCount int, paymentMethodID *uuid.UUID, specialRequests *string) (*domain.Booking, *PaymentResult, error)
	RequestBooking(ctx context.Context, listingID uuid.UUID, guestID uuid.UUID, checkIn, checkOut time.Time, guestCount int, specialRequests *string) (*domain.Booking, error)
	PayForBooking(ctx context.Context, bookingID uuid.UUID, actorID uuid.UUID, paymentMethodID *uuid.UUID) (*domain.Booking, *PaymentResult, error)
	ConfirmBooking(ctx context.Context, bookingID uuid.UUID, actorID uuid.UUID) (*domain.Booking, error)
	CancelBooking(ctx context.Context, bookingID uuid.UUID, actorID uuid.UUID, reason *string) (*domain.Booking, error)
	GetBooking(ctx context.Context, bookingID uuid.UUID, requestorID uuid.UUID) (*domain.Booking, error)
	GetBookingsByIDs(ctx context.Context, bookingIDs []uuid.UUID) ([]*domain.Booking, error)
	GetBookingByReference(ctx context.Context, reference string, requestorID uuid.UUID) (*domain.Booking, error)
	CheckInBooking(ctx context.Context, bookingID uuid.UUID, actorID uuid.UUID) (*domain.Booking, error)
	CheckOutBooking(ctx context.Context, bookingID uuid.UUID, actorID uuid.UUID) (*domain.Booking, error)

	// List methods
	ListBookingsForGuest(ctx context.Context, guestID uuid.UUID, limit, offset int) ([]*domain.Booking, error)
	ListBookingsForListing(ctx context.Context, listingID uuid.UUID, requestorID uuid.UUID, status *domain.BookingStatus, limit, offset int) ([]*domain.Booking, error)

	// Payment lifecycle methods
	HandlePaymentSuccess(ctx context.Context, bookingID uuid.UUID, paymentID uuid.UUID) error
	HandlePaymentFailure(ctx context.Context, bookingID uuid.UUID, paymentID uuid.UUID, reason string) error
	HandlePaymentRefund(ctx context.Context, bookingID uuid.UUID, paymentID uuid.UUID, refundedAmount int64) error
	ArchiveExpiredBookings(ctx context.Context, expiredBefore time.Time) ([]uuid.UUID, error)

	// Payout lifecycle methods
	MarkAsSettled(ctx context.Context, bookingID uuid.UUID) error

	// Completion lifecycle methods
	CompleteBookings(ctx context.Context) error

	// Check-in/out fallback
	AutoPopulateCheckInOut(ctx context.Context) (int, int, error)
}

type ContactInfo struct {
	ID           uuid.UUID
	Name         string
	Email        string
	Phone        *string
	PhotoURL     *string
	IsIDVerified bool
	ReviewsCount int
	Rating       float64
}

type CalendarGateway interface {
	CheckAvailability(ctx context.Context, listingID uuid.UUID, startTime, endTime time.Time) (*calendardomain.AvailabilityResult, error)
	CreateEvent(ctx context.Context, event *calendardomain.CalendarEvent) (*calendardomain.CalendarEvent, error)
	GetCalendarConfig(ctx context.Context, listingID uuid.UUID) (*calendardomain.CalendarConfig, error)
	GetEvent(ctx context.Context, eventID uuid.UUID, requestorID uuid.UUID) (*calendardomain.CalendarEvent, error)
	UpdateEvent(ctx context.Context, event *calendardomain.CalendarEvent, requestorID uuid.UUID) (*calendardomain.CalendarEvent, error)
	CancelEvent(ctx context.Context, eventID uuid.UUID, requestorID uuid.UUID) error
	DeleteEvent(ctx context.Context, eventID uuid.UUID, requestorID uuid.UUID) error
}

type PricingService interface {
	CalculatePrice(ctx context.Context, listingID uuid.UUID, checkIn, checkOut time.Time, guestCount int) (*pricingdomain.PriceBreakdown, error)
	GetBasePrice(ctx context.Context, listingID uuid.UUID) (float64, string, error)
	CalculateRefund(ctx context.Context, input pricingdomain.RefundCalculationInput) (*pricingdomain.RefundBreakdown, error)
}

type PaymentGateway interface {
	InitiatePayment(ctx context.Context, input PaymentInput) (*PaymentResult, error)
	VerifyPayment(ctx context.Context, paymentID uuid.UUID) (*PaymentStatus, error)
	RefundPayment(ctx context.Context, input RefundPaymentInput) (*RefundResult, error)
	GetDefaultPaymentMethodID(ctx context.Context, userID uuid.UUID) (*uuid.UUID, error)
}

type PaymentInput struct {
	BookingID       uuid.UUID
	Amount          int64
	Currency        string
	PayerID         uuid.UUID
	PayerEmail      string
	PayerName       string
	PaymentMethodID *uuid.UUID
	CallbackURL     string
	Description     string
}

type PaymentResult struct {
	PaymentID        uuid.UUID
	Status           string
	AuthorizationURL *string
	Reference        string
}

type PaymentStatus struct {
	PaymentID uuid.UUID
	Status    string
	Amount    int64
}

type RefundPaymentInput struct {
	PaymentID  uuid.UUID
	Amount     *int64 // Partial refund amount (nil for full refund)
	Reason     string
	RefundedBy uuid.UUID
}

type RefundResult struct {
	RefundID    string
	Status      string
	Amount      int64
	RefundedAt  time.Time
	PaymentID   uuid.UUID
	BookingID   *uuid.UUID
	RefundedBy  uuid.UUID
	ProcessedAt time.Time
}

type ListingHooks interface {
	GetListingConstraints(ctx context.Context, listingID uuid.UUID) (*ListingConstraints, error)
	GetListingOwner(ctx context.Context, listingID uuid.UUID) (uuid.UUID, error)
}

type ProfileProvider interface {
	GetUserContact(ctx context.Context, userID uuid.UUID) (*ContactInfo, error)
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

type ListingConstraints struct {
	ListingID          uuid.UUID
	MinNights          int
	MaxNights          *int
	MaxGuests          int
	CheckInTime        *string
	CheckOutTime       *string
	Currency           string
	Timezone           string
	AutoAcceptBookings bool
	RefundPolicy       string

	// New pricing and constraint fields
	Fees            []CustomFee
	Discounts       []Discount
	BookingSettings *BookingSettings
	AdvanceBooking  *AdvanceBooking
}

type BookingServiceImpl struct {
	repo           repository.BookingRepository
	calendar       CalendarGateway
	pricing        PricingService
	payment        PaymentGateway
	listingHooks   ListingHooks
	profiles       ProfileProvider
	notifier       *notification.NotificationService
	refundQueue    *platformQueue.Client
	refundSubject  string
	financeHooks   FinanceHooks
	reviewHooks    ReviewHooks
	platformConfig config.PlatformYAMLConfig
	log            *slog.Logger
}

func NewBookingService(
	repo repository.BookingRepository,
	calendar CalendarGateway,
	pricing PricingService,
	payment PaymentGateway,
	listingHooks ListingHooks,
	profiles ProfileProvider,
	notifier *notification.NotificationService,
	refundQueue *platformQueue.Client,
	refundSubject string,
	financeHooks FinanceHooks,
	reviewHooks ReviewHooks,
	platformConfig config.PlatformYAMLConfig,
	log *slog.Logger,
) BookingService {
	return &BookingServiceImpl{
		repo:           repo,
		calendar:       calendar,
		pricing:        pricing,
		payment:        payment,
		listingHooks:   listingHooks,
		profiles:       profiles,
		notifier:       notifier,
		refundQueue:    refundQueue,
		refundSubject:  refundSubject,
		financeHooks:   financeHooks,
		reviewHooks:    reviewHooks,
		platformConfig: platformConfig,
		log:            log,
	}
}
