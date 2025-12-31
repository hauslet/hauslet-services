package notification

import (
	"context"
	"fmt"
	"hauslet/internal/modules/finance/templates"
	"hauslet/internal/platform/email"
	"hauslet/internal/platform/payment"
	"hauslet/internal/platform/queue"
	emailJob "hauslet/internal/queue/jobs/emails"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// NotificationService handles finance-related email notifications
type NotificationService struct {
	mailClient   *email.Client
	queueClient  *queue.Client
	queueSubject string
	baseURL      string
	log          *slog.Logger
}

// NewNotificationService creates a new finance notification service
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
		baseURL:      baseURL,
		log:          log,
	}
}

// sendEmailAsync runs the provided send function in a goroutine and logs errors
func (s *NotificationService) sendEmailAsync(label string, fn func() error) {
	go func() {
		if err := fn(); err != nil && s.log != nil {
			s.log.Warn("%s: %v", label, err)
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
			s.log.Warn("failed to publish finance email job to %s: %v", s.queueSubject, err)
		}
		return err
	}

	return nil
}

// SendPaymentReceipt sends a payment receipt email to the guest
func (s *NotificationService) SendPaymentReceipt(
	ctx context.Context,
	bookingID uuid.UUID,
	guestEmail string,
	guestName string,
	amount int64,
	currency string,
) error {
	if s.mailClient == nil {
		return nil
	}

	subject := "Payment Receipt - Booking Confirmed"
	preview := "Your payment has been successfully processed."

	emailData := map[string]any{
		"BookingID":    bookingID.String(),
		"Amount":       payment.FormatAmount(amount, payment.Currency(currency)),
		"Currency":     currency,
		"GuestName":    guestName,
		"PaidAt":       time.Now().Format("January 2, 2006 at 3:04 PM"),
		"DashboardURL": fmt.Sprintf("%s/bookings/%s", s.baseURL, bookingID),

		// Required for the Layout
		"Subject": subject,
		"Preview": preview,
		"Year":    time.Now().Year(),
	}

	htmlBody, err := s.mailClient.RenderTemplate(
		templates.FS,
		"payment_receipt.html",
		emailData,
	)
	if err != nil {
		if s.log != nil {
			s.log.Error("failed to render payment receipt template: %v", err)
		}
		return nil
	}

	s.sendEmailAsync("send payment receipt email", func() error {
		job := emailJob.EmailJob{
			To:      guestEmail,
			Subject: subject,
			HTML:    htmlBody,
		}
		if err := s.publishEmailJob(job); err == nil {
			return nil
		} else if s.queueClient != nil && !s.queueClient.AllowFallback() {
			return err
		}
		return s.mailClient.SendHTML(ctx, guestEmail, subject, htmlBody)
	})
	return nil
}

// SendPayoutInitiated sends notification when payout processing starts
func (s *NotificationService) SendPayoutInitiated(
	ctx context.Context,
	bookingID uuid.UUID,
	hostEmail string,
	hostName string,
	amount int64,
	currency string,
) error {
	if s.mailClient == nil {
		return nil
	}

	subject := "Payout Processing Started"
	preview := "Your payout is being processed."

	emailData := map[string]any{
		"BookingID":          bookingID.String(),
		"Amount":             payment.FormatAmount(amount, payment.Currency(currency)),
		"Currency":           currency,
		"HostName":           hostName,
		"InitiatedAt":        time.Now().Format("January 2, 2006 at 3:04 PM"),
		"ExpectedCompletion": "1-3 business days",
		"DashboardURL":       fmt.Sprintf("%s/earnings", s.baseURL),

		// Required for the Layout
		"Subject": subject,
		"Preview": preview,
		"Year":    time.Now().Year(),
	}

	htmlBody, err := s.mailClient.RenderTemplate(
		templates.FS,
		"payout_initiated.html",
		emailData,
	)
	if err != nil {
		if s.log != nil {
			s.log.Error("failed to render payout initiated template: %v", err)
		}
		return nil
	}

	s.sendEmailAsync("send payout initiated email", func() error {
		job := emailJob.EmailJob{
			To:      hostEmail,
			Subject: subject,
			HTML:    htmlBody,
		}
		if err := s.publishEmailJob(job); err == nil {
			return nil
		} else if s.queueClient != nil && !s.queueClient.AllowFallback() {
			return err
		}
		return s.mailClient.SendHTML(ctx, hostEmail, subject, htmlBody)
	})
	return nil
}

