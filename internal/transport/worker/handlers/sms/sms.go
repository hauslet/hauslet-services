package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"hauslet/internal/platform/sms"
	smsJob "hauslet/internal/queue/jobs/sms"
)

// SMSHandler handles SMS sending jobs
type SMSHandler struct {
	smsClient *sms.Client
	log       *slog.Logger
	subject   string
}

// NewSMSHandler creates a new SMS handler
func NewSMSHandler(
	smsClient *sms.Client,
	log *slog.Logger,
	subject string,
) *SMSHandler {
	return &SMSHandler{
		smsClient: smsClient,
		log:       log,
		subject:   subject,
	}
}

// JobType returns the job type this handler processes
func (h *SMSHandler) JobType() string {
	return smsJob.SMSJobType
}

// Subject returns the NATS subject this handler listens to
func (h *SMSHandler) Subject() string {
	return h.subject
}

// Handle processes an SMS job
func (h *SMSHandler) Handle(ctx context.Context, data []byte) error {
	var job smsJob.SMSJob
	if err := json.Unmarshal(data, &job); err != nil {
		return fmt.Errorf("unmarshal sms job: %w", err)
	}

	// Validate job
	if err := job.Validate(); err != nil {
		return fmt.Errorf("invalid sms job: %w", err)
	}

	h.log.Info("sending SMS",
		"phone_number", job.PhoneNumber,
		"provider", job.Provider,
	)

	req := sms.SMSRequest{
		To:      job.PhoneNumber,
		Message: job.Message,
	}

	resp, err := h.smsClient.Send(ctx, req)
	if err != nil {
		h.log.Error("failed to send SMS",
			"phone_number", job.PhoneNumber,
			"error", err,
		)
		return fmt.Errorf("sms send failed: %w", err)
	}

	h.log.Info("✅ SMS sent successfully",
		"phone_number", job.PhoneNumber,
		"provider", resp.Provider,
		"message_id", resp.MessageID,
	)

	return nil
}
