package hooks

import (
	"context"
	"hauslet/internal/modules/payments/domain"
	"hauslet/internal/modules/payments/service"

	"github.com/google/uuid"
)

// PaymentHooksAdapter provides payment hooks for other modules
type PaymentHooksAdapter struct {
	svc service.PaymentService
}

// NewPaymentHooksAdapter creates a new payment hooks adapter
func NewPaymentHooksAdapter(svc service.PaymentService) *PaymentHooksAdapter {
	return &PaymentHooksAdapter{svc: svc}
}

// GetPaymentStatus retrieves the status of a payment
func (a *PaymentHooksAdapter) GetPaymentStatus(ctx context.Context, paymentID uuid.UUID) (string, error) {
	payment, err := a.svc.GetPayment(ctx, paymentID)
	if err != nil {
		return "", err
	}
	return string(payment.Status), nil
}

// HasUserPaid checks if a user has successfully paid for a booking
func (a *PaymentHooksAdapter) HasUserPaid(ctx context.Context, bookingID uuid.UUID) (bool, error) {
	payments, err := a.svc.ListPaymentsByBooking(ctx, bookingID)
	if err != nil {
		return false, err
	}

	for _, pmt := range payments {
		if pmt.Status == domain.PaymentStatusSucceeded {
			return true, nil
		}
	}

	return false, nil
}

// GetTotalPaidForBooking returns the total amount paid for a booking
func (a *PaymentHooksAdapter) GetTotalPaidForBooking(ctx context.Context, bookingID uuid.UUID) (int64, error) {
	payments, err := a.svc.ListPaymentsByBooking(ctx, bookingID)
	if err != nil {
		return 0, err
	}

	var total int64
	for _, pmt := range payments {
		if pmt.Status == domain.PaymentStatusSucceeded {
			total += pmt.Amount
		}
	}

	return total, nil
}

// CanRefund checks if a payment can be refunded
func (a *PaymentHooksAdapter) CanRefund(ctx context.Context, paymentID uuid.UUID) (bool, error) {
	payment, err := a.svc.GetPayment(ctx, paymentID)
	if err != nil {
		return false, err
	}
	return payment.CanRefund(), nil
}

// HasPayoutDetails checks if a user or business has payout details configured
func (a *PaymentHooksAdapter) HasPayoutDetails(ctx context.Context, userID *uuid.UUID, businessID *uuid.UUID) (bool, error) {
	details, err := a.svc.ListPayoutDetails(ctx, userID, businessID)
	if err != nil {
		return false, err
	}
	return len(details) > 0, nil
}
