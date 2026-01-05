package email

import "context"

// Attachment represents a binary email attachment.
type Attachment struct {
	Filename    string
	ContentType string
	Content     []byte
}

// Sender is the contract for any email provider (SMTP, Resend, Mailgun)
type Sender interface {
	// SendHtml sends an email with HTML content
	SendHtml(ctx context.Context, to, subject, htmlBody string) error
}

// SenderWithAttachments extends Sender to support attachments.
type SenderWithAttachments interface {
	SendHtmlWithAttachments(ctx context.Context, to, subject, htmlBody string, attachments []Attachment) error
}
