package email

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/go-pkgz/email"
)

type SMTPAdapter struct {
	client *email.Sender
	from   string
}

// slogAdapter wraps *slog.Logger to implement go-pkgz/auth/logger.L interface.
type slogAdapter struct {
	logger *slog.Logger
}

// Logf implements the logger.L interface required by go-pkgz/auth.
func (a *slogAdapter) Logf(format string, args ...interface{}) {
	a.logger.Info(fmt.Sprintf(format, args...))
}

// newSlogAdapter creates a logger adapter from *slog.Logger.
func newSlogAdapter(logger *slog.Logger) *slogAdapter {
	return &slogAdapter{logger: logger}
}

// NewSMTPAdapter creates a sender that uses standard SMTP
func NewSMTPAdapter(host string, port int, user, pass, from string, log *slog.Logger) *SMTPAdapter {
	opts := []email.Option{
		email.Auth(user, pass),
		email.Port(port),
		email.ContentType("text/html"),
		email.Log(newSlogAdapter(log)),
	}

	// Use STARTTLS on submission ports (e.g., 587); fall back to implicit TLS on 465.
	if port == 465 {
		opts = append(opts, email.TLS(true))
	} else {
		opts = append(opts, email.STARTTLS(true))
	}

	sender := email.NewSender(host, opts...)

	return &SMTPAdapter{
		client: sender,
		from:   from,
	}
}

// SendHtml implements the Sender interface
func (s *SMTPAdapter) SendHtml(ctx context.Context, to, subject, htmlBody string) error {
	subj := "[Hauslet] " + subject
	return s.client.Send(htmlBody, email.Params{
		From:    s.from,
		To:      []string{to},
		Subject: subj,
	})
}

// SendHtmlWithAttachments sends an email with attachments via SMTP.
func (s *SMTPAdapter) SendHtmlWithAttachments(ctx context.Context, to, subject, htmlBody string, attachments []Attachment) error {
	subj := "[Hauslet] " + subject
	params := email.Params{
		From:    s.from,
		To:      []string{to},
		Subject: subj,
	}

	if len(attachments) == 0 {
		return s.client.Send(htmlBody, params)
	}

	tempDir, err := os.MkdirTemp("", "hauslet-email-")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	paths := make([]string, 0, len(attachments))
	for _, attachment := range attachments {
		filename := attachment.Filename
		if filename == "" {
			filename = "attachment"
		}
		path := filepath.Join(tempDir, filename)
		if err := os.WriteFile(path, attachment.Content, 0o600); err != nil {
			return fmt.Errorf("write attachment: %w", err)
		}
		paths = append(paths, path)
	}

	params.Attachments = paths
	return s.client.Send(htmlBody, params)
}
