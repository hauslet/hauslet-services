package notification

import (
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"hauslet/internal/modules/calendar/domain"
	calendartemplates "hauslet/internal/modules/calendar/templates"
	"hauslet/internal/platform/email"
	"hauslet/internal/platform/queue"
	emailJob "hauslet/internal/queue/jobs/emails"

	ics "github.com/arran4/golang-ical"
	"github.com/google/uuid"
)

// ListingInfo contains the minimal listing data needed for calendar notifications.
type ListingInfo struct {
	ID        uuid.UUID
	Title     string
	OwnerID   uuid.UUID
	OwnerType string
}

// ContactInfo represents user contact details for notifications.
type ContactInfo struct {
	ID    uuid.UUID
	Name  string
	Email string
	Phone *string
}

// ListingInfoProvider resolves listing data for notifications.
type ListingInfoProvider interface {
	GetListingInfo(ctx context.Context, listingID uuid.UUID) (*ListingInfo, error)
}

// UserContactProvider resolves user contact data for notifications.
type UserContactProvider interface {
	GetUserContact(ctx context.Context, userID uuid.UUID) (*ContactInfo, error)
}

// NotificationService handles calendar email notifications.
type NotificationService struct {
	mailClient   *email.Client
	queueClient  *queue.Client
	queueSubject string
	baseURL      string
	listings     ListingInfoProvider
	contacts     UserContactProvider
	log          *slog.Logger
}

// NewNotificationService wires the dependencies needed for calendar notifications.
func NewNotificationService(
	mailClient *email.Client,
	queueClient *queue.Client,
	queueSubject string,
	baseURL string,
	listings ListingInfoProvider,
	contacts UserContactProvider,
	log *slog.Logger,
) *NotificationService {
	return &NotificationService{
		mailClient:   mailClient,
		queueClient:  queueClient,
		queueSubject: queueSubject,
		baseURL:      strings.TrimRight(baseURL, "/"),
		listings:     listings,
		contacts:     contacts,
		log:          log,
	}
}

// SendShowingRequest notifies the host and prospect about a showing request.
func (s *NotificationService) SendShowingRequest(ctx context.Context, event *domain.CalendarEvent) {
	if event == nil || event.ShowingDetails == nil {
		return
	}

	listing, err := s.getListingInfo(ctx, event.ListingID)
	if err != nil || listing == nil {
		return
	}

	prospect := ContactInfo{
		Name:  event.ShowingDetails.ProspectName,
		Email: event.ShowingDetails.ProspectEmail,
		Phone: event.ShowingDetails.ProspectPhone,
	}

	if host := s.getOwnerContact(ctx, listing.OwnerID); host != nil && host.Email != "" {
		subject := fmt.Sprintf("New showing request for %s", listing.Title)
		preview := fmt.Sprintf("%s requested a showing on %s", prospect.Name, s.formatEventTime(event.StartTime))

		data := s.baseData(subject, preview)
		data["HostName"] = s.fallbackName(host.Name)
		data["ProspectName"] = prospect.Name
		data["ProspectEmail"] = prospect.Email
		data["ProspectPhone"] = s.formatOptionalPhone(prospect.Phone)
		data["ListingTitle"] = listing.Title
		data["StartTime"] = s.formatEventTime(event.StartTime)
		data["EndTime"] = s.formatEventTime(event.EndTime)
		data["Notes"] = s.formatOptionalNotes(event.ShowingDetails.Notes)
		data["ManageURL"] = s.hostListingURL(listing.ID)

		s.renderAndSend(ctx, "showing_request_host.html", host.Email, subject, data, nil, "send showing request host email")
	}

	if prospect.Email != "" {
		subject := "We've received your showing request"
		preview := fmt.Sprintf("We'll confirm your showing for %s soon.", listing.Title)

		data := s.baseData(subject, preview)
		data["ProspectName"] = s.fallbackName(prospect.Name)
		data["ListingTitle"] = listing.Title
		data["StartTime"] = s.formatEventTime(event.StartTime)
		data["EndTime"] = s.formatEventTime(event.EndTime)
		data["ListingURL"] = s.listingURL(listing.ID)

		s.renderAndSend(ctx, "showing_request_prospect.html", prospect.Email, subject, data, nil, "send showing request prospect email")
	}
}

