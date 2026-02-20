package hooks

import (
	"context"
	"time"

	bookingservice "hauslet/internal/modules/booking/service"
	discoveryservice "hauslet/internal/modules/discovery/service"

	"github.com/google/uuid"
)

// BookingDiscoveryAdapter wraps BookingService to implement BookingDiscoveryHooks.
type BookingDiscoveryAdapter struct {
	bookingSvc bookingservice.BookingService
}

// NewBookingDiscoveryAdapter creates a new BookingDiscoveryAdapter.
func NewBookingDiscoveryAdapter(bookingSvc bookingservice.BookingService) discoveryservice.BookingDiscoveryHooks {
	return &BookingDiscoveryAdapter{
		bookingSvc: bookingSvc,
	}
}

// QuoteBooking delegates to booking quote flow for availability-aware pricing.
func (a *BookingDiscoveryAdapter) QuoteBooking(
	ctx context.Context,
	listingID uuid.UUID,
	checkIn, checkOut time.Time,
	guestCount int,
) (*discoveryservice.ShortletQuote, error) {
	quote, err := a.bookingSvc.QuoteBooking(ctx, listingID, checkIn, checkOut, guestCount)
	if err != nil {
		return nil, err
	}
	if quote == nil {
		return nil, nil
	}

	return &discoveryservice.ShortletQuote{
		Available:  quote.Available,
		CheckIn:    quote.CheckIn,
		CheckOut:   quote.CheckOut,
		TotalPrice: quote.TotalPrice,
		Currency:   quote.Currency,
	}, nil
}
