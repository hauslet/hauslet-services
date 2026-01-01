package graphql

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"time"

	"hauslet/internal/modules/booking/domain"
	"hauslet/internal/modules/booking/service"
	"hauslet/internal/platform/xchange"
	localization "hauslet/internal/transport/middleware/localization"

	"github.com/google/uuid"
)

// Resolver handles GraphQL queries and mutations for bookings
type Resolver struct {
	bookingService service.BookingService
	log            *slog.Logger
	fx             xchange.XChange
}

// NewResolver creates a new GraphQL resolver
func NewResolver(bookingService service.BookingService, fx xchange.XChange, log *slog.Logger) *Resolver {
	return &Resolver{
		bookingService: bookingService,
		log:            log,
		fx:             fx,
	}
}

// ============================================================================
// Query Resolvers
// ============================================================================

// QuoteBooking returns pricing and availability without creating a booking
func (r *Resolver) QuoteBooking(ctx context.Context, listingID string, checkIn string, checkOut string, guestCount int) (*domain.BookingQuote, error) {
	// Parse listing ID
	lID, err := uuid.Parse(listingID)
	if err != nil {
		return nil, fmt.Errorf("invalid listing ID")
	}

	// Parse check-in and check-out times
	checkInTime, err := time.Parse(time.RFC3339, checkIn)
	if err != nil {
		return nil, fmt.Errorf("invalid check-in time format, use RFC3339")
	}

	checkOutTime, err := time.Parse(time.RFC3339, checkOut)
	if err != nil {
		return nil, fmt.Errorf("invalid check-out time format, use RFC3339")
	}

	// Validate guest count
	if guestCount < 1 {
		return nil, fmt.Errorf("guest count must be at least 1")
	}

	quote, err := r.bookingService.QuoteBooking(ctx, lID, checkInTime, checkOutTime, guestCount)
	if err != nil {
		r.log.Error("failed to generate quote", "error", err)
		return nil, err
	}

	r.localizeBookingQuote(ctx, quote)
	return quote, nil
}

// Booking retrieves a booking by ID
func (r *Resolver) Booking(ctx context.Context, id string) (*domain.Booking, error) {
	bookingID, err := uuid.Parse(id)
	if err != nil {
		r.log.Error("invalid booking ID", "booking_id", id, "error", err)
		return nil, fmt.Errorf("invalid booking ID")
	}

	// Get current user from context
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	booking, err := r.bookingService.GetBooking(ctx, bookingID, userID)
	if err != nil {
		r.log.Error("failed to get booking", "booking_id", id, "error", err)
		return nil, err
	}

	r.localizeBooking(ctx, booking)
	return booking, nil
}

// MyBookings lists bookings for the authenticated guest
func (r *Resolver) MyBookings(ctx context.Context, limit *int, offset *int) ([]*domain.Booking, error) {
	// Get current user from context
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	l := 20
	if limit != nil && *limit > 0 {
		l = *limit
	}

	o := 0
	if offset != nil && *offset > 0 {
		o = *offset
	}

	bookings, err := r.bookingService.ListBookingsForGuest(ctx, userID, l, o)
	if err != nil {
		r.log.Error("failed to list bookings for guest", "user_id", userID, "error", err)
		return nil, err
	}

	for i := range bookings {
		r.localizeBooking(ctx, bookings[i])
	}
	return bookings, nil
}

// ListingBookings lists bookings for a specific listing (owner/host view)
func (r *Resolver) ListingBookings(ctx context.Context, listingID string, status *string, limit *int, offset *int) ([]*domain.Booking, error) {
	// Get current user from context
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	lID, err := uuid.Parse(listingID)
	if err != nil {
		r.log.Error("invalid listing ID", "listing_id", listingID, "error", err)
		return nil, fmt.Errorf("invalid listing ID")
	}

	l := 20
	if limit != nil && *limit > 0 {
		l = *limit
	}

	o := 0
	if offset != nil && *offset > 0 {
		o = *offset
	}

	var bookingStatus *domain.BookingStatus
	if status != nil {
		bs := domain.BookingStatus(*status)
		bookingStatus = &bs
	}

	bookings, err := r.bookingService.ListBookingsForListing(ctx, lID, userID, bookingStatus, l, o)
	if err != nil {
		r.log.Error("failed to list bookings for listing", "listing_id", listingID, "error", err)
		return nil, err
	}

	for i := range bookings {
		r.localizeBooking(ctx, bookings[i])
	}
	return bookings, nil
}

