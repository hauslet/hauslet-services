package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"hauslet/internal/modules/promotions/service"
	promotionJob "hauslet/internal/queue/jobs/promotions"
)

// SubscriptionBillingHandler processes subscription billing for due subscriptions.
type SubscriptionBillingHandler struct {
	subscriptionSvc service.SubscriptionService
	log             *slog.Logger
	subject         string
}

// NewSubscriptionBillingHandler constructs a subscription billing handler.
func NewSubscriptionBillingHandler(subscriptionSvc service.SubscriptionService, log *slog.Logger, subject string) *SubscriptionBillingHandler {
	return &SubscriptionBillingHandler{
		subscriptionSvc: subscriptionSvc,
		log:             log,
		subject:         subject,
	}
}

func (h *SubscriptionBillingHandler) JobType() string {
	return promotionJob.SubscriptionBillingJobType
}

func (h *SubscriptionBillingHandler) Subject() string {
	return h.subject
}

func (h *SubscriptionBillingHandler) Handle(ctx context.Context, data []byte) error {
	var job promotionJob.SubscriptionBillingJob
	if err := json.Unmarshal(data, &job); err != nil {
		return fmt.Errorf("unmarshal subscription billing job: %w", err)
	}
	if err := job.Validate(); err != nil {
		return fmt.Errorf("invalid subscription billing job: %w", err)
	}

	checkTime := job.GetCheckTime()
	h.log.Info("processing subscription billing", "check_time", checkTime)

	// Process billing
	if err := h.subscriptionSvc.ProcessBilling(ctx); err != nil {
		return fmt.Errorf("failed to process subscription billing: %w", err)
	}

	h.log.Info("subscription billing processing completed")
	return nil
}
