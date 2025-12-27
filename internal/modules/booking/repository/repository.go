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

// bookingPayoutRow is used for scanning the joined query result
type bookingPayoutRow struct {
	ID            uuid.UUID  `gorm:"column:id"`
	HostID        uuid.UUID  `gorm:"column:host_id"`
	TotalPrice    float64    `gorm:"column:total_price"`
	Currency      string     `gorm:"column:currency"`
	LastPaymentID *uuid.UUID `gorm:"column:last_payment_id"`
}

// FindBookingsReadyForPayout finds completed bookings ready for host payout
// Criteria: status=completed, checkout + payoutWindowHours has passed, not settled, has payment
func (r *BookingRepositoryImpl) FindBookingsReadyForPayout(ctx context.Context, payoutWindowHours int, limit int) ([]*BookingPayoutInfo, error) {
	var rows []bookingPayoutRow

	// Calculate the cutoff time (now - payoutWindowHours)
	cutoffTime := time.Now().Add(-time.Duration(payoutWindowHours) * time.Hour)

	// Join with listings table to get the owner_id (host)
	// Select only the fields needed for payout processing
	if err := r.db.WithContext(ctx).
		Table("bookings").
		Select("bookings.id, bookings.total_price, bookings.currency, bookings.last_payment_id, listings.owner_id as host_id").
		Joins("JOIN listings ON listings.id = bookings.listing_id").
		Where("bookings.status = ?", schema.BookingStatusCompleted).
		Where("bookings.check_out < ?", cutoffTime).
		Where("bookings.last_payment_id IS NOT NULL").
		Order("bookings.check_out ASC").
		Limit(limit).
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	// Convert to BookingPayoutInfo
	results := make([]*BookingPayoutInfo, 0, len(rows))
	for _, row := range rows {
		// Skip if last_payment_id is nil (shouldn't happen due to WHERE clause, but be safe)
		if row.LastPaymentID == nil {
			continue
		}

		results = append(results, &BookingPayoutInfo{
			ID:            row.ID,
			HostID:        row.HostID,
			TotalPrice:    row.TotalPrice,
			Currency:      row.Currency,
			LastPaymentID: *row.LastPaymentID,
		})
	}

	return results, nil
}
