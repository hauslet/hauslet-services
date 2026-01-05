package handlers

import (
	"context"
	"encoding/base64"
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

	attachments := make([]email.Attachment, 0, len(job.Attachments))
	for _, attachment := range job.Attachments {
		// Debug: log first 50 chars of ContentBase64
		previewLen := 50
		previewLen = min(previewLen, len(attachment.ContentBase64))
		h.log.Info("Processing attachment",
			"filename", attachment.Filename,
			"content_length", len(attachment.ContentBase64),
			"content_preview", attachment.ContentBase64[:previewLen],
		)

		payload, err := base64.StdEncoding.DecodeString(attachment.ContentBase64)
		if err != nil {
			h.log.Error("Failed to decode attachment",
				"filename", attachment.Filename,
				"content_preview", attachment.ContentBase64[:previewLen],
				"error", err,
			)
			return fmt.Errorf("failed to decode attachment %s: %w", attachment.Filename, err)
		}
		attachments = append(attachments, email.Attachment{
			Filename:    attachment.Filename,
			ContentType: attachment.ContentType,
			Content:     payload,
		})
	}

	// Send email
	if len(attachments) == 0 {
		if err := h.client.SendHTML(ctx, job.To, job.Subject, job.HTML); err != nil {
			return fmt.Errorf("failed to send email: %w", err)
		}
	} else {
		if err := h.client.SendHTMLWithAttachments(ctx, job.To, job.Subject, job.HTML, attachments); err != nil {
			return fmt.Errorf("failed to send email with attachments: %w", err)
		}
	}

	h.log.Info("✅ Email sent successfully", slog.String("to", job.To))
	return nil
}
