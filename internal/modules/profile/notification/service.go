package notification

import (
	"context"
	profiletemplates "hauslet/internal/modules/profile/templates"
	"hauslet/internal/platform/email"
	"hauslet/internal/platform/queue"
	emailJob "hauslet/internal/queue/jobs/emails"
	"time"

	"github.com/go-pkgz/lgr"
)

type NotificationService struct {
	mailClient   *email.Client
	queueClient  *queue.Client
	queueSubject string
	baseURL      string
	log          *lgr.Logger
}

func NewNotificationService(
	mailClient *email.Client,
	queueClient *queue.Client,
	queueSubject string,
	baseURL string,
	logger *lgr.Logger) *NotificationService {
	return &NotificationService{
		mailClient:   mailClient,
		queueClient:  queueClient,
		queueSubject: queueSubject,
		baseURL:      baseURL,
		log:          logger,
	}
}

// sendEmailAsync runs the provided send function in a goroutine and logs errors
func (s *NotificationService) sendEmailAsync(label string, fn func() error) {
	go func() {
		if err := fn(); err != nil && s.log != nil {
			s.log.Logf("[WARN] %s: %v", label, err)
		}
	}()
}

// publishEmailJob tries to enqueue the email job and returns true on success.
// It uses a short-lived background context so request cancellation does not
// prevent publishing. On failure, it logs a warning and callers can fall back
// to direct send.
func (s *NotificationService) publishEmailJob(job emailJob.EmailJob) bool {
	if s.queueClient == nil || s.queueSubject == "" {
		return false
	}

	pubCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.queueClient.Publish(pubCtx, s.queueSubject, job); err != nil {
		if s.log != nil {
			s.log.Logf("[WARN] failed to publish business email job to %s: %v; falling back to direct send", s.queueSubject, err)
		}
		return false
	}

	return true
}

// SendProfileModerationRejectedEmail sends an email to the profile owner notifying them
func (s *NotificationService) SendProfileModerationRejectedEmail(ctx context.Context, toEmail, toName, profileID string, rejectionReasons []string) error {
	subject := "Your profile has been rejected on Hauslet"
	preview := "We're sorry to inform you that your profile has been rejected."

	emailData := map[string]any{
		"UserName":         toName,
		"RejectionReasons": rejectionReasons,
		"ProfileURL":       s.baseURL + "/profiles/" + profileID,

		// Required for the Layout
		"Subject": subject,
		"Preview": preview,
		"Year":    time.Now().Year(),
	}

	htmlBody, err := s.mailClient.RenderTemplate(
		profiletemplates.FS,
		"profile_moderation_rejected.html",
		emailData,
	)
	if err != nil {
		return err
	}

	s.sendEmailAsync("send profile moderation rejection email", func() error {
		// Send directly no queuing as this will be called by a worker
		return s.mailClient.SendHTML(ctx, toEmail, subject, htmlBody)
	})
	return nil
}
