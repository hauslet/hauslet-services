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
		Order("check_in_time DESC").
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
		Order("check_in_time DESC").
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
// Criteria: status=completed, event time + payoutWindowHours has passed, has payment
// escrowReleaseEvent determines which timestamp to use: "checkin_confirmed" uses check_in, "checkout_confirmed" uses check_out
func (r *BookingRepositoryImpl) FindBookingsReadyForPayout(ctx context.Context, escrowReleaseEvent string, payoutWindowHours int, limit int) ([]*BookingPayoutInfo, error) {
	var rows []bookingPayoutRow

	// Calculate the cutoff time (now - payoutWindowHours)
	cutoffTime := time.Now().Add(-time.Duration(payoutWindowHours) * time.Hour)

	// Determine which field to use based on escrow release event
	timeField := "bookings.check_out" // Default to checkout
	if escrowReleaseEvent == "checkin_confirmed" {
		timeField = "bookings.check_in"
	}

	// Join with listings table to get the owner_id (host)
	// Select only the fields needed for payout processing
	query := r.db.WithContext(ctx).
		Table("bookings").
		Select("bookings.id, bookings.total_price, bookings.currency, bookings.last_payment_id, listings.owner_id as host_id").
		Joins("JOIN listings ON listings.id = bookings.listing_id").
		Where("bookings.status = ?", schema.BookingStatusCompleted).
		Where(timeField+" < ?", cutoffTime).
		Where("bookings.last_payment_id IS NOT NULL").
		Order(timeField + " ASC").
		Limit(limit)

	if err := query.Scan(&rows).Error; err != nil {
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

// FindBookingsReadyForCompletion finds active bookings ready to be marked as completed
// Criteria: status=active, event time + escrowReleaseHours has passed
// escrowReleaseEvent determines which timestamp to use: "checkin_confirmed" uses check_in, "checkout_confirmed" uses check_out
func (r *BookingRepositoryImpl) FindBookingsReadyForCompletion(
	ctx context.Context,
	escrowReleaseEvent string,
	escrowReleaseHours int,
	limit int,
) ([]*schema.Booking, error) {
	// Calculate the cutoff time (now - escrowReleaseHours)
	cutoffTime := time.Now().Add(-time.Duration(escrowReleaseHours) * time.Hour)

	// Determine which field to use based on escrow release event
	timeField := "check_out" // Default to checkout
	if escrowReleaseEvent == "checkin_confirmed" {
		timeField = "check_in"
	}

	var bookings []*schema.Booking

	// Query active bookings where the event time has passed
	query := r.db.WithContext(ctx).
		Where("status = ?", schema.BookingStatusActive).
		Where(timeField+" < ?", cutoffTime).
		Order(timeField + " ASC").
		Limit(limit)

	if err := query.Find(&bookings).Error; err != nil {
		return nil, err
	}

	return bookings, nil
}

// FindBookingsPendingCheckIn returns bookings past scheduled check-in without actual check-in recorded
func (r *BookingRepositoryImpl) FindBookingsPendingCheckIn(ctx context.Context, cutoff time.Time, limit int) ([]*schema.Booking, error) {
	var bookings []*schema.Booking
	query := r.db.WithContext(ctx).
		Where("check_in IS NULL").
		Where("check_in_time IS NOT NULL").
		Where("check_in_time <= ?", cutoff).
		Where("status IN ?", []schema.BookingStatus{
			schema.BookingStatusConfirmed,
			schema.BookingStatusActive,
		}).
		Order("check_in_time ASC").
		Limit(limit)

	if err := query.Find(&bookings).Error; err != nil {
		return nil, err
	}

	return bookings, nil
}

// FindBookingsPendingCheckOut returns bookings past scheduled check-out without actual check-out recorded
func (r *BookingRepositoryImpl) FindBookingsPendingCheckOut(ctx context.Context, cutoff time.Time, limit int) ([]*schema.Booking, error) {
	var bookings []*schema.Booking
	query := r.db.WithContext(ctx).
		Where("check_out IS NULL").
		Where("check_out_time IS NOT NULL").
		Where("check_out_time <= ?", cutoff).
		Where("status IN ?", []schema.BookingStatus{
			schema.BookingStatusActive,
		}).
		Order("check_out_time ASC").
		Limit(limit)

	if err := query.Find(&bookings).Error; err != nil {
		return nil, err
	}

	return bookings, nil
}

// FindCompletedBookingsInRange finds bookings completed within a specific time range
// Used by the review reminder system to find bookings that need review invites/reminders
func (r *BookingRepositoryImpl) FindCompletedBookingsInRange(
	ctx context.Context,
	startTime, endTime time.Time,
	limit int,
) ([]*schema.Booking, error) {
	var bookings []*schema.Booking

	// Query completed bookings within the time range
	query := r.db.WithContext(ctx).
		Where("status = ?", schema.BookingStatusCompleted).
		Where("completed_at IS NOT NULL").
		Where("completed_at >= ?", startTime).
		Where("completed_at <= ?", endTime).
		Order("completed_at ASC")

	// Apply limit if specified (0 = no limit)
	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Find(&bookings).Error; err != nil {
		return nil, err
	}

	return bookings, nil
}
