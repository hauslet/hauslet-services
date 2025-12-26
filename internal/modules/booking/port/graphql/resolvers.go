package graphql

import (
	"context"
	"fmt"
	"hauslet/internal/modules/booking/domain"
	"hauslet/internal/modules/booking/service"
	"time"

	"github.com/go-pkgz/lgr"
	"github.com/google/uuid"
)

// Resolver handles GraphQL queries and mutations for bookings
type Resolver struct {
	bookingService service.BookingService
	log            *lgr.Logger
}

// NewResolver creates a new GraphQL resolver
func NewResolver(bookingService service.BookingService, log *lgr.Logger) *Resolver {
	return &Resolver{
		bookingService: bookingService,
		log:            log,
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
		r.log.Logf("ERROR failed to generate quote: %v", err)
		return nil, err
	}

	return quote, nil
}

// Booking retrieves a booking by ID
func (r *Resolver) Booking(ctx context.Context, id string) (*domain.Booking, error) {
	bookingID, err := uuid.Parse(id)
	if err != nil {
		r.log.Logf("ERROR invalid booking ID %s: %v", id, err)
		return nil, fmt.Errorf("invalid booking ID")
	}

	// Get current user from context
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	booking, err := r.bookingService.GetBooking(ctx, bookingID, userID)
	if err != nil {
		r.log.Logf("ERROR failed to get booking %s: %v", id, err)
		return nil, err
	}

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
		r.log.Logf("ERROR failed to list bookings for guest %s: %v", userID, err)
		return nil, err
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
		r.log.Logf("ERROR invalid listing ID %s: %v", listingID, err)
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
		r.log.Logf("ERROR failed to list bookings for listing %s: %v", listingID, err)
		return nil, err
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
		r.log.Logf("ERROR failed to reserve booking: %v", err)
		return nil, err
	}

	r.log.Logf("INFO instant booking reserved: %s (payment=%s, status=%s)",
		booking.ID, paymentResult.PaymentID, paymentResult.Status)

	// Build response payload
	payload := &CompleteBookingPayload{
		Booking:               booking,
		PaymentID:             paymentResult.PaymentID.String(),
		PaymentStatus:         paymentResult.Status,
		PaymentReference:      paymentResult.Reference,
		AuthorizationURL:      paymentResult.AuthorizationURL,
		RequiresAuthorization: paymentResult.AuthorizationURL != nil,
	}

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
		r.log.Logf("ERROR failed to create booking request: %v", err)
		return nil, err
	}

	r.log.Logf("INFO booking request created: %s", booking.ID)
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
		r.log.Logf("ERROR failed to process payment for booking %s: %v", input.BookingID, err)
		return nil, err
	}

	r.log.Logf("INFO payment processed for booking %s (payment=%s, status=%s)",
		booking.ID, paymentResult.PaymentID, paymentResult.Status)

	// Build response payload
	payload := &CompleteBookingPayload{
		Booking:               booking,
		PaymentID:             paymentResult.PaymentID.String(),
		PaymentStatus:         paymentResult.Status,
		PaymentReference:      paymentResult.Reference,
		AuthorizationURL:      paymentResult.AuthorizationURL,
		RequiresAuthorization: paymentResult.AuthorizationURL != nil,
	}

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
		r.log.Logf("ERROR failed to confirm booking %s: %v", bookingID, err)
		return nil, err
	}

	r.log.Logf("INFO booking confirmed: %s", booking.ID)
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
		r.log.Logf("ERROR failed to cancel booking %s: %v", input.BookingID, err)
		return nil, err
	}

	r.log.Logf("INFO booking cancelled: %s", booking.ID)
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
