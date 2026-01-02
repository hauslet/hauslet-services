package calendar

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"hauslet/internal/modules/calendar/domain"
	calendarnotification "hauslet/internal/modules/calendar/notification"
	calendarrepo "hauslet/internal/modules/calendar/repository"
	calendarschema "hauslet/internal/modules/calendar/repository/schema"
	calendarjobs "hauslet/internal/queue/jobs/calendar"
)

// ReminderHandler handles calendar reminder jobs for a specific event type.
type ReminderHandler struct {
	repo      calendarrepo.CalendarRepository
	notifier  *calendarnotification.NotificationService
	log       *slog.Logger
	subject   string
	eventType calendarschema.EventType
	jobType   string
}

// NewReminderHandler creates a new reminder handler.
func NewReminderHandler(
	repo calendarrepo.CalendarRepository,
	notifier *calendarnotification.NotificationService,
	log *slog.Logger,
	subject string,
	eventType calendarschema.EventType,
	jobType string,
) *ReminderHandler {
	return &ReminderHandler{
		repo:      repo,
		notifier:  notifier,
		log:       log,
		subject:   subject,
		eventType: eventType,
		jobType:   jobType,
	}
}

// JobType returns the job type identifier.
func (h *ReminderHandler) JobType() string {
	return h.jobType
}

// Subject returns the queue subject this handler listens to.
func (h *ReminderHandler) Subject() string {
	return h.subject
}

// Handle processes reminder jobs.
func (h *ReminderHandler) Handle(ctx context.Context, data []byte) error {
	if h.repo == nil {
		return fmt.Errorf("calendar repository not configured")
	}
	if h.notifier == nil {
		if h.log != nil {
			h.log.Warn("calendar notifier not configured; skipping reminders")
		}
		return nil
	}

	var job calendarjobs.ReminderJob
	if err := json.Unmarshal(data, &job); err != nil {
		return fmt.Errorf("failed to unmarshal reminder job: %w", err)
	}

	reminderMinutes := job.GetReminderMinutes()
	windowMinutes := job.GetWindowMinutes()

	target := time.Now().UTC().Add(time.Duration(reminderMinutes) * time.Minute)
	window := time.Duration(windowMinutes) * time.Minute
	start := target.Add(-window / 2)
	end := target.Add(window / 2)

	events, err := h.repo.GetEventsStartingBetween(ctx, start, end, []calendarschema.EventType{h.eventType}, []calendarschema.EventStatus{calendarschema.EventStatusConfirmed})
	if err != nil {
		return fmt.Errorf("failed to query events: %w", err)
	}

	if len(events) == 0 {
		return nil
	}

	for _, event := range events {
		domainEvent := domain.MapEventFromSchemaToEntity(event)
		if domainEvent == nil {
			continue
		}

		switch h.eventType {
		case calendarschema.EventTypeShowing:
			if domainEvent.ShowingDetails == nil {
				continue
			}
			h.notifier.SendShowingReminder(ctx, domainEvent, reminderMinutes)
		case calendarschema.EventTypeOpenHouse:
			if domainEvent.OpenHouseDetails == nil {
				continue
			}
			for _, attendee := range domainEvent.OpenHouseDetails.Attendees {
				if attendee.Email == "" {
					continue
				}
				h.notifier.SendOpenHouseReminder(ctx, domainEvent, attendee, reminderMinutes)
			}
		}
	}

	return nil
}
