package email

import (
	"context"
	"fmt"

	"github.com/resend/resend-go/v2"
)

type ResendAdapter struct {
	client *resend.Client
	from   string
}

// NewResendAdapter creates a sender that uses Resend API
func NewResendAdapter(apiKey, from string) *ResendAdapter {
	client := resend.NewClient(apiKey)
	return &ResendAdapter{
		client: client,
		from:   from,
	}
}

// SendHtml implements the Sender interface
func (r *ResendAdapter) SendHtml(ctx context.Context, to, subject, htmlBody string) error {
	subj := "[Hauslet] " + subject
	params := &resend.SendEmailRequest{
		From:    r.from,
		To:      []string{to},
		Subject: subj,
		Html:    htmlBody,
	}

	_, err := r.client.Emails.SendWithContext(ctx, params)
	if err != nil {
		return fmt.Errorf("resend failed: %w", err)
	}
	return nil
}

// SendHtmlWithAttachments sends an email with attachments using Resend API.
func (r *ResendAdapter) SendHtmlWithAttachments(ctx context.Context, to, subject, htmlBody string, attachments []Attachment) error {
	subj := "[Hauslet] " + subject
	params := &resend.SendEmailRequest{
		From:    r.from,
		To:      []string{to},
		Subject: subj,
		Html:    htmlBody,
	}

	if len(attachments) > 0 {
		params.Attachments = make([]*resend.Attachment, 0, len(attachments))
		for _, attachment := range attachments {
			params.Attachments = append(params.Attachments, &resend.Attachment{
				Content:     attachment.Content,
				Filename:    attachment.Filename,
				ContentType: attachment.ContentType,
			})
		}
	}

	if _, err := r.client.Emails.SendWithContext(ctx, params); err != nil {
		return fmt.Errorf("resend failed: %w", err)
	}
	return nil
}
