package email

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"io/fs"
)

// Client is the high-level service used by modules
type Client struct {
	sender   Sender // <--- This is the Interface!
	layoutFS fs.FS
}

// New creates a Client with a specific backend
func New(sender Sender) *Client {
	return &Client{
		sender:   sender,
		layoutFS: layoutFS,
	}
}

// SendTemplate parses the template and hands the HTML to the sender
func (c *Client) SendTemplate(ctx context.Context, to string, subject string, fsys fs.FS, templateName string, data interface{}) error {
	htmlBody, err := c.RenderTemplate(fsys, templateName, data)
	if err != nil {
		return err
	}

	// 3. Send using whatever adapter was injected (SMTP or Resend)
	return c.sender.SendHtml(ctx, to, subject, htmlBody)
}

// SendHTML sends a pre-rendered HTML body.
func (c *Client) SendHTML(ctx context.Context, to, subject, htmlBody string) error {
	return c.sender.SendHtml(ctx, to, subject, htmlBody)
}

// SendHTMLWithAttachments sends HTML with optional attachments when supported.
func (c *Client) SendHTMLWithAttachments(ctx context.Context, to, subject, htmlBody string, attachments []Attachment) error {
	if len(attachments) == 0 {
		return c.sender.SendHtml(ctx, to, subject, htmlBody)
	}

	senderWithAttachments, ok := c.sender.(SenderWithAttachments)
	if !ok {
		return fmt.Errorf("email sender does not support attachments")
	}

	return senderWithAttachments.SendHtmlWithAttachments(ctx, to, subject, htmlBody, attachments)
}

// RenderTemplate builds the HTML string without sending.
func (c *Client) RenderTemplate(fsys fs.FS, templateName string, data interface{}) (string, error) {
	if c.layoutFS == nil {
		return "", fmt.Errorf("email client missing layout templates")
	}

	// Parse shared layout first, then module-specific template into the same set.
	tmpl, err := template.New("layout").ParseFS(c.layoutFS, "templates/layout.html")
	if err != nil {
		return "", fmt.Errorf("failed to parse layout template: %w", err)
	}

	if _, err := tmpl.ParseFS(fsys, templateName); err != nil {
		return "", fmt.Errorf("failed to parse module template: %w", err)
	}

	// 2. Execute Template
	var body bytes.Buffer
	if err := tmpl.Execute(&body, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return body.String(), nil
}
