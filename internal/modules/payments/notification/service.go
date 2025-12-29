package notification

import (
	"context"
	"fmt"
	"hauslet/internal/modules/payments/domain"
	"hauslet/internal/modules/payments/templates"
	"hauslet/internal/platform/email"
	"hauslet/internal/platform/payment"
	"hauslet/internal/platform/queue"
	emailJob "hauslet/internal/queue/jobs/emails"
	"time"

	"github.com/go-pkgz/lgr"
	"github.com/google/uuid"
)

// NotificationService handles payment-related email notifications
type NotificationService struct {
	mailClient   *email.Client
	queueClient  *queue.Client
	queueSubject string
	baseURL      string
	log          *lgr.Logger
	userContacts UserContactProvider
	bizContacts  BusinessContactProvider
}

// UserContactProvider exposes minimal user contact info.
type UserContactProvider interface {
	GetUserContactEmail(ctx context.Context, userID string) (string, error)
}

// BusinessContactProvider exposes minimal business contact info.
type BusinessContactProvider interface {
	GetBusinessContactEmail(ctx context.Context, businessID uuid.UUID) (string, error)
}

// NewNotificationService creates a new notification service
func NewNotificationService(
	mailClient *email.Client,
	queueClient *queue.Client,
	queueSubject string,
	baseURL string,
	userContacts UserContactProvider,
	bizContacts BusinessContactProvider,
	log *lgr.Logger,
) *NotificationService {
	return &NotificationService{
		mailClient:   mailClient,
		queueClient:  queueClient,
		queueSubject: queueSubject,
		baseURL:      baseURL,
		userContacts: userContacts,
		bizContacts:  bizContacts,
		log:          log,
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
func (s *NotificationService) publishEmailJob(job emailJob.EmailJob) error {
	if s.queueClient == nil || s.queueSubject == "" {
		return fmt.Errorf("queue not configured")
	}

	pubCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.queueClient.Publish(pubCtx, s.queueSubject, job); err != nil {
		if s.log != nil {
			s.log.Logf("[WARN] failed to publish payment email job to %s: %v", s.queueSubject, err)
		}
		return err
	}

	return nil
}

// SendPaymentReceipt sends a payment receipt email
func (s *NotificationService) SendPaymentReceipt(pmt *domain.Payment) error {
	ctx := context.Background()
	subject := fmt.Sprintf("Payment Receipt - %s", pmt.Reference)
	preview := "Your payment has been successfully processed."

	emailData := map[string]any{
		"PaymentID":    pmt.ID.String(),
		"Reference":    pmt.Reference,
		"Amount":       payment.FormatAmount(pmt.Amount, pmt.Currency),
		"Currency":     pmt.Currency,
		"PayerName":    pmt.PayerName,
		"PayerEmail":   pmt.PayerEmail,
		"Description":  pmt.Description,
		"PaidAt":       pmt.UpdatedAt.Format("January 2, 2006 at 3:04 PM"),
		"DashboardURL": fmt.Sprintf("%s/payments/%s", s.baseURL, pmt.ID),

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
			s.log.Logf("[ERROR] failed to render payment receipt template: %v", err)
		}
		return nil
	}

	s.sendEmailAsync("send payment receipt email", func() error {
		job := emailJob.EmailJob{
			To:      pmt.PayerEmail,
			Subject: subject,
			HTML:    htmlBody,
		}
		if err := s.publishEmailJob(job); err == nil {
			return nil
		} else if s.queueClient != nil && !s.queueClient.AllowFallback() {
			return err
		}
		return s.mailClient.SendHTML(ctx, pmt.PayerEmail, subject, htmlBody)
	})
	return nil
}

// SendRefundNotification sends a refund notification email
func (s *NotificationService) SendRefundNotification(pmt *domain.Payment, refundAmount int64) {
	ctx := context.Background()
	subject := fmt.Sprintf("Refund Processed - %s", pmt.Reference)
	preview := "A refund has been processed for your payment."

	emailData := map[string]any{
		"PaymentID":         pmt.ID.String(),
		"Reference":         pmt.Reference,
		"RefundAmount":      payment.FormatAmount(refundAmount, pmt.Currency),
		"OriginalAmount":    payment.FormatAmount(pmt.Amount, pmt.Currency),
		"Currency":          pmt.Currency,
		"PayerName":         pmt.PayerName,
		"PayerEmail":        pmt.PayerEmail,
		"RefundedAt":        pmt.RefundedAt.Format("January 2, 2006 at 3:04 PM"),
		"ProcessingMessage": "The refund will be processed within 5-10 business days.",
		"DashboardURL":      fmt.Sprintf("%s/payments/%s", s.baseURL, pmt.ID),

		// Required for the Layout
		"Subject": subject,
		"Preview": preview,
		"Year":    time.Now().Year(),
	}

	htmlBody, err := s.mailClient.RenderTemplate(
		templates.FS,
		"refund_notification.html",
		emailData,
	)
	if err != nil {
		if s.log != nil {
			s.log.Logf("[ERROR] failed to render refund notification template: %v", err)
		}
		return
	}

	s.sendEmailAsync("send refund notification email", func() error {
		job := emailJob.EmailJob{
			To:      pmt.PayerEmail,
			Subject: subject,
			HTML:    htmlBody,
		}
		if err := s.publishEmailJob(job); err == nil {
			return nil
		} else if s.queueClient != nil && !s.queueClient.AllowFallback() {
			return err
		}
		return s.mailClient.SendHTML(ctx, pmt.PayerEmail, subject, htmlBody)
	})
}

// SendPayoutNotification sends a payout notification email
func (s *NotificationService) SendPayoutNotification(tx *domain.Transaction, pd *domain.PayoutDetail) {
	ctx := context.Background()
	subject := fmt.Sprintf("Payout Processed - %s", tx.Reference)
	preview := "Your payout has been successfully processed."

	recipientEmail := ""
	switch {
	case pd != nil && pd.UserID != nil:
		if s.userContacts == nil {
			if s.log != nil {
				s.log.Logf("[WARN] user service not configured; cannot resolve payout recipient for user %s", pd.UserID.String())
			}
			return
		}
		email, err := s.userContacts.GetUserContactEmail(ctx, pd.UserID.String())
		if err != nil {
			if s.log != nil {
				s.log.Logf("[WARN] failed to fetch payout recipient user %s: %v", pd.UserID.String(), err)
			}
			return
		}
		recipientEmail = email
	case pd != nil && pd.BusinessID != nil:
		if s.bizContacts == nil {
			if s.log != nil {
				s.log.Logf("[WARN] business service not configured; cannot resolve payout recipient for business %s", pd.BusinessID.String())
			}
			return
		}
		email, err := s.bizContacts.GetBusinessContactEmail(ctx, *pd.BusinessID)
		if err != nil {
			if s.log != nil {
				s.log.Logf("[WARN] failed to fetch payout recipient business %s: %v", pd.BusinessID.String(), err)
			}
			return
		}
		recipientEmail = email
	case tx != nil && tx.BusinessID != nil:
		if s.bizContacts == nil {
			if s.log != nil {
				s.log.Logf("[WARN] business service not configured; cannot resolve payout recipient for business %s", tx.BusinessID.String())
			}
			return
		}
		email, err := s.bizContacts.GetBusinessContactEmail(ctx, *tx.BusinessID)
		if err != nil {
			if s.log != nil {
				s.log.Logf("[WARN] failed to fetch payout recipient business %s: %v", tx.BusinessID.String(), err)
			}
			return
		}
		recipientEmail = email
	default:
		if s.log != nil {
			s.log.Logf("[WARN] payout recipient not found for transaction %s", tx.ID)
		}
		return
	}
	if recipientEmail == "" {
		if s.log != nil {
			s.log.Logf("[WARN] payout recipient email missing for transaction %s", tx.ID)
		}
		return
	}

	emailData := map[string]any{
		"TransactionID": tx.ID.String(),
		"Reference":     tx.Reference,
		"Amount":        payment.FormatAmount(tx.Amount, tx.Currency),
		"Currency":      tx.Currency,
		"AccountName":   pd.AccountName,
		"BankName":      pd.BankName,
		"AccountNumber": domain.MaskAccountNumber(pd.AccountNumber),
		"Description":   tx.Description,
		"ProcessedAt":   tx.ProcessedAt.Format("January 2, 2006 at 3:04 PM"),
		"StatusMessage": getPayoutStatusMessage(tx.Status),
		"DashboardURL":  fmt.Sprintf("%s/payouts/%s", s.baseURL, tx.ID),

		// Required for the Layout
		"Subject": subject,
		"Preview": preview,
		"Year":    time.Now().Year(),
	}

	htmlBody, err := s.mailClient.RenderTemplate(
		templates.FS,
		"payout_notification.html",
		emailData,
	)
	if err != nil {
		if s.log != nil {
			s.log.Logf("[ERROR] failed to render payout notification template: %v", err)
		}
		return
	}

	s.sendEmailAsync("send payout notification email", func() error {
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
}

// getPayoutStatusMessage returns a user-friendly status message
func getPayoutStatusMessage(status domain.TransactionStatus) string {
	switch status {
	case domain.TransactionStatusSucceeded:
		return "Your payout has been successfully processed and should arrive in your bank account within 1-3 business days."
	case domain.TransactionStatusPending:
		return "Your payout is being processed. You will receive a confirmation email once it's complete."
	case domain.TransactionStatusFailed:
		return "Your payout failed. Please contact support for assistance."
	default:
		return "Payout status unknown. Please check your dashboard for details."
	}
}
