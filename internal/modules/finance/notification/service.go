package notification

import (
	"context"
	"fmt"
	"hauslet/internal/modules/finance/domain"
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
			s.log.Warn("notification dispatch failed", "label", label, "error", err)
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
			s.log.Warn("failed to publish finance email job", "queue_subject", s.queueSubject, "error", err)
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
			s.log.Error("failed to render payment receipt template", "error", err)
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
			s.log.Error("failed to render payout initiated template", "error", err)
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
			s.log.Error("failed to render payout success template", "error", err)
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
			s.log.Error("failed to render payout failed template", "error", err)
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
			s.log.Error("failed to render refund processed template", "error", err)
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

// SendReconciliationAlert sends alert to admins when discrepancies are found
func (s *NotificationService) SendReconciliationAlert(
	ctx context.Context,
	adminEmails []string,
	report *domain.ReconciliationReport,
) error {
	if s.mailClient == nil || len(adminEmails) == 0 {
		return nil
	}

	subject := fmt.Sprintf("[FINANCE ALERT] Reconciliation Discrepancies - Report %s",
		report.ID.String()[:8])
	preview := fmt.Sprintf("Found %d discrepancies during financial reconciliation",
		report.DiscrepanciesFound)

	// Categorize discrepancies by severity
	critical, high, medium, low := 0, 0, 0, 0
	for _, d := range report.Discrepancies {
		switch d.Severity {
		case domain.DiscrepancySeverityCritical:
			critical++
		case domain.DiscrepancySeverityHigh:
			high++
		case domain.DiscrepancySeverityMedium:
			medium++
		case domain.DiscrepancySeverityLow:
			low++
		}
	}

	emailData := map[string]any{
		"ReportID":            report.ID.String(),
		"StartedAt":           report.StartedAt.Format("January 2, 2006 at 3:04 PM"),
		"CompletedAt":         report.CompletedAt.Format("January 2, 2006 at 3:04 PM"),
		"TotalDiscrepancies":  report.DiscrepanciesFound,
		"CriticalCount":       critical,
		"HighCount":           high,
		"MediumCount":         medium,
		"LowCount":            low,
		"WalletsChecked":      report.TotalWalletsChecked,
		"TransactionsChecked": report.TotalTransactionsChecked,
		"Summary":             report.Summary,
		"Discrepancies":       formatDiscrepancies(report.Discrepancies),
		"DashboardURL":        fmt.Sprintf("%s/admin/finance/reconciliation/%s", s.baseURL, report.ID),
		"Subject":             subject,
		"Preview":             preview,
		"Year":                time.Now().Year(),
	}

	htmlBody, err := s.mailClient.RenderTemplate(
		templates.FS,
		"reconciliation_alert.html",
		emailData,
	)
	if err != nil {
		if s.log != nil {
			s.log.Error("failed to render reconciliation alert template", "error", err)
		}
		return fmt.Errorf("failed to render template: %w", err)
	}

	// Send to all admins (fire async for each)
	for _, email := range adminEmails {
		adminEmail := email
		s.sendEmailAsync(fmt.Sprintf("send reconciliation alert to %s", adminEmail), func() error {
			job := emailJob.EmailJob{
				To:      adminEmail,
				Subject: subject,
				HTML:    htmlBody,
			}
			if err := s.publishEmailJob(job); err == nil {
				return nil
			} else if s.queueClient != nil && !s.queueClient.AllowFallback() {
				return err
			}
			return s.mailClient.SendHTML(ctx, adminEmail, subject, htmlBody)
		})
	}

	return nil
}

// formatDiscrepancies converts discrepancies to email-friendly format
func formatDiscrepancies(discrepancies []domain.Discrepancy) []map[string]any {
	result := make([]map[string]any, 0, len(discrepancies))

	for _, d := range discrepancies {
		item := map[string]any{
			"Type":        string(d.Type),
			"Severity":    string(d.Severity),
			"Description": d.Description,
		}

		if d.WalletID != nil {
			item["WalletID"] = d.WalletID.String()[:8]
		}
		if d.TransactionID != nil {
			item["TransactionID"] = d.TransactionID.String()[:8]
		}
		if d.ExpectedValue != nil && d.ActualValue != nil {
			item["Expected"] = formatMinorUnits(*d.ExpectedValue)
			item["Actual"] = formatMinorUnits(*d.ActualValue)
			item["Difference"] = formatMinorUnits(*d.ActualValue - *d.ExpectedValue)
		}

		result = append(result, item)
	}

	return result
}

func formatMinorUnits(amount int64) string {
	return fmt.Sprintf("₦%.2f", float64(amount)/100.0)
}