// SendShowingConfirmed notifies the prospect that a showing was confirmed.
func (s *NotificationService) SendShowingConfirmed(ctx context.Context, event *domain.CalendarEvent) {
	if event == nil || event.ShowingDetails == nil {
		return
	}

	listing, err := s.getListingInfo(ctx, event.ListingID)
	if err != nil || listing == nil {
		return
	}

	prospect := ContactInfo{
		Name:  event.ShowingDetails.ProspectName,
		Email: event.ShowingDetails.ProspectEmail,
		Phone: event.ShowingDetails.ProspectPhone,
	}
	if prospect.Email == "" {
		return
	}

	subject := "Your showing is confirmed"
	preview := fmt.Sprintf("Your showing for %s is confirmed.", listing.Title)

	calendarURL, attachments := s.buildCalendarAssets(event, s.showingSummary(listing.Title), s.showingDescription(listing.Title))

	data := s.baseData(subject, preview)
	data["ProspectName"] = s.fallbackName(prospect.Name)
	data["ListingTitle"] = listing.Title
	data["StartTime"] = s.formatEventTime(event.StartTime)
	data["EndTime"] = s.formatEventTime(event.EndTime)
	data["ListingURL"] = s.listingURL(listing.ID)
	data["CalendarURL"] = calendarURL

	s.renderAndSend(ctx, "showing_confirmed_prospect.html", prospect.Email, subject, data, attachments, "send showing confirmed email")
}

// SendOpenHouseRegistration confirms an open house registration.
func (s *NotificationService) SendOpenHouseRegistration(ctx context.Context, event *domain.CalendarEvent, attendee domain.Attendee) {
	if event == nil || event.OpenHouseDetails == nil || attendee.Email == "" {
		return
	}

	listing, err := s.getListingInfo(ctx, event.ListingID)
	if err != nil || listing == nil {
		return
	}

	subject := "You're registered for an open house"
	preview := fmt.Sprintf("Your spot for %s is confirmed.", listing.Title)

	openHouseTitle := s.openHouseTitle(event, listing.Title)
	calendarURL, attachments := s.buildCalendarAssets(event, openHouseTitle, s.openHouseDescription(openHouseTitle, listing.Title))

	data := s.baseData(subject, preview)
	data["AttendeeName"] = s.fallbackName(attendee.Name)
	data["ListingTitle"] = listing.Title
	data["OpenHouseTitle"] = openHouseTitle
	data["StartTime"] = s.formatEventTime(event.StartTime)
	data["EndTime"] = s.formatEventTime(event.EndTime)
	data["ListingURL"] = s.listingURL(listing.ID)
	data["CalendarURL"] = calendarURL

	s.renderAndSend(ctx, "open_house_registration.html", attendee.Email, subject, data, attachments, "send open house registration email")
}

// SendShowingReminder sends reminder emails for a showing.
func (s *NotificationService) SendShowingReminder(ctx context.Context, event *domain.CalendarEvent, reminderMinutes int) {
	if event == nil || event.ShowingDetails == nil {
		return
	}

	listing, err := s.getListingInfo(ctx, event.ListingID)
	if err != nil || listing == nil {
		return
	}

	reminderLabel := s.reminderLabel(reminderMinutes)
	subject := fmt.Sprintf("Reminder: your showing is %s", reminderLabel)
	preview := fmt.Sprintf("Your showing for %s starts %s.", listing.Title, reminderLabel)

	data := s.baseData(subject, preview)
	data["ListingTitle"] = listing.Title
	data["StartTime"] = s.formatEventTime(event.StartTime)
	data["EndTime"] = s.formatEventTime(event.EndTime)
	data["ReminderLabel"] = reminderLabel
	data["ListingURL"] = s.listingURL(listing.ID)

	if host := s.getOwnerContact(ctx, listing.OwnerID); host != nil && host.Email != "" {
		data["RecipientName"] = s.fallbackName(host.Name)
		s.renderAndSend(ctx, "showing_reminder.html", host.Email, subject, data, nil, "send showing reminder host email")
	}

	if event.ShowingDetails.ProspectEmail != "" {
		data["RecipientName"] = s.fallbackName(event.ShowingDetails.ProspectName)
		s.renderAndSend(ctx, "showing_reminder.html", event.ShowingDetails.ProspectEmail, subject, data, nil, "send showing reminder prospect email")
	}
}

