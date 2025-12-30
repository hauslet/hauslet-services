package notification

import (
	"context"
	"fmt"
	"strings"
	"time"

	"hauslet/internal/modules/review/domain"
	reviewtemplates "hauslet/internal/modules/review/templates"
	"hauslet/internal/platform/email"
	"hauslet/internal/platform/queue"
	emailJob "hauslet/internal/queue/jobs/emails"

	"github.com/go-pkgz/lgr"
	"github.com/google/uuid"
)

// NotificationService handles review-module notifications.
type NotificationService struct {
	mailClient   *email.Client
	queueClient  *queue.Client
	queueSubject string
	baseURL      string
	log          *lgr.Logger
}

// ContactInfo represents user contact details for notifications
type ContactInfo struct {
	ID    uuid.UUID
	Name  string
	Email string
}

// NewNotificationService wires the dependencies needed for review notifications.
func NewNotificationService(
	mailClient *email.Client,
	queueClient *queue.Client,
	queueSubject string,
	baseURL string,
	log *lgr.Logger,
) *NotificationService {
	return &NotificationService{
		mailClient:   mailClient,
		queueClient:  queueClient,
		queueSubject: queueSubject,
		baseURL:      strings.TrimRight(baseURL, "/"),
		log:          log,
	}
}

// SendReviewPublished notifies the host when a guest reviews their listing.
func (s *NotificationService) SendReviewPublished(ctx context.Context, review *domain.Review, host ContactInfo, listingTitle string) {
	if review == nil || host.Email == "" {
		return
	}

	subject := "You have a new review"
	preview := fmt.Sprintf("A guest left a %d-star review for %s", review.Rating, listingTitle)

	data := s.baseReviewData(subject, preview, review)
	data["HostName"] = s.fallbackName(host.Name)
	data["ListingTitle"] = listingTitle
	data["Rating"] = review.Rating
	data["ReviewURL"] = s.reviewURL(review.ID)
	data["ReviewBody"] = review.Body

	s.renderAndSend(ctx, "review_published_host.html", host.Email, subject, data, "send review published host email")
}

// SendResponseCreated notifies the guest when the host responds to their review.
func (s *NotificationService) SendResponseCreated(ctx context.Context, review *domain.Review, response *domain.ReviewResponse, guest ContactInfo, listingTitle string) {
	if review == nil || response == nil || guest.Email == "" {
		return
	}

	subject := "Host responded to your review"
	preview := fmt.Sprintf("The host replied to your review of %s", listingTitle)

	data := s.baseReviewData(subject, preview, review)
	data["GuestName"] = s.fallbackName(guest.Name)
	data["ListingTitle"] = listingTitle
	data["ResponseBody"] = response.Body
	data["ReviewURL"] = s.reviewURL(review.ID)

	s.renderAndSend(ctx, "review_response_created.html", guest.Email, subject, data, "send review response created email")
}

// SendBothReviewsPublished notifies both parties when both reviews are published.
func (s *NotificationService) SendBothReviewsPublished(ctx context.Context, guestReview, hostReview *domain.Review, guest, host ContactInfo, listingTitle string) {
	// Notify Guest
	if guest.Email != "" && guestReview != nil {
		subject := "You can now see your review"
		preview := fmt.Sprintf("Both reviews for %s are now visible", listingTitle)

		data := s.baseReviewData(subject, preview, guestReview)
		data["GuestName"] = s.fallbackName(guest.Name)
		data["ListingTitle"] = listingTitle
		data["ReviewURL"] = s.reviewURL(guestReview.ID)

		s.renderAndSend(ctx, "both_reviews_published_guest.html", guest.Email, subject, data, "send both reviews published guest email")
	}

	// Notify Host
	if host.Email != "" && hostReview != nil {
		subject := "Reviews are now public"
		preview := fmt.Sprintf("Both reviews for %s are now visible", listingTitle)

		data := s.baseReviewData(subject, preview, hostReview)
		data["HostName"] = s.fallbackName(host.Name)
		data["ListingTitle"] = listingTitle
		data["ReviewURL"] = s.reviewURL(hostReview.ID)

		s.renderAndSend(ctx, "both_reviews_published_host.html", host.Email, subject, data, "send both reviews published host email")
	}
}

