package graphql

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"hauslet/internal/modules/booking/domain"
	"hauslet/internal/modules/booking/service"
	propertydomain "hauslet/internal/modules/property/domain"
	"hauslet/internal/transport/graph/loaders"
	"hauslet/internal/transport/graph/viewer"

	"github.com/google/uuid"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

// Resolver handles GraphQL queries and mutations for bookings
type Resolver struct {
	bookingService service.BookingService
	log            *slog.Logger
}

// NewResolver creates a new GraphQL resolver
func NewResolver(bookingService service.BookingService, log *slog.Logger) *Resolver {
	return &Resolver{
		bookingService: bookingService,
		log:            log,
	}
}

// ============================================================================
// Query Resolvers
// ============================================================================

// QuoteBooking returns pricing and availability without creating a booking
func (r *Resolver) QuoteBooking(ctx context.Context, listingID uuid.UUID, checkIn time.Time, checkOut time.Time, guestCount int) (*domain.BookingQuote, error) {

	quote, err := r.bookingService.QuoteBooking(ctx, listingID, checkIn, checkOut, guestCount)
	if err != nil {
		r.log.Error("failed to generate quote", "error", err)
		return nil, r.translateBookingError(err)
	}

	r.bookingService.LocalizeQuote(ctx, quote)
	return quote, nil
}

// Booking retrieves a booking by ID
func (r *Resolver) Booking(ctx context.Context, id uuid.UUID) (*domain.Booking, error) {
	// Get current user from context
	userID, err := viewer.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	booking, err := r.bookingService.GetBooking(ctx, id, userID)
	if err != nil {
		r.log.Error("failed to get booking", "booking_id", id, "error", err)
		return nil, err
	}

	r.bookingService.LocalizeBooking(ctx, booking)
	return booking, nil
}

// BookingByReference retrieves a booking by reference
func (r *Resolver) BookingByReference(ctx context.Context, reference string) (*domain.Booking, error) {
	userID, err := viewer.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	booking, err := r.bookingService.GetBookingByReference(ctx, reference, userID)
	if err != nil {
		r.log.Error("failed to get booking by reference", "reference", reference, "error", err)
		return nil, err
	}

	r.bookingService.LocalizeBooking(ctx, booking)
	return booking, nil
}

// MyBookings lists bookings for the authenticated guest
func (r *Resolver) MyBookings(ctx context.Context, limit *int, offset *int) ([]*domain.Booking, error) {
	// Get current user from context
	userID, err := viewer.GetUserIDFromContext(ctx)
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
		r.bookingService.LocalizeBooking(ctx, bookings[i])
	}
	return bookings, nil
}

// ListingBookings lists bookings for a specific listing (owner/host view)
func (r *Resolver) ListingBookings(ctx context.Context, listingID uuid.UUID, status *domain.BookingStatus, limit *int, offset *int) ([]*domain.Booking, error) {
	// Get current user from context
	userID, err := viewer.GetUserIDFromContext(ctx)
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

	bookings, err := r.bookingService.ListBookingsForListing(ctx, listingID, userID, status, l, o)
	if err != nil {
		r.log.Error("failed to list bookings for listing", "listing_id", listingID, "error", err)
		return nil, err
	}

	for i := range bookings {
		r.bookingService.LocalizeBooking(ctx, bookings[i])
	}
	return bookings, nil
}

// MyHostBookings lists bookings for listings owned by the authenticated host
func (r *Resolver) MyHostBookings(ctx context.Context, status *domain.BookingStatus, limit *int, offset *int) ([]*domain.Booking, error) {
	userID, err := viewer.GetUserIDFromContext(ctx)
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

	bookings, err := r.bookingService.ListBookingsForHost(ctx, userID, status, l, o)
	if err != nil {
		r.log.Error("failed to list host bookings", "host_id", userID, "error", err)
		return nil, err
	}

	for i := range bookings {
		r.bookingService.LocalizeBooking(ctx, bookings[i])
	}
	return bookings, nil
}

// Listing resolves the listing for a booking
func (r *Resolver) Listing(ctx context.Context, obj *domain.Booking) (*propertydomain.Listing, error) {
	loaders := loaders.For(ctx)
	if loaders == nil || loaders.Listing == nil {
		r.log.Warn("listing loader not found in context")
		return nil, nil
	}

	return loaders.Listing.Load(ctx, obj.ListingID)
}

