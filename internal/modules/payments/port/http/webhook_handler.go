package http

import (
	"context"
	paymentdomain "hauslet/internal/modules/payments/domain"
	"hauslet/internal/modules/payments/service"
	"hauslet/internal/platform/payment"
	platformQueue "hauslet/internal/platform/queue"
	"log/slog"

	"github.com/google/uuid"
)

// BookingHooks defines the interface for booking lifecycle callbacks.
type BookingHooks interface {
	OnPaymentSucceeded(ctx context.Context, bookingID uuid.UUID, payment *paymentdomain.Payment) error
	OnPaymentFailed(ctx context.Context, bookingID uuid.UUID, payment *paymentdomain.Payment, reason string) error
	OnPaymentRefunded(ctx context.Context, bookingID uuid.UUID, payment *paymentdomain.Payment) error
}

// FinanceHooks defines the interface for finance module callbacks
type FinanceHooks interface {
	OnPaymentSucceeded(ctx context.Context, bookingID, paymentID uuid.UUID, amount int64, currency string) error
	OnRefundProcessed(ctx context.Context, bookingID, paymentID uuid.UUID, amount int64, currency string) error
	OnBookingCompleted(ctx context.Context, bookingID, hostID uuid.UUID) error
}

// PayoutHooks defines the interface for payout/disbursement callbacks
type PayoutHooks interface {
	OnTransferSuccess(ctx context.Context, transferCode string) error
	OnTransferFailed(ctx context.Context, transferCode string, reason string) error
}

// PromotionHooks defines the interface for promotion/subscription lifecycle callbacks
type PromotionHooks interface {
	OnPromotionPaymentSucceeded(ctx context.Context, promotionID uuid.UUID, payment *paymentdomain.Payment) error
	OnSubscriptionPaymentSucceeded(ctx context.Context, subscriptionID uuid.UUID, payment *paymentdomain.Payment) error
}

// WebhookHandler handles payment provider webhooks
type WebhookHandler struct {
	paymentService service.PaymentService
	paymentClient  *payment.Client
	bookingHooks   BookingHooks
	financeHooks   FinanceHooks
	payoutHooks    PayoutHooks
	promotionHooks PromotionHooks
	queueClient    *platformQueue.Client
	queueSubject   string
	log            *slog.Logger
}

// NewWebhookHandler creates a new webhook handler
func NewWebhookHandler(
	paymentService service.PaymentService,
	paymentClient *payment.Client,
	bookingHooks BookingHooks,
	financeHooks FinanceHooks,
	payoutHooks PayoutHooks,
	promotionHooks PromotionHooks,
	queueClient *platformQueue.Client,
	queueSubject string,
	log *slog.Logger,
) *WebhookHandler {
	return &WebhookHandler{
		paymentService: paymentService,
		paymentClient:  paymentClient,
		bookingHooks:   bookingHooks,
		financeHooks:   financeHooks,
		payoutHooks:    payoutHooks,
		promotionHooks: promotionHooks,
		queueClient:    queueClient,
		queueSubject:   queueSubject,
		log:            log,
	}
}