// SendReviewReminderToGuest reminds guest to leave a review before the standoff expires.
func (s *NotificationService) SendReviewReminderToGuest(ctx context.Context, bookingID uuid.UUID, guest ContactInfo, listingTitle string, daysLeft int) {
	if guest.Email == "" {
		return
	}

	subject := fmt.Sprintf("Leave a review - %d day(s) left", daysLeft)
	preview := fmt.Sprintf("Share your experience at %s before the window closes", listingTitle)

	data := map[string]any{
		"Subject":      subject,
		"Preview":      preview,
		"Year":         time.Now().Year(),
		"GuestName":    s.fallbackName(guest.Name),
		"ListingTitle": listingTitle,
		"DaysLeft":     daysLeft,
		"ReviewURL":    s.createReviewURL(bookingID),
	}

	s.renderAndSend(ctx, "review_reminder_guest.html", guest.Email, subject, data, "send review reminder guest email")
}

// SendReviewReminderToHost reminds host to leave a review before the standoff expires.
func (s *NotificationService) SendReviewReminderToHost(ctx context.Context, bookingID uuid.UUID, host ContactInfo, guestName string, daysLeft int) {
	if host.Email == "" {
		return
	}

	subject := fmt.Sprintf("Leave a review - %d day(s) left", daysLeft)
	preview := fmt.Sprintf("Review your experience with %s before the window closes", guestName)

	data := map[string]any{
		"Subject":   subject,
		"Preview":   preview,
		"Year":      time.Now().Year(),
		"HostName":  s.fallbackName(host.Name),
		"GuestName": guestName,
		"DaysLeft":  daysLeft,
		"ReviewURL": s.createReviewURL(bookingID),
	}

	s.renderAndSend(ctx, "review_reminder_host.html", host.Email, subject, data, "send review reminder host email")
}

// SendReviewInviteToGuest notifies guest that it's time to review their stay.
func (s *NotificationService) SendReviewInviteToGuest(ctx context.Context, bookingID uuid.UUID, guest ContactInfo, listingTitle string, reviewWindowDays int) {
	if guest.Email == "" {
		return
	}

	subject := "How was your stay? Leave a review"
	preview := fmt.Sprintf("Share your experience at %s", listingTitle)

	data := map[string]any{
		"Subject":         subject,
		"Preview":         preview,
		"Year":            time.Now().Year(),
		"GuestName":       s.fallbackName(guest.Name),
		"ListingTitle":    listingTitle,
		"ReviewWindowDays": reviewWindowDays,
		"ReviewURL":       s.createReviewURL(bookingID),
	}

	s.renderAndSend(ctx, "review_invite_guest.html", guest.Email, subject, data, "send review invite guest email")
}

// SendReviewInviteToHost notifies host that it's time to review their guest.
func (s *NotificationService) SendReviewInviteToHost(ctx context.Context, bookingID uuid.UUID, host ContactInfo, guestName string, reviewWindowDays int) {
	if host.Email == "" {
		return
	}

	subject := "Leave a review for your guest"
	preview := fmt.Sprintf("Review your experience with %s", guestName)

	data := map[string]any{
		"Subject":         subject,
		"Preview":         preview,
		"Year":            time.Now().Year(),
		"HostName":        s.fallbackName(host.Name),
		"GuestName":       guestName,
		"ReviewWindowDays": reviewWindowDays,
		"ReviewURL":       s.createReviewURL(bookingID),
	}

	s.renderAndSend(ctx, "review_invite_host.html", host.Email, subject, data, "send review invite host email")
}

// SendReviewModerationRejected notifies reviewer that their review was rejected by moderation.
func (s *NotificationService) SendReviewModerationRejected(ctx context.Context, review *domain.Review, reviewer ContactInfo, reason string) {
	if review == nil || reviewer.Email == "" {
		return
	}

	subject := "Your review needs revision"
	preview := "We couldn't publish your review in its current form"

	data := s.baseReviewData(subject, preview, review)
	data["ReviewerName"] = s.fallbackName(reviewer.Name)
	data["Reason"] = s.formatModerationReason(reason)
	data["ReviewURL"] = s.reviewURL(review.ID)
	data["GuidelinesURL"] = s.baseURL + "/help/review-guidelines"

	s.renderAndSend(ctx, "review_moderation_rejected.html", reviewer.Email, subject, data, "send review moderation rejected email")
}