// CancellationPenaltyPreview represents the penalty preview for a host cancellation.
type CancellationPenaltyPreview struct {
	CancellationCount    int    `json:"cancellationCount"`
	PenaltyAmount        int    `json:"penaltyAmount"`
	SuspensionDays       int    `json:"suspensionDays"`
	IsNewHostGracePeriod bool   `json:"isNewHostGracePeriod"`
	RequiresReview       bool   `json:"requiresReview"`
	WarningMessage       string `json:"warningMessage"`
}

// PreviewHostCancellationPenalty previews the penalty for a host cancellation.
func (r *Resolver) PreviewHostCancellationPenalty(ctx context.Context, bookingID uuid.UUID) (*CancellationPenaltyPreview, error) {
	userID, err := viewer.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	result, err := r.bookingService.PreviewHostCancellationPenalty(ctx, bookingID, userID)
	if err != nil {
		r.log.Error("failed to preview host cancellation penalty", "booking_id", bookingID, "error", err)
		return nil, err
	}

	return &CancellationPenaltyPreview{
		CancellationCount:    result.CancellationCount,
		PenaltyAmount:        int(result.PenaltyAmount),
		SuspensionDays:       result.SuspensionDays,
		IsNewHostGracePeriod: result.IsNewHostGracePeriod,
		RequiresReview:       result.RequiresReview,
		WarningMessage:       result.WarningMessage,
	}, nil
}

// ============================================================================
// Mutation Resolvers
// ============================================================================

// ReserveBookingInput represents the input for creating an instant booking with payment
type ReserveBookingInput struct {
	ListingID       uuid.UUID  `json:"listingId"`
	CheckIn         time.Time  `json:"checkIn"`
	CheckOut        time.Time  `json:"checkOut"`
	GuestCount      int        `json:"guestCount"`
	PaymentMethodID *uuid.UUID `json:"paymentMethodId"`
	SpecialRequests *string    `json:"specialRequests"`
}

// ReserveBooking creates an instant booking and initiates payment in one operation
func (r *Resolver) ReserveBooking(ctx context.Context, input ReserveBookingInput) (*CompleteBookingPayload, error) {
	// Get current user from context
	userID, err := viewer.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	booking, paymentResult, err := r.bookingService.ReserveBooking(
		ctx,
		input.ListingID,
		userID,
		input.CheckIn,
		input.CheckOut,
		input.GuestCount,
		input.PaymentMethodID,
		input.SpecialRequests,
	)
	if err != nil {
		r.log.Error("failed to reserve booking", "error", err)
		return nil, r.translateBookingError(err)
	}

	r.log.Info("instant booking reserved", "booking_id", booking.ID, "payment_id", paymentResult.PaymentID, "status", paymentResult.Status)

	// Build response payload
	payload := &CompleteBookingPayload{
		Booking:               booking,
		PaymentID:             paymentResult.PaymentID.String(),
		PaymentStatus:         paymentResult.Status,
		PaymentReference:      paymentResult.Reference,
		AuthorizationURL:      paymentResult.AuthorizationURL,
		RequiresAuthorization: paymentResult.RequiresAction,
	}

	r.bookingService.LocalizeBooking(ctx, payload.Booking)
	return payload, nil
}

// RequestBookingInput represents the input for creating a manual-approval booking
type RequestBookingInput struct {
	ListingID       uuid.UUID `json:"listingId"`
	CheckIn         time.Time `json:"checkIn"`
	CheckOut        time.Time `json:"checkOut"`
	GuestCount      int       `json:"guestCount"`
	SpecialRequests *string   `json:"specialRequests"`
}

// RequestBooking creates a manual-approval booking request
func (r *Resolver) RequestBooking(ctx context.Context, input RequestBookingInput) (*domain.Booking, error) {
	// Get current user from context
	userID, err := viewer.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	booking, err := r.bookingService.RequestBooking(
		ctx,
		input.ListingID,
		userID,
		input.CheckIn,
		input.CheckOut,
		input.GuestCount,
		input.SpecialRequests,
	)
	if err != nil {
		r.log.Error("failed to create booking request", "error", err)
		return nil, r.translateBookingError(err)
	}

	r.log.Info("booking request created", "booking_id", booking.ID)
	r.bookingService.LocalizeBooking(ctx, booking)
	return booking, nil
}

