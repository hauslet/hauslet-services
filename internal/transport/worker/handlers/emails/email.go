package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"hauslet/internal/platform/email"
	emailJob "hauslet/internal/queue/jobs/emails"
)

// EmailHandler handles email sending jobs
type EmailHandler struct {
	client  *email.Client
	log     *slog.Logger
	subject string
}

// NewEmailHandler creates a new email handler
func NewEmailHandler(client *email.Client, log *slog.Logger, subject string) *EmailHandler {
	return &EmailHandler{
		client:  client,
		log:     log,
		subject: subject,
	}
}

// JobType returns the job type this handler processes
func (h *EmailHandler) JobType() string {
	return emailJob.EmailJobType
}

// Subject returns the NATS subject this handler listens to
func (h *EmailHandler) Subject() string {
	return h.subject
}

// Handle processes an email job
func (h *EmailHandler) Handle(ctx context.Context, data []byte) error {
	var job emailJob.EmailJob
	if err := json.Unmarshal(data, &job); err != nil {
		return fmt.Errorf("failed to unmarshal email job: %w", err)
	}

	// Validate job
	if err := job.Validate(); err != nil {
		return fmt.Errorf("invalid email job: %w", err)
	}

	h.log.Info("Sending email", "to", job.To, "subject", job.Subject)

	// Send email
	if err := h.client.SendHTML(ctx, job.To, job.Subject, job.HTML); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	h.log.Info("✅ Email sent successfully", slog.String("to", job.To))
	return nil
}