// ============================================================================
// Mutation Resolvers
// ============================================================================

// ReserveBookingInput represents the input for creating an instant booking with payment
type ReserveBookingInput struct {
	ListingID       string  `json:"listingId"`
	CheckIn         string  `json:"checkIn"`
	CheckOut        string  `json:"checkOut"`
	GuestCount      int     `json:"guestCount"`
	PaymentMethodID *string `json:"paymentMethodId"`
	SpecialRequests *string `json:"specialRequests"`
}

// ReserveBooking creates an instant booking and initiates payment in one operation
func (r *Resolver) ReserveBooking(ctx context.Context, input *ReserveBookingInput) (*CompleteBookingPayload, error) {
	// Get current user from context
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	// Parse listing ID
	listingID, err := uuid.Parse(input.ListingID)
	if err != nil {
		return nil, fmt.Errorf("invalid listing ID")
	}

	// Parse check-in and check-out times
	checkIn, err := time.Parse(time.RFC3339, input.CheckIn)
	if err != nil {
		return nil, fmt.Errorf("invalid check-in time format, use RFC3339")
	}

	checkOut, err := time.Parse(time.RFC3339, input.CheckOut)
	if err != nil {
		return nil, fmt.Errorf("invalid check-out time format, use RFC3339")
	}

	// Parse payment method ID if provided
	var paymentMethodID *uuid.UUID
	if input.PaymentMethodID != nil {
		pmID, err := uuid.Parse(*input.PaymentMethodID)
		if err != nil {
			return nil, fmt.Errorf("invalid payment method ID")
		}
		paymentMethodID = &pmID
	}

	booking, paymentResult, err := r.bookingService.ReserveBooking(
		ctx,
		listingID,
		userID,
		checkIn,
		checkOut,
		input.GuestCount,
		paymentMethodID,
		input.SpecialRequests,
	)
	if err != nil {
		r.log.Error("failed to reserve booking", "error", err)
		return nil, err
	}

	r.log.Info("instant booking reserved", "booking_id", booking.ID, "payment_id", paymentResult.PaymentID, "status", paymentResult.Status)

	// Build response payload
	payload := &CompleteBookingPayload{
		Booking:               booking,
		PaymentID:             paymentResult.PaymentID.String(),
		PaymentStatus:         paymentResult.Status,
		PaymentReference:      paymentResult.Reference,
		AuthorizationURL:      paymentResult.AuthorizationURL,
		RequiresAuthorization: paymentResult.AuthorizationURL != nil,
	}

	r.localizeBooking(ctx, payload.Booking)
	return payload, nil
}

// RequestBookingInput represents the input for creating a manual-approval booking
type RequestBookingInput struct {
	ListingID       string  `json:"listingId"`
	CheckIn         string  `json:"checkIn"`
	CheckOut        string  `json:"checkOut"`
	GuestCount      int     `json:"guestCount"`
	SpecialRequests *string `json:"specialRequests"`
}

// RequestBooking creates a manual-approval booking request
func (r *Resolver) RequestBooking(ctx context.Context, input *RequestBookingInput) (*domain.Booking, error) {
	// Get current user from context
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	// Parse listing ID
	listingID, err := uuid.Parse(input.ListingID)
	if err != nil {
		return nil, fmt.Errorf("invalid listing ID")
	}

	// Parse check-in and check-out times
	checkIn, err := time.Parse(time.RFC3339, input.CheckIn)
	if err != nil {
		return nil, fmt.Errorf("invalid check-in time format, use RFC3339")
	}

	checkOut, err := time.Parse(time.RFC3339, input.CheckOut)
	if err != nil {
		return nil, fmt.Errorf("invalid check-out time format, use RFC3339")
	}

	booking, err := r.bookingService.RequestBooking(
		ctx,
		listingID,
		userID,
		checkIn,
		checkOut,
		input.GuestCount,
		input.SpecialRequests,
	)
	if err != nil {
		r.log.Error("failed to create booking request", "error", err)
		return nil, err
	}

	r.log.Info("booking request created", "booking_id", booking.ID)
	r.localizeBooking(ctx, booking)
	return booking, nil
}

