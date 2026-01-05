package hooks

import (
	"context"
	"fmt"

	"hauslet/internal/modules/booking/service"
	paymentdomain "hauslet/internal/modules/payments/domain"

	"github.com/google/uuid"
)

// BookingHooksAdapter provides payment lifecycle hooks for the booking module.
// This adapter is called by the payment webhook handler to notify booking
// of payment events (success, failure, refund).
type BookingHooksAdapter struct {
	svc service.BookingService
}

// NewBookingHooksAdapter creates a new booking hooks adapter.
func NewBookingHooksAdapter(svc service.BookingService) *BookingHooksAdapter {
	return &BookingHooksAdapter{
		svc: svc,
	}
}

// OnPaymentSucceeded handles successful payment events.
// Called by payment webhook handler when charge.success event is received.
func (a *BookingHooksAdapter) OnPaymentSucceeded(ctx context.Context, bookingID uuid.UUID, payment *paymentdomain.Payment) error {
	if payment == nil {
		return fmt.Errorf("payment is nil")
	}

	// Call booking service to handle payment success
	// This will transition booking from awaiting_payment -> confirmed
	if err := a.svc.HandlePaymentSuccess(ctx, bookingID, payment.ID); err != nil {
		return fmt.Errorf("failed to handle payment success for booking %s: %w", bookingID, err)
	}

	return nil
}

// OnPaymentFailed handles failed payment events.
// Called by payment webhook handler when charge.failed event is received.
func (a *BookingHooksAdapter) OnPaymentFailed(ctx context.Context, bookingID uuid.UUID, payment *paymentdomain.Payment, reason string) error {
	if payment == nil {
		return fmt.Errorf("payment is nil")
	}

	// Call booking service to handle payment failure
	// This may log the failure but not immediately cancel the booking
	// (user might retry within the hold window)
	if err := a.svc.HandlePaymentFailure(ctx, bookingID, payment.ID, reason); err != nil {
		return fmt.Errorf("failed to handle payment failure for booking %s: %w", bookingID, err)
	}

	return nil
}

// OnPaymentRefunded handles refund events.
// Called by payment webhook handler when refund.processed event is received.
func (a *BookingHooksAdapter) OnPaymentRefunded(ctx context.Context, bookingID uuid.UUID, payment *paymentdomain.Payment) error {
	if payment == nil {
		return fmt.Errorf("payment is nil")
	}

	// Call booking service to handle refund
	// This may update booking status or create a dispute record
	if err := a.svc.HandlePaymentRefund(ctx, bookingID, payment.ID, payment.RefundedAmount); err != nil {
		return fmt.Errorf("failed to handle payment refund for booking %s: %w", bookingID, err)
	}

	return nil
}

// GetBookingStatus returns the current status of a booking.
// Helper method for payment module to check booking state.
func (a *BookingHooksAdapter) GetBookingStatus(ctx context.Context, bookingID uuid.UUID) (string, error) {
	booking, err := a.svc.GetBooking(ctx, bookingID, uuid.Nil) // Admin context for internal call
	if err != nil {
		return "", fmt.Errorf("failed to get booking %s: %w", bookingID, err)
	}

	return string(booking.Status), nil
}

// GetBookingTotal returns the total amount and currency for a booking.
// Helper method for payment validation.
func (a *BookingHooksAdapter) GetBookingTotal(ctx context.Context, bookingID uuid.UUID) (float64, string, error) {
	booking, err := a.svc.GetBooking(ctx, bookingID, uuid.Nil) // Admin context for internal call
	if err != nil {
		return 0, "", fmt.Errorf("failed to get booking %s: %w", bookingID, err)
	}

	return booking.TotalPrice, booking.Currency, nil
}

// CanCancelBooking checks if a booking can be cancelled.
// Helper method for refund logic.
func (a *BookingHooksAdapter) CanCancelBooking(ctx context.Context, bookingID uuid.UUID) (bool, error) {
	booking, err := a.svc.GetBooking(ctx, bookingID, uuid.Nil) // Admin context for internal call
	if err != nil {
		return false, fmt.Errorf("failed to get booking %s: %w", bookingID, err)
	}

	return booking.CanBeCancelled(), nil
}

// GetBookingGuest returns the guest ID for a booking.
// Helper method for authorization checks.
func (a *BookingHooksAdapter) GetBookingGuest(ctx context.Context, bookingID uuid.UUID) (uuid.UUID, error) {
	booking, err := a.svc.GetBooking(ctx, bookingID, uuid.Nil) // Admin context for internal call
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to get booking %s: %w", bookingID, err)
	}

	return booking.GuestID, nil
}
