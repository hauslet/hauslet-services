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
	// FindBookingsReadyForPayout finds completed bookings ready for host payout (checkout + payoutWindowHours passed, not settled, has payment)
	FindBookingsReadyForPayout(ctx context.Context, payoutWindowHours int, limit int) ([]*BookingPayoutInfo, error)
}

type BookingRepositoryImpl struct {
	db *gorm.DB
}

func NewBookingRepository(db *gorm.DB) BookingRepository {
	return &BookingRepositoryImpl{db: db}
}