// PayForBookingInput represents the input for paying for an approved booking
type PayForBookingInput struct {
	BookingID       string  `json:"bookingId"`
	PaymentMethodID *string `json:"paymentMethodId"`
}

// PayForBooking processes payment for an existing booking
func (r *Resolver) PayForBooking(ctx context.Context, input *PayForBookingInput) (*CompleteBookingPayload, error) {
	// Get current user from context
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	bookingID, err := uuid.Parse(input.BookingID)
	if err != nil {
		return nil, fmt.Errorf("invalid booking ID")
	}

	var paymentMethodID *uuid.UUID
	if input.PaymentMethodID != nil {
		pmID, err := uuid.Parse(*input.PaymentMethodID)
		if err != nil {
			return nil, fmt.Errorf("invalid payment method ID")
		}
		paymentMethodID = &pmID
	}

	booking, paymentResult, err := r.bookingService.PayForBooking(ctx, bookingID, userID, paymentMethodID)
	if err != nil {
		r.log.Error("failed to process payment for booking", "booking_id", input.BookingID, "error", err)
		return nil, err
	}

	r.log.Info("payment processed for booking", "booking_id", booking.ID, "payment_id", paymentResult.PaymentID, "status", paymentResult.Status)

	// Build response payload
	payload := &CompleteBookingPayload{
		Booking:               booking,
		PaymentID:             paymentResult.PaymentID.String(),
		PaymentStatus:         paymentResult.Status,
		PaymentReference:      paymentResult.Reference,
		AuthorizationURL:      paymentResult.AuthorizationURL,
		RequiresAuthorization: paymentResult.AuthorizationURL != nil,
	}

	r.localizeBooking(ctx, payload.Booking)
	return payload, nil
}

// ConfirmBooking confirms a booking (for request-type bookings)
func (r *Resolver) ConfirmBooking(ctx context.Context, bookingID string) (*domain.Booking, error) {
	// Get current user from context
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	bID, err := uuid.Parse(bookingID)
	if err != nil {
		return nil, fmt.Errorf("invalid booking ID")
	}

	booking, err := r.bookingService.ConfirmBooking(ctx, bID, userID)
	if err != nil {
		r.log.Error("failed to confirm booking", "booking_id", bookingID, "error", err)
		return nil, err
	}

	r.log.Info("booking confirmed", "booking_id", booking.ID)
	r.localizeBooking(ctx, booking)
	return booking, nil
}

// CancelBookingInput represents the input for cancelling a booking
type CancelBookingInput struct {
	BookingID string  `json:"bookingId"`
	Reason    *string `json:"reason"`
}

// CancelBooking cancels a booking
func (r *Resolver) CancelBooking(ctx context.Context, input *CancelBookingInput) (*domain.Booking, error) {
	// Get current user from context
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	bookingID, err := uuid.Parse(input.BookingID)
	if err != nil {
		return nil, fmt.Errorf("invalid booking ID")
	}

	booking, err := r.bookingService.CancelBooking(ctx, bookingID, userID, input.Reason)
	if err != nil {
		r.log.Error("failed to cancel booking", "booking_id", input.BookingID, "error", err)
		return nil, err
	}

	r.log.Info("booking cancelled", "booking_id", booking.ID)
	r.localizeBooking(ctx, booking)
	return booking, nil
}

// CheckInBooking records an actual check-in time for a booking (host-only).
func (r *Resolver) CheckInBooking(ctx context.Context, bookingID string) (*domain.Booking, error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	id, err := uuid.Parse(bookingID)
	if err != nil {
		return nil, fmt.Errorf("invalid booking ID")
	}

	booking, err := r.bookingService.CheckInBooking(ctx, id, userID)
	if err != nil {
		r.log.Error("failed to check in booking", "error", err)
		return nil, err
	}

	r.localizeBooking(ctx, booking)
	return booking, nil
}

// CheckOutBooking records an actual check-out time for a booking (host-only).
func (r *Resolver) CheckOutBooking(ctx context.Context, bookingID string) (*domain.Booking, error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	id, err := uuid.Parse(bookingID)
	if err != nil {
		return nil, fmt.Errorf("invalid booking ID")
	}

	booking, err := r.bookingService.CheckOutBooking(ctx, id, userID)
	if err != nil {
		r.log.Error("failed to check out booking", "error", err)
		return nil, err
	}

	r.localizeBooking(ctx, booking)
	return booking, nil
}