// SendReviewReportedToAdmins notifies admins when a review is reported.
func (s *NotificationService) SendReviewReportedToAdmins(ctx context.Context, review *domain.Review, adminEmails []string, reportReason string) {
	if review == nil || len(adminEmails) == 0 {
		return
	}

	subject := fmt.Sprintf("Review flagged: %s", review.ID.String()[:8])
	preview := "A review has been flagged for moderation"

	data := s.baseReviewData(subject, preview, review)
	data["ReportReason"] = reportReason
	data["ReviewBody"] = review.Body
	data["Rating"] = review.Rating
	data["AdminReviewURL"] = s.adminReviewURL(review.ID)

	for _, adminEmail := range adminEmails {
		s.renderAndSend(ctx, "review_reported_admin.html", adminEmail, subject, data, "send review reported admin email")
	}
}

// SendReviewEscalatedToAdmins notifies admins when AI moderation escalates a review.
func (s *NotificationService) SendReviewEscalatedToAdmins(ctx context.Context, review *domain.Review, adminEmails []string, aiReasons []string) {
	if review == nil || len(adminEmails) == 0 {
		return
	}

	subject := fmt.Sprintf("AI escalated review: %s", review.ID.String()[:8])
	preview := "AI moderation needs human review"

	data := s.baseReviewData(subject, preview, review)
	data["AIReasons"] = strings.Join(aiReasons, ", ")
	data["ReviewBody"] = review.Body
	data["Rating"] = review.Rating
	data["AdminReviewURL"] = s.adminReviewURL(review.ID)

	for _, adminEmail := range adminEmails {
		s.renderAndSend(ctx, "review_escalated_admin.html", adminEmail, subject, data, "send review escalated admin email")
	}
}

// ----------------------------------------------------------------
// HELPER METHODS
// ----------------------------------------------------------------

func (s *NotificationService) baseReviewData(subject, preview string, review *domain.Review) map[string]any {
	return map[string]any{
		"Subject":   subject,
		"Preview":   preview,
		"Year":      time.Now().Year(),
		"ReviewID":  review.ID.String(),
		"CreatedAt": s.formatDate(review.CreatedAt),
	}
}

func (s *NotificationService) formatDate(t time.Time) string {
	return t.Format("Mon, Jan 2 2006")
}

func (s *NotificationService) fallbackName(name string) string {
	if strings.TrimSpace(name) == "" {
		return "there"
	}
	return name
}

func (s *NotificationService) reviewURL(reviewID uuid.UUID) string {
	return fmt.Sprintf("%s/reviews/%s", s.baseURL, reviewID)
}

func (s *NotificationService) createReviewURL(bookingID uuid.UUID) string {
	return fmt.Sprintf("%s/bookings/%s/review", s.baseURL, bookingID)
}

func (s *NotificationService) adminReviewURL(reviewID uuid.UUID) string {
	return fmt.Sprintf("%s/admin/reviews/%s", s.baseURL, reviewID)
}

func (s *NotificationService) formatModerationReason(reason string) string {
	switch reason {
	case "spam":
		return "Your review appears to be spam or promotional content"
	case "offensive":
		return "Your review contains offensive or inappropriate language"
	case "fraudulent":
		return "Your review appears to be fraudulent or fake"
	case "irrelevant":
		return "Your review is not relevant to the booking experience"
	case "personal_info":
		return "Your review contains personal information (email, phone, etc.)"
	default:
		return "Your review doesn't meet our community guidelines"
	}
}

// sendEmailAsync runs the provided send function in a goroutine and logs errors.
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
			s.log.Logf("[WARN] failed to publish review email job to %s: %v", s.queueSubject, err)
		}
		return err
	}

	return nil
}

func (s *NotificationService) renderAndSend(ctx context.Context, templateName, to, subject string, data map[string]any, label string) {
	htmlBody, err := s.mailClient.RenderTemplate(reviewtemplates.FS, templateName, data)
	if err != nil {
		if s.log != nil {
			s.log.Logf("[ERROR] failed to render %s: %v", templateName, err)
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
