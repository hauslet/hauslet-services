package adapters

import (
	"context"
	"hauslet/internal/modules/booking/service"
	paymentdomain "hauslet/internal/modules/payments/domain"
	paymentsservice "hauslet/internal/modules/payments/service"
	"hauslet/internal/platform/payment"
	"log/slog"

	"github.com/google/uuid"
)

// PaymentServiceAdapter adapts the payment service for booking's PaymentGateway interface.
type PaymentServiceAdapter struct {
	paymentSvc paymentsservice.PaymentService
	log        *slog.Logger
}

// NewPaymentServiceAdapter creates a new payment service adapter.
func NewPaymentServiceAdapter(svc paymentsservice.PaymentService, log *slog.Logger) service.PaymentGateway {
	return &PaymentServiceAdapter{
		paymentSvc: svc,
		log:        log,
	}
}

// InitiatePayment creates a payment via the payment service.
func (a *PaymentServiceAdapter) InitiatePayment(
	ctx context.Context,
	input service.PaymentInput,
) (*service.PaymentResult, error) {
	if a.log != nil {
		a.log.Info("initiating payment for booking", "booking_id", input.BookingID, "amount", input.Amount, "currency", input.Currency)
	}

	// Map to payment domain input
	paymentInput := paymentdomain.CreatePaymentInput{
		BookingID:       &input.BookingID,
		Amount:          input.Amount,
		Currency:        payment.Currency(input.Currency),
		PayerID:         input.PayerID,
		PayerEmail:      input.PayerEmail,
		PayerName:       input.PayerName,
		PaymentMethodID: input.PaymentMethodID,
		CallbackURL:     input.CallbackURL,
		Description:     input.Description,
		Market:          a.detectMarket(input.Currency),
	}

	// Call payment service
	payment, err := a.paymentSvc.CreatePayment(ctx, paymentInput)
	if err != nil {
		if a.log != nil {
			a.log.Error("payment creation failed", "error", err)
		}
		return nil, err
	}

	if a.log != nil {
		a.log.Info("payment created", "payment_id", payment.ID, "status", payment.Status)
	}

	// Map to booking's PaymentResult
	result := &service.PaymentResult{
		PaymentID: payment.ID,
		Status:    string(payment.Status),
		Reference: payment.Reference,
	}

	// Add redirect URL if present
	if payment.RedirectURL != nil {
		result.AuthorizationURL = payment.RedirectURL
	}

	return result, nil
}

// VerifyPayment retrieves payment status from the payment service.
func (a *PaymentServiceAdapter) VerifyPayment(
	ctx context.Context,
	paymentID uuid.UUID,
) (*service.PaymentStatus, error) {
	if a.log != nil {
		a.log.Info("verifying payment", "payment_id", paymentID)
	}

	payment, err := a.paymentSvc.GetPayment(ctx, paymentID)
	if err != nil {
		if a.log != nil {
			a.log.Error("failed to get payment", "payment_id", paymentID, "error", err)
		}
		return nil, err
	}

	return &service.PaymentStatus{
		PaymentID: payment.ID,
		Status:    string(payment.Status),
		Amount:    payment.Amount,
	}, nil
}

// RefundPayment processes a refund via the payment service.
func (a *PaymentServiceAdapter) RefundPayment(
	ctx context.Context,
	input service.RefundPaymentInput,
) (*service.RefundResult, error) {
	if a.log != nil {
		a.log.Info("processing refund for payment", "payment_id", input.PaymentID)
	}

	// Map to payment domain input
	refundInput := paymentdomain.RefundPaymentInput{
		PaymentID:  input.PaymentID,
		Amount:     input.Amount,
		Reason:     input.Reason,
		RefundedBy: input.RefundedBy,
	}

	// Call payment service
	refundedPayment, err := a.paymentSvc.RefundPayment(ctx, refundInput)
	if err != nil {
		if a.log != nil {
			a.log.Error("refund failed", "error", err)
		}
		return nil, err
	}

	if a.log != nil {
		a.log.Info("refund processed", "payment_id", refundedPayment.ID, "refunded_amount", refundedPayment.RefundedAmount)
	}

	// Map to booking's RefundResult
	result := &service.RefundResult{
		RefundID:    refundedPayment.ID.String(), // Using payment ID as refund reference
		Status:      string(refundedPayment.Status),
		Amount:      refundedPayment.RefundedAmount,
		PaymentID:   refundedPayment.ID,
		BookingID:   refundedPayment.BookingID,
		RefundedBy:  input.RefundedBy,
		ProcessedAt: refundedPayment.UpdatedAt,
	}

	if refundedPayment.RefundedAt != nil {
		result.RefundedAt = *refundedPayment.RefundedAt
	}

	return result, nil
}

// GetDefaultPaymentMethodID returns the default payment method ID for a user, if any.
func (a *PaymentServiceAdapter) GetDefaultPaymentMethodID(ctx context.Context, userID uuid.UUID) (*uuid.UUID, error) {
	method, err := a.paymentSvc.GetDefaultPaymentMethod(ctx, userID)
	if err != nil {
		if a.log != nil {
			a.log.Error("failed to get default payment method for user", "user_id", userID, "error", err)
		}
		return nil, err
	}
	if method == nil {
		return nil, nil
	}
	return &method.ID, nil
}

// detectMarket infers market from currency code
func (a *PaymentServiceAdapter) detectMarket(currency string) paymentdomain.Market {
	switch currency {
	case "NGN":
		return paymentdomain.MarketNigeria
	case "GHS":
		return paymentdomain.MarketGhana
	case "KES":
		return paymentdomain.MarketKenya
	case "ZAR":
		return paymentdomain.MarketSouthAfrica
	default:
		return paymentdomain.MarketOther
	}
}