// SendOpenHouseReminder sends reminder emails to open house attendees.
func (s *NotificationService) SendOpenHouseReminder(ctx context.Context, event *domain.CalendarEvent, attendee domain.Attendee, reminderMinutes int) {
	if event == nil || event.OpenHouseDetails == nil || attendee.Email == "" {
		return
	}

	listing, err := s.getListingInfo(ctx, event.ListingID)
	if err != nil || listing == nil {
		return
	}

	reminderLabel := s.reminderLabel(reminderMinutes)
	subject := fmt.Sprintf("Reminder: open house is %s", reminderLabel)
	preview := fmt.Sprintf("%s starts %s.", listing.Title, reminderLabel)

	openHouseTitle := s.openHouseTitle(event, listing.Title)

	data := s.baseData(subject, preview)
	data["AttendeeName"] = s.fallbackName(attendee.Name)
	data["ListingTitle"] = listing.Title
	data["OpenHouseTitle"] = openHouseTitle
	data["StartTime"] = s.formatEventTime(event.StartTime)
	data["EndTime"] = s.formatEventTime(event.EndTime)
	data["ReminderLabel"] = reminderLabel
	data["ListingURL"] = s.listingURL(listing.ID)

	s.renderAndSend(ctx, "open_house_reminder.html", attendee.Email, subject, data, nil, "send open house reminder email")
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
			s.log.Warn("failed to publish calendar email job", "queue_subject", s.queueSubject, "error", err)
		}
		return err
	}

	return nil
}

func (s *NotificationService) renderAndSend(ctx context.Context, templateName, to, subject string, data map[string]any, attachments []emailJob.Attachment, label string) {
	htmlBody, err := s.mailClient.RenderTemplate(calendartemplates.FS, templateName, data)
	if err != nil {
		if s.log != nil {
			s.log.Error("failed to render template", "template", templateName, "error", err)
		}
		return
	}

	s.sendEmailAsync(label, func() error {
		job := emailJob.EmailJob{
			To:          to,
			Subject:     subject,
			HTML:        htmlBody,
			Attachments: attachments,
		}
		if err := s.publishEmailJob(job); err == nil {
			return nil
		} else if s.queueClient != nil && !s.queueClient.AllowFallback() {
			return err
		}

		return s.sendFallbackEmail(ctx, to, subject, htmlBody, attachments)
	})
}

func (s *NotificationService) sendFallbackEmail(ctx context.Context, to, subject, htmlBody string, attachments []emailJob.Attachment) error {
	if len(attachments) == 0 {
		return s.mailClient.SendHTML(ctx, to, subject, htmlBody)
	}

	decoded := make([]email.Attachment, 0, len(attachments))
	for _, attachment := range attachments {
		payload, err := base64.StdEncoding.DecodeString(attachment.ContentBase64)
		if err != nil {
			return err
		}
		decoded = append(decoded, email.Attachment{
			Filename:    attachment.Filename,
			ContentType: attachment.ContentType,
			Content:     payload,
		})
	}

	return s.mailClient.SendHTMLWithAttachments(ctx, to, subject, htmlBody, decoded)
}

func (s *NotificationService) baseData(subject, preview string) map[string]any {
	return map[string]any{
		"Subject": subject,
		"Preview": preview,
		"Year":    time.Now().Year(),
	}
}

func (s *NotificationService) getListingInfo(ctx context.Context, listingID uuid.UUID) (*ListingInfo, error) {
	if s.listings == nil {
		return nil, fmt.Errorf("listing provider not configured")
	}
	listing, err := s.listings.GetListingInfo(ctx, listingID)
	if err != nil {
		if s.log != nil {
			s.log.Warn("failed to load listing info", "listing_id", listingID, "error", err)
		}
		return nil, err
	}
	return listing, nil
}

func (s *NotificationService) getOwnerContact(ctx context.Context, ownerID uuid.UUID) *ContactInfo {
	if s.contacts == nil {
		return nil
	}
	contact, err := s.contacts.GetUserContact(ctx, ownerID)
	if err != nil {
		if s.log != nil {
			s.log.Warn("failed to load owner contact", "owner_id", ownerID, "error", err)
		}
		return nil
	}
	return contact
}

