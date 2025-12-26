package repository

import (
	"context"
	"hauslet/internal/modules/booking/repository/schema"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BookingRepository interface {
	CreateBooking(ctx context.Context, booking *schema.Booking) error
	GetBookingByID(ctx context.Context, id uuid.UUID) (*schema.Booking, error)
	UpdateBooking(ctx context.Context, booking *schema.Booking) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status schema.BookingStatus, confirmedAt, cancelledAt *time.Time) error
	ListBookingsForGuest(ctx context.Context, guestID uuid.UUID, limit, offset int) ([]*schema.Booking, error)
	ListBookingsForListing(ctx context.Context, listingID uuid.UUID, limit, offset int) ([]*schema.Booking, error)
	FindExpiredHolds(ctx context.Context, expiredBefore time.Time) ([]*schema.Booking, error)
}

type BookingRepositoryImpl struct {
	db *gorm.DB
}

func NewBookingRepository(db *gorm.DB) BookingRepository {
	return &BookingRepositoryImpl{db: db}
}
