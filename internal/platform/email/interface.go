package email

import "context"

// Sender is the contract for any email provider (SMTP, Resend, Mailgun)
type Sender interface {
	// SendHtml sends an email with HTML content
	SendHtml(ctx context.Context, to, subject, htmlBody string) error
}
