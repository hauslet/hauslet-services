package service

import (
	"context"
	"fmt"
	"hauslet/internal/modules/booking/domain"
	"time"

	"github.com/google/uuid"
)

// QuoteBooking returns pricing and availability information without creating a booking.
// This allows users to check prices and availability before committing to a reservation.
func (s *BookingServiceImpl) QuoteBooking(
	ctx context.Context,
	listingID uuid.UUID,
	checkIn, checkOut time.Time,
	guestCount int,
) (*domain.BookingQuote, error) {
	if s.log != nil {
		s.log.Info(" quoting booking listing=%s checkIn=%s checkOut=%s guests=%d",
			listingID, checkIn.Format("2006-01-02"), checkOut.Format("2006-01-02"), guestCount)
	}

	// Validate guest count
	if guestCount < 1 {
		return nil, fmt.Errorf("guest count must be at least 1")
	}

	// Get listing constraints
	constraints, err := s.listingHooks.GetListingConstraints(ctx, listingID)
	if err != nil {
		return nil, err
	}

	quote := &domain.BookingQuote{
		ListingID:    listingID,
		CheckIn:      checkIn,
		CheckOut:     checkOut,
		GuestCount:   guestCount,
		Currency:     constraints.Currency,
		MinNights:    constraints.MinNights,
		MaxNights:    constraints.MaxNights,
		MaxGuests:    constraints.MaxGuests,
		CheckInTime:  constraints.CheckInTime,
		CheckOutTime: constraints.CheckOutTime,
	}

	// Validate constraints
	if err := s.validateBookingConstraints(checkIn, checkOut, guestCount, constraints); err != nil {
		quote.Available = false
		reason := err.Error()
		quote.UnavailabilityReason = &reason
		return quote, nil // Return quote with unavailable status, not an error
	}

	// Check calendar availability
	availability, err := s.calendar.CheckAvailability(ctx, listingID, checkIn, checkOut)
	if err != nil {
		return nil, err
	}

	if availability == nil || !availability.Available {
		quote.Available = false
		reason := "Dates not available"
		quote.UnavailabilityReason = &reason
		return quote, nil
	}

	// Determine booking type and response window
	hoursUntilCheckIn := time.Until(checkIn).Hours()
	autoAccept := constraints.AutoAcceptBookings

	// Get calendar config to check instant booking setting
	if calendarConfig, err := s.calendar.GetCalendarConfig(ctx, listingID); err == nil && calendarConfig != nil {
		autoAccept = calendarConfig.InstantBooking
	}

	quote.InstantBooking = autoAccept

	// Calculate response window for manual approval
	if !autoAccept && hoursUntilCheckIn >= 6 {
		responseWindow := s.calculateResponseWindow(hoursUntilCheckIn)
		quote.ResponseWindowHours = responseWindow.Hours()
	}

	// Calculate pricing
	if s.pricing != nil {
		priceBreakdown, err := s.pricing.CalculatePrice(ctx, listingID, checkIn, checkOut, guestCount)
		if err != nil {
			// Fallback to base price if pricing calculation fails
			if s.log != nil {
				s.log.Warn("pricing calculation failed for quote: %v", err)
			}
			baseRate, baseCurrency, baseErr := s.pricing.GetBasePrice(ctx, listingID)
			if baseErr == nil {
				nights := int(checkOut.Sub(checkIn).Hours() / 24)
				quote.TotalPrice = baseRate * float64(nights)
				quote.Currency = baseCurrency
			}
		} else {
			quote.PriceBreakdown = domain.NewPriceBreakdownSnapshotFromPricing(priceBreakdown)
			quote.TotalPrice = priceBreakdown.Total
			quote.Currency = priceBreakdown.Currency
		}
	}

	quote.Available = true

	if s.log != nil {
		s.log.Info(" quote generated: listing=%s available=%t instant=%t total=%.2f %s",
			listingID, quote.Available, quote.InstantBooking, quote.TotalPrice, quote.Currency)
	}

	return quote, nil
}
