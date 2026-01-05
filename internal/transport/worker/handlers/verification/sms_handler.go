package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"hauslet/internal/platform/sms"
	verificationJob "hauslet/internal/queue/jobs/verification"
)

type SMSHandler struct {
	smsClient *sms.Client
	log       *slog.Logger
	subject   string
}

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

func (h *SMSHandler) JobType() string {
	return verificationJob.SMSJobType
}

func (h *SMSHandler) Subject() string {
	return h.subject
}

func (h *SMSHandler) Handle(ctx context.Context, data []byte) error {
	var job verificationJob.SMSJob
	if err := json.Unmarshal(data, &job); err != nil {
		return fmt.Errorf("unmarshal sms job: %w", err)
	}

	if err := job.Validate(); err != nil {
		return fmt.Errorf("invalid sms job: %w", err)
	}

	h.log.Info("sending verification SMS",
		"phone_number", job.PhoneNumber,
		"provider", job.Provider,
		"retry_count", job.RetryCount,
	)

	message := fmt.Sprintf("Your Hauslet verification code is: %s", job.OTP)

	req := sms.SMSRequest{
		To:      job.PhoneNumber,
		Message: message,
	}

	resp, err := h.smsClient.Send(ctx, req)
	if err != nil {
		h.log.Error("failed to send SMS",
			"phone_number", job.PhoneNumber,
			"provider", job.Provider,
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
