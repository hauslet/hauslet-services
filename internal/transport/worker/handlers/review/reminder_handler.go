package review

import (
	"context"
	"encoding/json"
	"time"

	bookingRepo "hauslet/internal/modules/booking/repository"
	bookingSchema "hauslet/internal/modules/booking/repository/schema"
	reviewNotification "hauslet/internal/modules/review/notification"
	reviewRepo "hauslet/internal/modules/review/repository"
	reviewService "hauslet/internal/modules/review/service"
	reviewJobs "hauslet/internal/queue/jobs/review"

	"github.com/go-pkgz/lgr"
)

// ReminderHandler handles sending review reminders
type ReminderHandler struct {
	bookingRepo     bookingRepo.BookingRepository
	reviewRepo      reviewRepo.ReviewRepository
	bookingQuerier  reviewService.BookingQuerier
	userQuerier     reviewService.UserQuerier
	notificationSvc *reviewNotification.NotificationService
	log             *lgr.Logger
	subject         string
}

// NewReminderHandler creates a new handler for sending review reminders
func NewReminderHandler(
	bookingRepo bookingRepo.BookingRepository,
	reviewRepo reviewRepo.ReviewRepository,
	bookingQuerier reviewService.BookingQuerier,
	userQuerier reviewService.UserQuerier,
	notificationSvc *reviewNotification.NotificationService,
	log *lgr.Logger,
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
		h.log.Logf("ERROR failed to unmarshal SendReviewRemindersJob: %v", err)
		return err
	}

	thresholdDays := job.GetThresholdDays()
	reviewWindowDays := 14 // Standard Airbnb review window

	h.log.Logf("INFO sending review reminders for bookings %d days from deadline", thresholdDays)

	// Calculate the target date range
	// If thresholdDays=3 and window=14, we want bookings completed 11 days ago (14-3=11)
	targetDaysAgo := reviewWindowDays - thresholdDays
	targetDate := time.Now().AddDate(0, 0, -targetDaysAgo)

	// Find bookings completed on the target date (give 1 hour window for precision)
	startTime := targetDate.Add(-30 * time.Minute)
	endTime := targetDate.Add(30 * time.Minute)

	h.log.Logf("INFO finding completed bookings between %s and %s", startTime, endTime)

	// Get completed bookings in the date range
	// This is a simplified implementation - in production, you'd want a dedicated repository method
	bookings, err := h.findCompletedBookingsInRange(ctx, startTime, endTime)
	if err != nil {
		h.log.Logf("ERROR failed to find completed bookings: %v", err)
		return err
	}

	h.log.Logf("INFO found %d completed bookings to check for reminders", len(bookings))

	guestRemindersSent := 0
	hostRemindersSent := 0

	// Process each booking
	for _, booking := range bookings {
		// Check if guest needs reminder
		if booking.GuestReviewedAt == nil {
			if err := h.sendGuestReminder(ctx, booking, thresholdDays); err != nil {
				h.log.Logf("WARN failed to send guest reminder for booking %s: %v", booking.ID, err)
			} else {
				guestRemindersSent++
			}
		}

		// Check if host needs reminder
		if booking.HostReviewedAt == nil {
			if err := h.sendHostReminder(ctx, booking, thresholdDays); err != nil {
				h.log.Logf("WARN failed to send host reminder for booking %s: %v", booking.ID, err)
			} else {
				hostRemindersSent++
			}
		}
	}

	h.log.Logf("INFO review reminder job completed: %d guest reminders, %d host reminders sent",
		guestRemindersSent, hostRemindersSent)

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
		h.log.Logf("WARN guest %s has no email, skipping reminder", booking.GuestID)
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
		h.log.Logf("WARN host %s has no email, skipping reminder", hostID)
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
