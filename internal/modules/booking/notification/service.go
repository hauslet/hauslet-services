package notification

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"hauslet/internal/modules/booking/domain"
	bookingtemplates "hauslet/internal/modules/booking/templates"
	"hauslet/internal/platform/email"
	"hauslet/internal/platform/queue"
	emailJob "hauslet/internal/queue/jobs/emails"
)

// NotificationService handles booking-module notifications.
type NotificationService struct {
	mailClient   *email.Client
	queueClient  *queue.Client
	queueSubject string
	baseURL      string
	log          *slog.Logger
}

// NewNotificationService wires the dependencies needed for booking notifications.
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

// SendHostApprovalRequest notifies a host when a manual request arrives.
func (s *NotificationService) SendHostApprovalRequest(ctx context.Context, booking *domain.Booking, hostName, hostEmail string) {
	if booking == nil || hostEmail == "" {
		return
	}

	subject := fmt.Sprintf("Booking request from %s", booking.GuestName)
	preview := fmt.Sprintf("%s wants to stay %d night(s) starting %s", booking.GuestName, booking.DurationNights(), s.formatDate(booking.ScheduledCheckIn()))

	data := s.baseBookingData(subject, preview, booking)
	data["HostName"] = s.fallbackName(hostName)
	data["GuestName"] = booking.GuestName
	data["ManageURL"] = s.hostBookingURL(booking)
	data["HoldWindow"] = s.holdWindowText(booking)
	if booking.SpecialRequests != nil {
		data["SpecialRequests"] = *booking.SpecialRequests
	}

	s.renderAndSend(ctx, "booking_request_host.html", hostEmail, subject, data, "send booking request host email")
}

// SendGuestRequestReceipt lets the guest know their manual request is pending.
func (s *NotificationService) SendGuestRequestReceipt(ctx context.Context, booking *domain.Booking) {
	if booking == nil || booking.GuestEmail == "" {
		return
	}

	subject := "We've sent your booking request"
	preview := "We'll notify you as soon as the host responds."

	data := s.baseBookingData(subject, preview, booking)
	data["GuestName"] = booking.GuestName
	data["BookingURL"] = s.guestBookingURL(booking)

	s.renderAndSend(ctx, "booking_request_guest.html", booking.GuestEmail, subject, data, "send booking request guest email")
}

// SendGuestApprovalGranted notifies the guest that a host approved the request.
func (s *NotificationService) SendGuestApprovalGranted(ctx context.Context, booking *domain.Booking, paymentURL *string) {
	if booking == nil || booking.GuestEmail == "" {
		return
	}

	subject := "Your booking was approved"
	preview := fmt.Sprintf("Complete payment %s to confirm.", s.holdWindowText(booking))

	data := s.baseBookingData(subject, preview, booking)
	data["GuestName"] = booking.GuestName
	data["HoldWindow"] = s.holdWindowText(booking)
	data["BookingURL"] = s.guestBookingURL(booking)
	if paymentURL != nil && *paymentURL != "" {
		data["PaymentURL"] = *paymentURL
	}

	s.renderAndSend(ctx, "booking_approved_guest.html", booking.GuestEmail, subject, data, "send booking approved guest email")
}

// SendGuestBookingRejected notifies the guest that the host declined the request.
func (s *NotificationService) SendGuestBookingRejected(ctx context.Context, booking *domain.Booking, reason *string) {
	if booking == nil || booking.GuestEmail == "" {
		return
	}

	subject := "Your booking request was not approved"
	preview := "The host declined this request."

	data := s.baseBookingData(subject, preview, booking)
	data["GuestName"] = booking.GuestName
	if reason != nil && *reason != "" {
		data["Reason"] = *reason
	}

	s.renderAndSend(ctx, "booking_rejected_guest.html", booking.GuestEmail, subject, data, "send booking rejected guest email")
}

// SendGuestPaymentFailed notifies the guest that payment failed.
func (s *NotificationService) SendGuestPaymentFailed(ctx context.Context, booking *domain.Booking) {
	if booking == nil || booking.GuestEmail == "" {
		return
	}

	subject := "Payment failed for your booking"
	preview := fmt.Sprintf("Please retry payment %s to keep your dates.", s.holdWindowText(booking))

	data := s.baseBookingData(subject, preview, booking)
	data["GuestName"] = booking.GuestName
	data["HoldWindow"] = s.holdWindowText(booking)
	data["BookingURL"] = s.guestBookingURL(booking)

	s.renderAndSend(ctx, "booking_payment_failed_guest.html", booking.GuestEmail, subject, data, "send booking payment failed guest email")
}

// SendGuestBookingExpired notifies the guest that the booking hold expired.
func (s *NotificationService) SendGuestBookingExpired(ctx context.Context, booking *domain.Booking, previousStatus domain.BookingStatus) {
	if booking == nil || booking.GuestEmail == "" {
		return
	}

	subject := "Your booking request expired"
	preview := "The hold on your dates has expired."

	data := s.baseBookingData(subject, preview, booking)
	data["GuestName"] = booking.GuestName
	data["BookingURL"] = s.guestBookingURL(booking)

	switch previousStatus {
	case domain.BookingStatusPendingApproval:
		data["Reason"] = "The host did not respond in time."
	default:
		data["Reason"] = "The payment window expired."
	}

	s.renderAndSend(ctx, "booking_expired_guest.html", booking.GuestEmail, subject, data, "send booking expired guest email")
}

