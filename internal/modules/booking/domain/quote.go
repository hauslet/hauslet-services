package domain

import (
	"time"

	"github.com/google/uuid"
)

// BookingQuote represents pricing and availability information without creating a booking.
type BookingQuote struct {
	ListingID     uuid.UUID              `json:"listing_id"`
	CheckIn       time.Time              `json:"check_in"`
	CheckOut      time.Time              `json:"check_out"`
	GuestCount    int                    `json:"guest_count"`
	Available     bool                   `json:"available"`
	PriceBreakdown *PriceBreakdownSnapshot `json:"price_breakdown,omitempty"`
	TotalPrice    float64                `json:"total_price"`
	Currency      string                 `json:"currency"`

	// Booking rules and constraints
	InstantBooking bool   `json:"instant_booking"`
	MinNights      int    `json:"min_nights"`
	MaxNights      *int   `json:"max_nights,omitempty"`
	MaxGuests      int    `json:"max_guests"`
	CheckInTime    *time.Time `json:"check_in_time,omitempty"`
	CheckOutTime   *time.Time `json:"check_out_time,omitempty"`

	// Response window for manual approval (in hours)
	ResponseWindowHours float64 `json:"response_window_hours,omitempty"`

	// Availability message if not available
	UnavailabilityReason *string `json:"unavailability_reason,omitempty"`
}

// DurationNights returns the total nights for the quote.
func (q *BookingQuote) DurationNights() int {
	startDate := time.Date(q.CheckIn.Year(), q.CheckIn.Month(), q.CheckIn.Day(), 0, 0, 0, 0, q.CheckIn.Location())
	endDate := time.Date(q.CheckOut.Year(), q.CheckOut.Month(), q.CheckOut.Day(), 0, 0, 0, 0, q.CheckOut.Location())
	if endDate.Before(startDate) {
		return 0
	}
	return int(endDate.Sub(startDate).Hours() / 24)
}
