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
