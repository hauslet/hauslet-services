package email

import (
	"context"

	"github.com/go-pkgz/email"
)

type SMTPAdapter struct {
	client *email.Sender
	from   string
}

// NewSMTPAdapter creates a sender that uses standard SMTP
func NewSMTPAdapter(host string, port int, user, pass, from string) *SMTPAdapter {
	opts := []email.Option{
		email.Auth(user, pass),
		email.Port(port),
		email.ContentType("text/html"),
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
