package hooks

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	paymentDomain "hauslet/internal/modules/payments/domain"
	"hauslet/internal/modules/promotions/service"
)

// PaymentHooksImpl implements payment webhook hooks for promotions and subscriptions
type PaymentHooksImpl struct {
	promotionSvc    service.PromotionService
	subscriptionSvc service.SubscriptionService
	log             *slog.Logger
}

// NewPaymentHooks creates a new payment hooks implementation
func NewPaymentHooks(
	promotionSvc service.PromotionService,
	subscriptionSvc service.SubscriptionService,
	log *slog.Logger,
) *PaymentHooksImpl {
	return &PaymentHooksImpl{
		promotionSvc:    promotionSvc,
		subscriptionSvc: subscriptionSvc,
		log:             log,
	}
}

// OnPromotionPaymentSucceeded activates a promotion after successful payment
func (h *PaymentHooksImpl) OnPromotionPaymentSucceeded(ctx context.Context, promotionID uuid.UUID, payment *paymentDomain.Payment) error {
	h.log.Info("processing promotion payment success webhook",
		"promotion_id", promotionID,
		"payment_id", payment.ID,
		"amount", payment.Amount,
	)

	// Start the promotion
	if err := h.promotionSvc.StartPromotion(ctx, promotionID); err != nil {
		return fmt.Errorf("failed to start promotion: %w", err)
	}

	h.log.Info("promotion activated successfully",
		"promotion_id", promotionID,
		"payment_id", payment.ID,
	)

	return nil
}

// OnSubscriptionPaymentSucceeded activates or renews a subscription after successful payment
func (h *PaymentHooksImpl) OnSubscriptionPaymentSucceeded(ctx context.Context, subscriptionID uuid.UUID, payment *paymentDomain.Payment) error {
	h.log.Info("processing subscription payment success webhook",
		"subscription_id", subscriptionID,
		"payment_id", payment.ID,
		"amount", payment.Amount,
	)

	// Renew the subscription
	if err := h.subscriptionSvc.RenewSubscription(ctx, subscriptionID, payment.ID); err != nil {
		return fmt.Errorf("failed to renew subscription: %w", err)
	}

	h.log.Info("subscription renewed successfully",
		"subscription_id", subscriptionID,
		"payment_id", payment.ID,
	)

	return nil
}
