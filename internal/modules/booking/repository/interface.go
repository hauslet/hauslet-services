package repository

import (
	"context"
	"hauslet/internal/modules/booking/repository/schema"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// BookingPayoutInfo contains minimal booking data needed for payout processing
type BookingPayoutInfo struct {
	ID            uuid.UUID
	HostID        uuid.UUID
	TotalPrice    float64
	Currency      string
	LastPaymentID uuid.UUID
}

type BookingRepository interface {
	CreateBooking(ctx context.Context, booking *schema.Booking) error
	GetBookingByID(ctx context.Context, id uuid.UUID) (*schema.Booking, error)
	UpdateBooking(ctx context.Context, booking *schema.Booking) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status schema.BookingStatus, confirmedAt, cancelledAt *time.Time) error
	ListBookingsForGuest(ctx context.Context, guestID uuid.UUID, limit, offset int) ([]*schema.Booking, error)
	ListBookingsForListing(ctx context.Context, listingID uuid.UUID, limit, offset int) ([]*schema.Booking, error)
	FindExpiredHolds(ctx context.Context, expiredBefore time.Time) ([]*schema.Booking, error)

	// FindBookingsReadyForPayout finds completed bookings ready for host payout
	// escrowReleaseEvent: "checkin_confirmed" or "checkout_confirmed" - determines which timestamp to use
	// payoutWindowHours: hours after the event before payout is available
	FindBookingsReadyForPayout(ctx context.Context, escrowReleaseEvent string, payoutWindowHours int, limit int) ([]*BookingPayoutInfo, error)

	// FindBookingsReadyForCompletion finds active bookings ready to be marked as completed
	// escrowReleaseEvent: "checkin_confirmed" or "checkout_confirmed" - determines which timestamp to use
	// escrowReleaseHours: hours after the event before booking is considered completed
	FindBookingsReadyForCompletion(ctx context.Context, escrowReleaseEvent string, escrowReleaseHours int, limit int) ([]*schema.Booking, error)

	// FindCompletedBookingsInRange finds bookings completed within a specific time range
	// Used for review reminder system to find bookings that need review invites
	// limit: maximum number of bookings to return (0 = no limit, but not recommended for production)
	FindCompletedBookingsInRange(ctx context.Context, startTime, endTime time.Time, limit int) ([]*schema.Booking, error)
}

type BookingRepositoryImpl struct {
	db *gorm.DB
}

func NewBookingRepository(db *gorm.DB) BookingRepository {
	return &BookingRepositoryImpl{db: db}
}