// SendPayoutSuccess sends notification when payout completes successfully
func (s *NotificationService) SendPayoutSuccess(
	ctx context.Context,
	disbursementID uuid.UUID,
	hostEmail string,
	hostName string,
	amount int64,
	currency string,
	accountName string,
	bankName string,
	maskedAccountNumber string,
) error {
	if s.mailClient == nil {
		return nil
	}

	subject := "Payout Completed Successfully"
	preview := "Your payout has been successfully transferred."

	emailData := map[string]any{
		"DisbursementID": disbursementID.String(),
		"Amount":         payment.FormatAmount(amount, payment.Currency(currency)),
		"Currency":       currency,
		"HostName":       hostName,
		"AccountName":    accountName,
		"BankName":       bankName,
		"AccountNumber":  maskedAccountNumber,
		"CompletedAt":    time.Now().Format("January 2, 2006 at 3:04 PM"),
		"ArrivalMessage": "Funds should arrive in your account within 1-3 business days.",
		"DashboardURL":   fmt.Sprintf("%s/earnings", s.baseURL),

		// Required for the Layout
		"Subject": subject,
		"Preview": preview,
		"Year":    time.Now().Year(),
	}

	htmlBody, err := s.mailClient.RenderTemplate(
		templates.FS,
		"payout_success.html",
		emailData,
	)
	if err != nil {
		if s.log != nil {
			s.log.Error("failed to render payout success template: %v", err)
		}
		return nil
	}

	s.sendEmailAsync("send payout success email", func() error {
		job := emailJob.EmailJob{
			To:      hostEmail,
			Subject: subject,
			HTML:    htmlBody,
		}
		if err := s.publishEmailJob(job); err == nil {
			return nil
		} else if s.queueClient != nil && !s.queueClient.AllowFallback() {
			return err
		}
		return s.mailClient.SendHTML(ctx, hostEmail, subject, htmlBody)
	})
	return nil
}

// SendPayoutFailed sends notification when payout fails
func (s *NotificationService) SendPayoutFailed(
	ctx context.Context,
	disbursementID uuid.UUID,
	hostEmail string,
	hostName string,
	amount int64,
	currency string,
	failureReason string,
	nextRetryAt time.Time,
) error {
	if s.mailClient == nil {
		return nil
	}

	subject := "Payout Failed - Action May Be Required"
	preview := "There was an issue processing your payout."

	retryMessage := fmt.Sprintf("We'll automatically retry on %s", nextRetryAt.Format("January 2, 2006 at 3:04 PM"))
	if time.Until(nextRetryAt) > 24*time.Hour {
		retryMessage = "Please contact support to resolve this issue."
	}

	emailData := map[string]any{
		"DisbursementID": disbursementID.String(),
		"Amount":         payment.FormatAmount(amount, payment.Currency(currency)),
		"Currency":       currency,
		"HostName":       hostName,
		"FailureReason":  failureReason,
		"RetryMessage":   retryMessage,
		"FailedAt":       time.Now().Format("January 2, 2006 at 3:04 PM"),
		"SupportEmail":   "support@hauslet.com",
		"DashboardURL":   fmt.Sprintf("%s/earnings", s.baseURL),

		// Required for the Layout
		"Subject": subject,
		"Preview": preview,
		"Year":    time.Now().Year(),
	}

	htmlBody, err := s.mailClient.RenderTemplate(
		templates.FS,
		"payout_failed.html",
		emailData,
	)
	if err != nil {
		if s.log != nil {
			s.log.Error("failed to render payout failed template: %v", err)
		}
		return nil
	}

	s.sendEmailAsync("send payout failed email", func() error {
		job := emailJob.EmailJob{
			To:      hostEmail,
			Subject: subject,
			HTML:    htmlBody,
		}
		if err := s.publishEmailJob(job); err == nil {
			return nil
		} else if s.queueClient != nil && !s.queueClient.AllowFallback() {
			return err
		}
		return s.mailClient.SendHTML(ctx, hostEmail, subject, htmlBody)
	})
	return nil
}

// SendRefundProcessed sends notification when refund is processed
func (s *NotificationService) SendRefundProcessed(
	ctx context.Context,
	bookingID uuid.UUID,
	guestEmail string,
	guestName string,
	refundAmount int64,
	currency string,
) error {
	if s.mailClient == nil {
		return nil
	}

	subject := "Refund Processed"
	preview := "Your refund has been processed."

	emailData := map[string]any{
		"BookingID":       bookingID.String(),
		"RefundAmount":    payment.FormatAmount(refundAmount, payment.Currency(currency)),
		"Currency":        currency,
		"GuestName":       guestName,
		"ProcessedAt":     time.Now().Format("January 2, 2006 at 3:04 PM"),
		"TimelineMessage": "Refunds typically appear in your account within 5-10 business days.",
		"DashboardURL":    fmt.Sprintf("%s/bookings/%s", s.baseURL, bookingID),

		// Required for the Layout
		"Subject": subject,
		"Preview": preview,
		"Year":    time.Now().Year(),
	}

	htmlBody, err := s.mailClient.RenderTemplate(
		templates.FS,
		"refund_processed.html",
		emailData,
	)
	if err != nil {
		if s.log != nil {
			s.log.Error("failed to render refund processed template: %v", err)
		}
		return nil
	}

	s.sendEmailAsync("send refund processed email", func() error {
		job := emailJob.EmailJob{
			To:      guestEmail,
			Subject: subject,
			HTML:    htmlBody,
		}
		if err := s.publishEmailJob(job); err == nil {
			return nil
		} else if s.queueClient != nil && !s.queueClient.AllowFallback() {
			return err
		}
		return s.mailClient.SendHTML(ctx, guestEmail, subject, htmlBody)
	})
	return nil
}
