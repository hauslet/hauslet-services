package repository

import (
	"context"
	"hauslet/internal/modules/booking/repository/schema"
	"time"

	"github.com/google/uuid"
)

func (r *BookingRepositoryImpl) CreateBooking(ctx context.Context, booking *schema.Booking) error {
	return r.db.WithContext(ctx).Create(booking).Error
}

func (r *BookingRepositoryImpl) GetBookingByID(ctx context.Context, id uuid.UUID) (*schema.Booking, error) {
	var booking schema.Booking
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&booking).Error; err != nil {
		return nil, err
	}
	return &booking, nil
}

func (r *BookingRepositoryImpl) UpdateBooking(ctx context.Context, booking *schema.Booking) error {
	return r.db.WithContext(ctx).Save(booking).Error
}

func (r *BookingRepositoryImpl) UpdateStatus(ctx context.Context, id uuid.UUID, status schema.BookingStatus, confirmedAt, cancelledAt *time.Time) error {
	update := map[string]any{
		"status": status,
	}
	if confirmedAt != nil {
		update["confirmed_at"] = confirmedAt
	}
	if cancelledAt != nil {
		update["cancelled_at"] = cancelledAt
	}
	return r.db.WithContext(ctx).
		Model(&schema.Booking{}).
		Where("id = ?", id).
		Updates(update).Error
}

func (r *BookingRepositoryImpl) ListBookingsForGuest(ctx context.Context, guestID uuid.UUID, limit, offset int) ([]*schema.Booking, error) {
	var bookings []*schema.Booking
	if err := r.db.WithContext(ctx).
		Where("guest_id = ?", guestID).
		Order("check_in DESC").
		Limit(limit).
		Offset(offset).
		Find(&bookings).Error; err != nil {
		return nil, err
	}
	return bookings, nil
}

func (r *BookingRepositoryImpl) ListBookingsForListing(ctx context.Context, listingID uuid.UUID, limit, offset int) ([]*schema.Booking, error) {
	var bookings []*schema.Booking
	if err := r.db.WithContext(ctx).
		Where("listing_id = ?", listingID).
		Order("check_in DESC").
		Limit(limit).
		Offset(offset).
		Find(&bookings).Error; err != nil {
		return nil, err
	}
	return bookings, nil
}

func (r *BookingRepositoryImpl) FindExpiredHolds(ctx context.Context, expiredBefore time.Time) ([]*schema.Booking, error) {
	var bookings []*schema.Booking
	if err := r.db.WithContext(ctx).
		Where("status IN ?", []schema.BookingStatus{
			schema.BookingStatusAwaitingPayment,
			schema.BookingStatusPaymentFailed,
			schema.BookingStatusPendingApproval,
		}).
		Where("hold_expires_at IS NOT NULL").
		Where("hold_expires_at < ?", expiredBefore).
		Order("hold_expires_at ASC").
		Find(&bookings).Error; err != nil {
		return nil, err
	}
	return bookings, nil
}