// CompleteBookingPayload represents the response from booking with payment
type CompleteBookingPayload struct {
	Booking               *domain.Booking `json:"booking"`
	PaymentID             string          `json:"paymentId"`
	PaymentStatus         string          `json:"paymentStatus"`
	PaymentReference      string          `json:"paymentReference"`
	AuthorizationURL      *string         `json:"authorizationUrl"`
	RequiresAuthorization bool            `json:"requiresAuthorization"`
}

func (r *Resolver) localizeBookingQuote(ctx context.Context, quote *domain.BookingQuote) {
	if quote == nil || r.fx == nil {
		return
	}

	target, ok := localization.PreferredCurrency(ctx)
	if !ok {
		return
	}
	target = xchange.NormalizeCurrency(target)

	source := quote.Currency
	if source == "" {
		source = xchange.NormalizeCurrency("")
	}
	if strings.EqualFold(source, target) {
		return
	}

	rate, err := r.fx.GetExchangeRate(source, target)
	if err != nil {
		if r.log != nil {
			r.log.Warn("failed to convert booking quote", "listing_id", quote.ListingID, "from", source, "to", target, "error", err)
		}
		return
	}

	convert := func(value float64) float64 {
		return math.Round(value*rate*100) / 100
	}

	quote.TotalPrice = convert(quote.TotalPrice)
	r.localizePriceBreakdownSnapshot(quote.PriceBreakdown, convert, target)
	quote.Currency = target
}

func (r *Resolver) localizeBooking(ctx context.Context, booking *domain.Booking) {
	if booking == nil || r.fx == nil {
		return
	}

	target, ok := localization.PreferredCurrency(ctx)
	if !ok {
		return
	}
	target = xchange.NormalizeCurrency(target)

	source := booking.Currency
	if source == "" {
		source = xchange.NormalizeCurrency("")
	}
	if strings.EqualFold(source, target) {
		return
	}

	rate, err := r.fx.GetExchangeRate(source, target)
	if err != nil {
		if r.log != nil {
			r.log.Warn("failed to convert booking", "booking_id", booking.ID, "from", source, "to", target, "error", err)
		}
		return
	}

	convert := func(value float64) float64 {
		return math.Round(value*rate*100) / 100
	}

	booking.TotalPrice = convert(booking.TotalPrice)
	r.localizePriceBreakdownSnapshot(booking.PriceBreakdown, convert, target)
	booking.Currency = target
}

func (r *Resolver) localizePriceBreakdownSnapshot(breakdown *domain.PriceBreakdownSnapshot, convert func(float64) float64, currency string) {
	if breakdown == nil {
		return
	}

	breakdown.BaseTotal = convert(breakdown.BaseTotal)
	breakdown.ExtraGuestFee = convert(breakdown.ExtraGuestFee)
	breakdown.Subtotal = convert(breakdown.Subtotal)
	breakdown.Total = convert(breakdown.Total)
	breakdown.Currency = currency

	if breakdown.CleaningFee != nil {
		val := convert(*breakdown.CleaningFee)
		breakdown.CleaningFee = &val
	}
	if breakdown.ServiceFee != nil {
		val := convert(*breakdown.ServiceFee)
		breakdown.ServiceFee = &val
	}
	if breakdown.CautionFee != nil {
		val := convert(*breakdown.CautionFee)
		breakdown.CautionFee = &val
	}

	for i := range breakdown.Discounts {
		breakdown.Discounts[i].Amount = convert(breakdown.Discounts[i].Amount)
	}

	for i := range breakdown.NightlyRates {
		breakdown.NightlyRates[i].BaseRate = convert(breakdown.NightlyRates[i].BaseRate)
		breakdown.NightlyRates[i].FinalRate = convert(breakdown.NightlyRates[i].FinalRate)
	}

	if breakdown.PlatformFees != nil {
		breakdown.PlatformFees.GuestFeeAmount = convert(breakdown.PlatformFees.GuestFeeAmount)
		breakdown.PlatformFees.HostCommissionAmount = convert(breakdown.PlatformFees.HostCommissionAmount)
		breakdown.PlatformFees.PayoutProcessingAmount = convert(breakdown.PlatformFees.PayoutProcessingAmount)
		breakdown.PlatformFees.HostNetAmount = convert(breakdown.PlatformFees.HostNetAmount)
	}
}