// PayForBookingInput represents the input for paying for an approved booking
type PayForBookingInput struct {
	BookingID       uuid.UUID  `json:"bookingId"`
	PaymentMethodID *uuid.UUID `json:"paymentMethodId"`
}

// PayForBooking processes payment for an existing booking
func (r *Resolver) PayForBooking(ctx context.Context, input PayForBookingInput) (*CompleteBookingPayload, error) {
	// Get current user from context
	userID, err := viewer.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	booking, paymentResult, err := r.bookingService.PayForBooking(ctx, input.BookingID, userID, input.PaymentMethodID)
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
		RequiresAuthorization: paymentResult.RequiresAction,
	}

	r.bookingService.LocalizeBooking(ctx, payload.Booking)
	return payload, nil
}

// ConfirmBooking confirms a booking (for request-type bookings)
func (r *Resolver) ConfirmBooking(ctx context.Context, bookingID uuid.UUID) (*domain.Booking, error) {
	// Get current user from context
	userID, err := viewer.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	booking, err := r.bookingService.ConfirmBooking(ctx, bookingID, userID)
	if err != nil {
		r.log.Error("failed to confirm booking", "booking_id", bookingID, "error", err)
		return nil, err
	}

	r.log.Info("booking confirmed", "booking_id", booking.ID)
	r.bookingService.LocalizeBooking(ctx, booking)
	return booking, nil
}

// CancelBookingInput represents the input for cancelling a booking
type CancelBookingInput struct {
	BookingID uuid.UUID `json:"bookingId"`
	Reason    *string   `json:"reason"`
}

// CancelBooking cancels a booking
func (r *Resolver) CancelBooking(ctx context.Context, input CancelBookingInput) (*domain.Booking, error) {
	// Get current user from context
	userID, err := viewer.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	booking, err := r.bookingService.CancelBooking(ctx, input.BookingID, userID, input.Reason)
	if err != nil {
		r.log.Error("failed to cancel booking", "booking_id", input.BookingID, "error", err)
		return nil, err
	}

	r.log.Info("booking cancelled", "booking_id", booking.ID)
	r.bookingService.LocalizeBooking(ctx, booking)
	return booking, nil
}

// CheckInBooking records an actual check-in time for a booking (host-only).
func (r *Resolver) CheckInBooking(ctx context.Context, bookingID uuid.UUID) (*domain.Booking, error) {
	userID, err := viewer.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	booking, err := r.bookingService.CheckInBooking(ctx, bookingID, userID)
	if err != nil {
		r.log.Error("failed to check in booking", "error", err)
		return nil, err
	}

	r.bookingService.LocalizeBooking(ctx, booking)
	return booking, nil
}

// CheckOutBooking records an actual check-out time for a booking (host-only).
func (r *Resolver) CheckOutBooking(ctx context.Context, bookingID uuid.UUID) (*domain.Booking, error) {
	userID, err := viewer.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	booking, err := r.bookingService.CheckOutBooking(ctx, bookingID, userID)
	if err != nil {
		r.log.Error("failed to check out booking", "error", err)
		return nil, err
	}

	r.bookingService.LocalizeBooking(ctx, booking)
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

func (r *Resolver) translateBookingError(err error) error {
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, domain.ErrListingSuspended):
		return gqlerror.Errorf("This listing is temporarily suspended from bookings")
	case errors.Is(err, domain.ErrListingUnavailable):
		return gqlerror.Errorf("This listing is currently unavailable for booking")
	case errors.Is(err, domain.ErrGuestIDVerificationRequired):
		return gqlerror.Errorf("ID verification is required before booking this listing")
	case errors.Is(err, domain.ErrGuestProfilePhotoRequired):
		return gqlerror.Errorf("Please upload a profile photo before booking this listing")
	case errors.Is(err, domain.ErrGuestPositiveReviewsRequired):
		return gqlerror.Errorf("This listing only accepts guests with positive reviews")
	case errors.Is(err, domain.ErrBookingWindowExceeded):
		return gqlerror.Errorf("The chosen dates fall outside the allowed advance booking window")
	case errors.Is(err, domain.ErrLeadTimeNotMet):
		return gqlerror.Errorf("The host requires more advance notice for this booking")
	default:
		return err
	}
}
