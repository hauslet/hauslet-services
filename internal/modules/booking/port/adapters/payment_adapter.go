package adapters

import (
	"context"
	"hauslet/internal/modules/booking/service"
	paymentdomain "hauslet/internal/modules/payments/domain"
	paymentsservice "hauslet/internal/modules/payments/service"
	"hauslet/internal/platform/payment"

	"github.com/go-pkgz/lgr"
	"github.com/google/uuid"
)

// PaymentServiceAdapter adapts the payment service for booking's PaymentGateway interface.
type PaymentServiceAdapter struct {
	paymentSvc paymentsservice.PaymentService
	log        *lgr.Logger
}

// NewPaymentServiceAdapter creates a new payment service adapter.
func NewPaymentServiceAdapter(svc paymentsservice.PaymentService, log *lgr.Logger) service.PaymentGateway {
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
		a.log.Logf("INFO initiating payment for booking=%s amount=%d %s",
			input.BookingID, input.Amount, input.Currency)
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
			a.log.Logf("ERROR payment creation failed: %v", err)
		}
		return nil, err
	}

	if a.log != nil {
		a.log.Logf("INFO payment created: id=%s status=%s", payment.ID, payment.Status)
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
		a.log.Logf("INFO verifying payment: id=%s", paymentID)
	}

	payment, err := a.paymentSvc.GetPayment(ctx, paymentID)
	if err != nil {
		if a.log != nil {
			a.log.Logf("ERROR failed to get payment %s: %v", paymentID, err)
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
		a.log.Logf("INFO processing refund for payment=%s", input.PaymentID)
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
			a.log.Logf("ERROR refund failed: %v", err)
		}
		return nil, err
	}

	if a.log != nil {
		a.log.Logf("INFO refund processed: payment=%s refunded_amount=%d",
			refundedPayment.ID, refundedPayment.RefundedAmount)
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
			a.log.Logf("ERROR failed to get default payment method for user=%s: %v", userID, err)
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
