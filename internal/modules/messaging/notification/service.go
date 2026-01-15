package notification

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"hauslet/internal/modules/messaging/domain"
	messagingtemplates "hauslet/internal/modules/messaging/templates"
	"hauslet/internal/platform/email"
	"hauslet/internal/platform/queue"
	emailJob "hauslet/internal/queue/jobs/emails"

	"github.com/google/uuid"
)

// NotificationService handles messaging-module notifications.
type NotificationService struct {
	mailClient   *email.Client
	queueClient  *queue.Client
	queueSubject string
	baseURL      string
	log          *slog.Logger
}

// NewNotificationService wires the dependencies needed for messaging notifications.
func NewNotificationService(
	mailClient *email.Client,
	queueClient *queue.Client,
	queueSubject string,
	baseURL string,
	log *slog.Logger,
) *NotificationService {
	return &NotificationService{
		mailClient:   mailClient,
		queueClient:  queueClient,
		queueSubject: queueSubject,
		baseURL:      strings.TrimRight(baseURL, "/"),
		log:          log,
	}
}

// SendNewMessageNotification notifies a recipient about a new message.
func (s *NotificationService) SendNewMessageNotification(
	ctx context.Context,
	conversationID uuid.UUID,
	recipientName string,
	recipientEmail string,
	senderName string,
	messageContent string,
) {
	if recipientEmail == "" {
		return
	}

	subject := fmt.Sprintf("New message from %s", senderName)
	preview := truncateMessage(messageContent, 100)

	data := map[string]any{
		"Subject":         subject,
		"Preview":         preview,
		"Year":            time.Now().Year(),
		"RecipientName":   fallbackName(recipientName),
		"SenderName":      fallbackName(senderName),
		"MessagePreview":  preview,
		"ConversationURL": s.conversationURL(conversationID),
	}

	s.renderAndSend(ctx, "new_message.html", recipientEmail, subject, data, "send new message notification")
}

// NotifyRecipients sends notifications to all recipients of a message.
func (s *NotificationService) NotifyRecipients(
	ctx context.Context,
	conversation *domain.Conversation,
	message *domain.Message,
	senderContact *domain.UserContact,
	getRecipientContact func(ctx context.Context, userID uuid.UUID) (*domain.UserContact, error),
) {
	if conversation == nil || message == nil {
		return
	}

	senderName := "Someone"
	if senderContact != nil && senderContact.Name != "" {
		senderName = senderContact.Name
	}

	for _, participant := range conversation.Participants {
		// Skip the sender
		if participant.UserID == message.SenderID {
			continue
		}

		// Skip muted participants
		if participant.IsMuted {
			continue
		}

		// Skip non-visible participants (shadow agents)
		if !participant.IsVisible {
			continue
		}

		// Get recipient contact info
		recipientContact, err := getRecipientContact(ctx, participant.UserID)
		if err != nil {
			if s.log != nil {
				s.log.Warn("failed to get recipient contact", "user_id", participant.UserID, "error", err)
			}
			continue
		}
		if recipientContact == nil || recipientContact.Email == "" {
			continue
		}

		s.SendNewMessageNotification(
			ctx,
			conversation.ID,
			recipientContact.Name,
			recipientContact.Email,
			senderName,
			message.Content,
		)
	}
}

// sendEmailAsync runs the provided send function in a goroutine and logs errors.
func (s *NotificationService) sendEmailAsync(label string, fn func() error) {
	go func() {
		if err := fn(); err != nil && s.log != nil {
			s.log.Warn("send email async error", "label", label, "error", err)
		}
	}()
}

// publishEmailJob tries to enqueue the email job and returns an error on failure.
func (s *NotificationService) publishEmailJob(job emailJob.EmailJob) error {
	if s.queueClient == nil || s.queueSubject == "" {
		return fmt.Errorf("queue not configured")
	}

	pubCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.queueClient.Publish(pubCtx, s.queueSubject, job); err != nil {
		if s.log != nil {
			s.log.Warn("failed to publish messaging email job", "queue_subject", s.queueSubject, "error", err)
		}
		return err
	}

	return nil
}

func (s *NotificationService) renderAndSend(ctx context.Context, templateName, to, subject string, data map[string]any, label string) {
	htmlBody, err := s.mailClient.RenderTemplate(messagingtemplates.FS, templateName, data)
	if err != nil {
		if s.log != nil {
			s.log.Error("failed to render template", "template", templateName, "error", err)
		}
		return
	}

	s.sendEmailAsync(label, func() error {
		job := emailJob.EmailJob{
			To:      to,
			Subject: subject,
			HTML:    htmlBody,
		}
		if err := s.publishEmailJob(job); err == nil {
			return nil
		} else if s.queueClient != nil && !s.queueClient.AllowFallback() {
			return err
		}
		return s.mailClient.SendHTML(ctx, to, subject, htmlBody)
	})
}

func (s *NotificationService) conversationURL(conversationID uuid.UUID) string {
	return fmt.Sprintf("%s/messages/%s", s.baseURL, conversationID)
}

func fallbackName(name string) string {
	if strings.TrimSpace(name) == "" {
		return "there"
	}
	return name
}

func truncateMessage(content string, maxLen int) string {
	content = strings.TrimSpace(content)
	if len(content) <= maxLen {
		return content
	}
	return content[:maxLen-3] + "..."
}
