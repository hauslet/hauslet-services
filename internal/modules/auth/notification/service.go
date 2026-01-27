package notification

import (
	"hauslet/internal/platform/email"
	"hauslet/internal/platform/queue"
	"hauslet/internal/platform/sms"
	"log/slog"
)

// NotificationService handles auth-module notifications (email and SMS).
type NotificationService struct {
	mailClient      *email.Client
	smsClient       *sms.Client
	queueClient     *queue.Client
	emailSubject    string // NATS subject for email queue
	smsQueueSubject string // NATS subject for SMS queue
	log             *slog.Logger
}

// NewNotificationService wires the dependencies needed for auth notifications.
func NewNotificationService(
	mailClient *email.Client,
	smsClient *sms.Client,
	queueClient *queue.Client,
	emailSubject string,
	smsQueueSubject string,
	log *slog.Logger,
) *NotificationService {
	return &NotificationService{
		mailClient:      mailClient,
		smsClient:       smsClient,
		queueClient:     queueClient,
		emailSubject:    emailSubject,
		smsQueueSubject: smsQueueSubject,
		log:             log,
	}
}
