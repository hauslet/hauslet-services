package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	paymentshttp "hauslet/internal/modules/payments/port/http"
	"hauslet/internal/platform/payment"
	paymentJob "hauslet/internal/queue/jobs/payments"
)

// PaymentWebhookHandler processes queued payment webhooks.
type PaymentWebhookHandler struct {
	paymentClient  *payment.Client
	webhookHandler *paymentshttp.WebhookHandler
	log            *slog.Logger
	subject        string
}

// NewPaymentWebhookHandler constructs a payment webhook handler.
func NewPaymentWebhookHandler(
	paymentClient *payment.Client,
	webhookHandler *paymentshttp.WebhookHandler,
	log *slog.Logger,
	subject string,
) *PaymentWebhookHandler {
	return &PaymentWebhookHandler{
		paymentClient:  paymentClient,
		webhookHandler: webhookHandler,
		log:            log,
		subject:        subject,
	}
}

func (h *PaymentWebhookHandler) JobType() string {
	return paymentJob.PaymentWebhookJobType
}

func (h *PaymentWebhookHandler) Subject() string {
	return h.subject
}

func (h *PaymentWebhookHandler) Handle(ctx context.Context, data []byte) error {
	var job paymentJob.PaymentWebhookJob
	if err := json.Unmarshal(data, &job); err != nil {
		return fmt.Errorf("unmarshal payment webhook job: %w", err)
	}
	if err := job.Validate(); err != nil {
		return fmt.Errorf("invalid payment webhook job: %w", err)
	}

	event, err := h.paymentClient.ParseWebhookEvent(job.Provider, job.Payload)
	if err != nil {
		return fmt.Errorf("parse payment webhook event: %w", err)
	}

	if event.Type == "" {
		event.Type = job.EventType
	}
	if event.Reference == "" {
		event.Reference = job.Reference
	}

	h.log.Info("processing payment webhook job", "type", event.Type, "ref", event.Reference)
	return h.webhookHandler.ProcessEvent(ctx, event)
}
