package handlers

import (
	"context"
	"encoding/json"
	"fmt"

	"hauslet/internal/modules/booking/service"
	bookingJob "hauslet/internal/queue/jobs/booking"

	"github.com/go-pkgz/lgr"
)

// BookingExpiryCheckHandler archives expired booking holds.
type BookingExpiryCheckHandler struct {
	bookingSvc service.BookingService
	log        *lgr.Logger
	subject    string
}

// NewBookingExpiryCheckHandler constructs an expiry check handler.
func NewBookingExpiryCheckHandler(bookingSvc service.BookingService, log *lgr.Logger, subject string) *BookingExpiryCheckHandler {
	return &BookingExpiryCheckHandler{
		bookingSvc: bookingSvc,
		log:        log,
		subject:    subject,
	}
}

func (h *BookingExpiryCheckHandler) JobType() string {
	return bookingJob.BookingExpiryCheckJobType
}

func (h *BookingExpiryCheckHandler) Subject() string {
	return h.subject
}

func (h *BookingExpiryCheckHandler) Handle(ctx context.Context, data []byte) error {
	var job bookingJob.BookingExpiryCheckJob
	if err := json.Unmarshal(data, &job); err != nil {
		return fmt.Errorf("unmarshal booking expiry job: %w", err)
	}
	if err := job.Validate(); err != nil {
		return fmt.Errorf("invalid booking expiry job: %w", err)
	}

	h.log.Logf("INFO processing booking expiry check for time=%s", job.CheckTime.Format("2006-01-02 15:04:05"))

	// Archive expired bookings
	archivedIDs, err := h.bookingSvc.ArchiveExpiredBookings(ctx, job.CheckTime)
	if err != nil {
		return fmt.Errorf("failed to archive expired bookings: %w", err)
	}

	if len(archivedIDs) == 0 {
		h.log.Logf("INFO no expired bookings found")
		return nil
	}

	h.log.Logf("INFO archived %d expired bookings", len(archivedIDs))
	for _, id := range archivedIDs {
		h.log.Logf("INFO archived booking: %s", id)
	}

	return nil
}
