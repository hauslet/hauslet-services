package review

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	bookingRepo "hauslet/internal/modules/booking/repository"
	bookingSchema "hauslet/internal/modules/booking/repository/schema"
	reviewNotification "hauslet/internal/modules/review/notification"
	reviewRepo "hauslet/internal/modules/review/repository"
	reviewService "hauslet/internal/modules/review/service"
	reviewJobs "hauslet/internal/queue/jobs/review"
)

// ReminderHandler handles sending review reminders
type ReminderHandler struct {
	bookingRepo     bookingRepo.BookingRepository
	reviewRepo      reviewRepo.ReviewRepository
	bookingQuerier  reviewService.BookingQuerier
	userQuerier     reviewService.UserQuerier
	notificationSvc *reviewNotification.NotificationService
	log             *slog.Logger
	subject         string
}

// NewReminderHandler creates a new handler for sending review reminders
func NewReminderHandler(
	bookingRepo bookingRepo.BookingRepository,
	reviewRepo reviewRepo.ReviewRepository,
	bookingQuerier reviewService.BookingQuerier,
	userQuerier reviewService.UserQuerier,
	notificationSvc *reviewNotification.NotificationService,
	log *slog.Logger,
	subject string,
) *ReminderHandler {
	return &ReminderHandler{
		bookingRepo:     bookingRepo,
		reviewRepo:      reviewRepo,
		bookingQuerier:  bookingQuerier,
		userQuerier:     userQuerier,
		notificationSvc: notificationSvc,
		log:             log,
		subject:         subject,
	}
}

// JobType returns the job type identifier
func (h *ReminderHandler) JobType() string {
	return reviewJobs.SendReviewRemindersJobType
}

// Subject returns the queue subject this handler listens to
func (h *ReminderHandler) Subject() string {
	return h.subject
}

// Handle processes the review reminder job
func (h *ReminderHandler) Handle(ctx context.Context, data []byte) error {
	var job reviewJobs.SendReviewRemindersJob
	if err := json.Unmarshal(data, &job); err != nil {
		h.log.Error("failed to unmarshal SendReviewRemindersJob", "error", err)
		return err
	}

	thresholdDays := job.GetThresholdDays()
	reviewWindowDays := 14 // Standard Airbnb review window

	h.log.Info(
		"sending review reminders job started",
		"days_from_deadline", thresholdDays,
	)

	// Calculate the target date range
	// If thresholdDays=3 and window=14, we want bookings completed 11 days ago (14-3=11)
	targetDaysAgo := reviewWindowDays - thresholdDays
	targetDate := time.Now().AddDate(0, 0, -targetDaysAgo)

	// Find bookings completed on the target date (give 1 hour window for precision)
	startTime := targetDate.Add(-30 * time.Minute)
	endTime := targetDate.Add(30 * time.Minute)

	h.log.Info(
		"finding completed bookings in time window",
		"start", startTime,
		"end", endTime,
	)

	bookings, err := h.findCompletedBookingsInRange(ctx, startTime, endTime)
	if err != nil {
		h.log.Error("failed to find completed bookings", "error", err)
		return err
	}

	// Nothing to process
	if len(bookings) == 0 {
		h.log.Info("no completed bookings found for review reminders")
		return nil
	}

	h.log.Info(
		"found completed bookings to process for reminders",
		"count", len(bookings),
	)

	guestRemindersSent := 0
	hostRemindersSent := 0

	for _, booking := range bookings {
		// Guest reminder
		if booking.GuestReviewedAt == nil {
			if err := h.sendGuestReminder(ctx, booking, thresholdDays); err != nil {
				h.log.Warn(
					"failed to send guest review reminder",
					"booking_id", booking.ID,
					"error", err,
				)
			} else {
				guestRemindersSent++
			}
		}

		// Host reminder
		if booking.HostReviewedAt == nil {
			if err := h.sendHostReminder(ctx, booking, thresholdDays); err != nil {
				h.log.Warn(
					"failed to send host review reminder",
					"booking_id", booking.ID,
					"error", err,
				)
			} else {
				hostRemindersSent++
			}
		}
	}

	h.log.Info(
		"review reminder job completed",
		"guest_reminders_sent", guestRemindersSent,
		"host_reminders_sent", hostRemindersSent,
	)

	return nil
}

// findCompletedBookingsInRange finds bookings completed in the given time range
func (h *ReminderHandler) findCompletedBookingsInRange(ctx context.Context, start, end time.Time) ([]*bookingSchema.Booking, error) {
	// Limit to 500 bookings per run to prevent memory/performance issues
	// If there are more, they'll be picked up in subsequent runs
	const maxBookingsPerRun = 500
	return h.bookingRepo.FindCompletedBookingsInRange(ctx, start, end, maxBookingsPerRun)
}

// sendGuestReminder sends a reminder to the guest to review the listing
func (h *ReminderHandler) sendGuestReminder(ctx context.Context, booking *bookingSchema.Booking, daysLeft int) error {
	// Get guest contact info
	guestName, guestEmail, err := h.userQuerier.GetUserContact(ctx, booking.GuestID)
	if err != nil {
		return err
	}

	if guestEmail == "" {
		h.log.Warn("guest has no email, skipping reminder", "guest_id", booking.GuestID)
		return nil
	}

	// Get listing title (simplified - would need listing querier)
	listingTitle := "the property" // TODO: Get actual listing title

	// Create contact info
	guestContact := reviewNotification.ContactInfo{
		ID:    booking.GuestID,
		Name:  guestName,
		Email: guestEmail,
	}

	// Send reminder
	h.notificationSvc.SendReviewReminderToGuest(ctx, booking.ID, guestContact, listingTitle, daysLeft)

	return nil
}

// sendHostReminder sends a reminder to the host to review the guest
func (h *ReminderHandler) sendHostReminder(ctx context.Context, booking *bookingSchema.Booking, daysLeft int) error {
	// Get host ID from booking parties
	_, hostID, err := h.bookingQuerier.GetBookingParties(ctx, booking.ID)
	if err != nil {
		return err
	}

	// Get host contact info
	hostName, hostEmail, err := h.userQuerier.GetUserContact(ctx, hostID)
	if err != nil {
		return err
	}

	if hostEmail == "" {
		h.log.Warn("host has no email, skipping reminder", "host_id", hostID)
		return nil
	}

	// Get guest name (we already have it from booking)
	guestName := booking.GuestName

	// Create contact info
	hostContact := reviewNotification.ContactInfo{
		ID:    hostID,
		Name:  hostName,
		Email: hostEmail,
	}

	// Send reminder
	h.notificationSvc.SendReviewReminderToHost(ctx, booking.ID, hostContact, guestName, daysLeft)

	return nil
}