func (s *NotificationService) buildCalendarAssets(event *domain.CalendarEvent, summary, description string) (string, []emailJob.Attachment) {
	ics, err := BuildICS(event, summary, description, "", ics.MethodPublish)
	if err != nil {
		if s.log != nil {
			s.log.Warn("failed to build ics", "event_id", event.ID, "error", err)
		}
		return "", nil
	}

	filename := s.calendarFilename(event)
	contentType := "text/calendar; charset=utf-8; method=PUBLISH"

	// Encode ICS content to base64 for email attachment
	icsBase64 := base64.StdEncoding.EncodeToString([]byte(ics))

	// Debug logging
	if s.log != nil {
		previewLen := 50
		icsPreviewLen := 50
		if len(icsBase64) < previewLen {
			previewLen = len(icsBase64)
		}
		if len(ics) < icsPreviewLen {
			icsPreviewLen = len(ics)
		}
		s.log.Info("Building calendar attachment",
			"filename", filename,
			"ics_raw_length", len(ics),
			"ics_raw_preview", ics[:icsPreviewLen],
			"ics_base64_length", len(icsBase64),
			"ics_base64_preview", icsBase64[:previewLen],
		)
	}

	attachments := []emailJob.Attachment{{
		Filename:      filename,
		ContentType:   contentType,
		ContentBase64: icsBase64,
	}}

	return "data:text/calendar;base64," + icsBase64, attachments
}

func (s *NotificationService) calendarFilename(event *domain.CalendarEvent) string {
	if event == nil {
		return "event.ics"
	}
	switch event.EventType {
	case domain.EventTypeOpenHouse:
		return fmt.Sprintf("open-house-%s.ics", event.ID.String())
	case domain.EventTypeShowing:
		return fmt.Sprintf("showing-%s.ics", event.ID.String())
	default:
		return fmt.Sprintf("event-%s.ics", event.ID.String())
	}
}

func (s *NotificationService) showingSummary(listingTitle string) string {
	return fmt.Sprintf("Property showing: %s", listingTitle)
}

func (s *NotificationService) showingDescription(listingTitle string) string {
	return fmt.Sprintf("Your showing for %s.", listingTitle)
}

func (s *NotificationService) openHouseTitle(event *domain.CalendarEvent, fallback string) string {
	if event != nil && event.OpenHouseDetails != nil && strings.TrimSpace(event.OpenHouseDetails.Title) != "" {
		return event.OpenHouseDetails.Title
	}
	return fmt.Sprintf("Open house: %s", fallback)
}

func (s *NotificationService) openHouseDescription(openHouseTitle, listingTitle string) string {
	return fmt.Sprintf("%s for %s.", openHouseTitle, listingTitle)
}

func (s *NotificationService) listingURL(listingID uuid.UUID) string {
	return fmt.Sprintf("%s/listings/%s", s.baseURL, listingID.String())
}

func (s *NotificationService) hostListingURL(listingID uuid.UUID) string {
	return fmt.Sprintf("%s/host/listings/%s", s.baseURL, listingID.String())
}

func (s *NotificationService) formatEventTime(t time.Time) string {
	return t.Format("Mon, Jan 2 2006 at 3:04 PM")
}

func (s *NotificationService) reminderLabel(minutes int) string {
	switch minutes {
	case 60:
		return "in 1 hour"
	case 1440:
		return "tomorrow"
	default:
		if minutes > 60 {
			hours := minutes / 60
			if minutes%60 == 0 {
				if hours == 1 {
					return "in 1 hour"
				}
				return fmt.Sprintf("in %d hours", hours)
			}
			return fmt.Sprintf("in %d hours %d minutes", hours, minutes%60)
		}
		return fmt.Sprintf("in %d minutes", minutes)
	}
}

func (s *NotificationService) formatOptionalPhone(phone *string) string {
	if phone == nil || strings.TrimSpace(*phone) == "" {
		return ""
	}
	return *phone
}

func (s *NotificationService) formatOptionalNotes(notes *string) string {
	if notes == nil || strings.TrimSpace(*notes) == "" {
		return ""
	}
	return *notes
}

func (s *NotificationService) fallbackName(name string) string {
	if strings.TrimSpace(name) == "" {
		return "there"
	}
	return name
}