// SendGuestBookingConfirmed confirms a booking to the guest after payment.
func (s *NotificationService) SendGuestBookingConfirmed(ctx context.Context, booking *domain.Booking) {
	if booking == nil || booking.GuestEmail == "" {
		return
	}

	subject := "Your booking is confirmed"
	preview := fmt.Sprintf("You're confirmed for %s.", s.formatDate(booking.ScheduledCheckIn()))

	data := s.baseBookingData(subject, preview, booking)
	data["GuestName"] = booking.GuestName
	data["BookingURL"] = s.guestBookingURL(booking)

	s.renderAndSend(ctx, "booking_instant_guest.html", booking.GuestEmail, subject, data, "send booking confirmed guest email")
}

// SendHostBookingConfirmed alerts the host that a booking is confirmed.
func (s *NotificationService) SendHostBookingConfirmed(ctx context.Context, booking *domain.Booking, hostName, hostEmail string) {
	if booking == nil || hostEmail == "" {
		return
	}

	subject := fmt.Sprintf("New confirmed booking from %s", booking.GuestName)
	preview := fmt.Sprintf("%s booked %d night(s) starting %s", booking.GuestName, booking.DurationNights(), s.formatDate(booking.ScheduledCheckIn()))

	data := s.baseBookingData(subject, preview, booking)
	data["HostName"] = s.fallbackName(hostName)
	data["GuestName"] = booking.GuestName
	data["ManageURL"] = s.hostBookingURL(booking)

	s.renderAndSend(ctx, "booking_instant_host.html", hostEmail, subject, data, "send booking confirmed host email")
}

// SendGuestBookingCancelled notifies the guest that their booking was cancelled.
func (s *NotificationService) SendGuestBookingCancelled(ctx context.Context, booking *domain.Booking, cancelledBy string, reason *string) {
	if booking == nil || booking.GuestEmail == "" {
		return
	}

	subject := "Your booking has been cancelled"
	preview := fmt.Sprintf("Your booking for %s has been cancelled.", s.formatDate(booking.ScheduledCheckIn()))

	data := s.baseBookingData(subject, preview, booking)
	data["GuestName"] = booking.GuestName
	data["CancelledBy"] = cancelledBy
	data["BookingURL"] = s.guestBookingURL(booking)

	if reason != nil && *reason != "" {
		data["Reason"] = *reason
	}

	s.renderAndSend(ctx, "booking_cancelled_guest.html", booking.GuestEmail, subject, data, "send booking cancelled guest email")
}

// SendHostBookingCancelled notifies the host that a booking was cancelled.
func (s *NotificationService) SendHostBookingCancelled(ctx context.Context, booking *domain.Booking, hostName, hostEmail, cancelledBy string, reason *string) {
	if booking == nil || hostEmail == "" {
		return
	}

	subject := fmt.Sprintf("Booking from %s has been cancelled", booking.GuestName)
	preview := fmt.Sprintf("The booking for %s has been cancelled.", s.formatDate(booking.ScheduledCheckIn()))

	data := s.baseBookingData(subject, preview, booking)
	data["HostName"] = s.fallbackName(hostName)
	data["GuestName"] = booking.GuestName
	data["CancelledBy"] = cancelledBy
	data["ManageURL"] = s.hostBookingURL(booking)

	if reason != nil && *reason != "" {
		data["Reason"] = *reason
	}

	s.renderAndSend(ctx, "booking_cancelled_host.html", hostEmail, subject, data, "send booking cancelled host email")
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
			s.log.Warn("failed to publish booking email job", "queue_subject", s.queueSubject, "error", err)
		}
		return err
	}

	return nil
}

func (s *NotificationService) renderAndSend(ctx context.Context, templateName, to, subject string, data map[string]any, label string) {
	htmlBody, err := s.mailClient.RenderTemplate(bookingtemplates.FS, templateName, data)
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

func (s *NotificationService) baseBookingData(subject, preview string, booking *domain.Booking) map[string]any {
	return map[string]any{
		"Subject":          subject,
		"Preview":          preview,
		"Year":             time.Now().Year(),
		"BookingReference": booking.BookingReference,
		"CheckIn":          s.formatDate(booking.ScheduledCheckIn()),
		"CheckOut":         s.formatDate(booking.ScheduledCheckOut()),
		"GuestCount":       booking.GuestCount,
		"Nights":           booking.DurationNights(),
		"TotalPrice":       s.formatTotal(booking),
		"SpecialRequests": func() string {
			if booking.SpecialRequests == nil {
				return ""
			}
			return *booking.SpecialRequests
		}(),
	}
}

func (s *NotificationService) formatDate(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("Mon, Jan 2 2006")
}

func (s *NotificationService) formatTotal(booking *domain.Booking) string {
	return fmt.Sprintf("%s %.2f", booking.Currency, booking.TotalPrice)
}

func (s *NotificationService) fallbackName(name string) string {
	if strings.TrimSpace(name) == "" {
		return "there"
	}
	return name
}

func (s *NotificationService) hostBookingURL(booking *domain.Booking) string {
	return fmt.Sprintf("%s/host/bookings/%s", s.baseURL, booking.ID)
}

func (s *NotificationService) guestBookingURL(booking *domain.Booking) string {
	return fmt.Sprintf("%s/bookings/%s", s.baseURL, booking.ID)
}

func (s *NotificationService) holdWindowText(booking *domain.Booking) string {
	if booking.HoldExpiresAt == nil {
		return "a short window"
	}
	return fmt.Sprintf("until %s", booking.HoldExpiresAt.Format("Mon, Jan 2 3:04 PM"))
}
