package notification

import (
	"context"
	"fmt"
	propertytemplates "hauslet/internal/modules/property/templates"
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

// publishEmailJob tries to enqueue the email job and returns an error on failure.
// It uses a short-lived background context so request cancellation does not
// prevent publishing.
func (s *NotificationService) publishEmailJob(job emailJob.EmailJob) error {
	if s.queueClient == nil || s.queueSubject == "" {
		return fmt.Errorf("queue not configured")
	}

	pubCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.queueClient.Publish(pubCtx, s.queueSubject, job); err != nil {
		if s.log != nil {
			s.log.Logf("[WARN] failed to publish business email job to %s: %v", s.queueSubject, err)
		}
		return err
	}

	return nil
}

// SendPublishListingRequestNotification notifies the listing owner that their listing is under review.
func (s *NotificationService) SendPublishListingRequestNotification(ctx context.Context, listingTitle, receipientName, recipientEmail string) error {
	subject := "Your listing is under review on Hauslet"
	preview := "We’re reviewing your listing and will notify you once it’s approved."

	emailData := map[string]any{
		"OwnerName":    receipientName,
		"ListingTitle": listingTitle,
		"ListingURL:":  s.baseURL + "/listings", // Link to listings page

		// Required for the Layout
		"Subject": subject,
		"Preview": preview,
		"Year":    time.Now().Year(),
	}

	htmlBody, err := s.mailClient.RenderTemplate(
		propertytemplates.FS,
		"listing_moderation_pending.html",
		emailData,
	)
	if err != nil {
		return err
	}
	s.sendEmailAsync("send publish listing request notification email", func() error {
		job := emailJob.EmailJob{
			To:      recipientEmail,
			Subject: subject,
			HTML:    htmlBody,
		}
		if err := s.publishEmailJob(job); err == nil {
			return nil
		} else if s.queueClient != nil && !s.queueClient.AllowFallback() {
			return err
		}
		return s.mailClient.SendHTML(ctx, recipientEmail, subject, htmlBody)
	})
	return nil
}

// SendListingAcceptedNotification notifies the listing owner that their listing has been approved.
func (s *NotificationService) SendListingAcceptedNotification(ctx context.Context, listingTitle, receipientName, recipientEmail string) error {
	subject := "Your listing has been approved on Hauslet"
	preview := "Congratulations! Your listing is now live on Hauslet."

	emailData := map[string]any{
		"OwnerName":    receipientName,
		"ListingTitle": listingTitle,
		"ListingURL:":  s.baseURL + "/listings", // Link to listings page

		// Required for the Layout
		"Subject": subject,
		"Preview": preview,
		"Year":    time.Now().Year(),
	}
	htmlBody, err := s.mailClient.RenderTemplate(
		propertytemplates.FS,
		"listing_moderation_approved.html",
		emailData,
	)
	if err != nil {
		return err
	}
	s.sendEmailAsync("send publish listing approved notification email", func() error {
		// Send directly no queuing as this will be called by a worker
		return s.mailClient.SendHTML(ctx, recipientEmail, subject, htmlBody)
	})
	return nil
}

// SendListingRejectedNotification notifies the listing owner that their listing has been rejected.
func (s *NotificationService) SendListingRejectedNotification(ctx context.Context, listingTitle, toName, toEmail string, reasons []string) error {
	subject := "Your listing has been rejected on Hauslet"
	preview := "We're sorry to inform you that your listing has been rejected."

	emailData := map[string]any{
		"OwnerName":        toName,
		"ListingTitle":     listingTitle,
		"RejectionReasons": reasons,
		"ListingURL:":      s.baseURL + "/listings", // Link to listings page

		// Required for the Layout
		"Subject": subject,
		"Preview": preview,
		"Year":    time.Now().Year(),
	}
	htmlBody, err := s.mailClient.RenderTemplate(
		propertytemplates.FS,
		"listing_moderation_rejected.html",
		emailData,
	)
	if err != nil {
		return err
	}
	s.sendEmailAsync("send publish listing rejected notification email", func() error {
		// Send directly no queuing as this will be called by a worker
		return s.mailClient.SendHTML(ctx, toEmail, subject, htmlBody)
	})
	return nil
}
