package hooks

import (
	"context"
	"hauslet/internal/modules/booking/repository"
	financeService "hauslet/internal/modules/finance/service"
	"math"
)

// BookingQuerierAdapter adapts booking repository to finance service BookingQuerier interface
type BookingQuerierAdapter struct {
	bookingRepo repository.BookingRepository
}

// NewBookingQuerierAdapter creates a new booking querier adapter
func NewBookingQuerierAdapter(bookingRepo repository.BookingRepository) *BookingQuerierAdapter {
	return &BookingQuerierAdapter{
		bookingRepo: bookingRepo,
	}
}

// FindBookingsReadyForPayout queries bookings ready for payout
func (a *BookingQuerierAdapter) FindBookingsReadyForPayout(
	ctx context.Context,
	escrowReleaseEvent string,
	payoutWindowHours int,
	limit int,
) ([]*financeService.BookingForPayout, error) {
	// Query from booking repository
	bookings, err := a.bookingRepo.FindBookingsReadyForPayout(ctx, escrowReleaseEvent, payoutWindowHours, limit)
	if err != nil {
		return nil, err
	}

	// Convert to finance service format
	result := make([]*financeService.BookingForPayout, 0, len(bookings))
	for _, b := range bookings {
		// Convert TotalPrice (float64) to TotalAmount (int64 in minor currency units)
		// Multiply by 100 to convert dollars/euros to cents/kobo
		totalAmount := int64(math.Round(b.TotalPrice * 100))

		result = append(result, &financeService.BookingForPayout{
			ID:            b.ID,
			HostID:        b.HostID,
			TotalAmount:   totalAmount,
			Currency:      b.Currency,
			LastPaymentID: b.LastPaymentID,
		})
	}

	return result, nil
}
