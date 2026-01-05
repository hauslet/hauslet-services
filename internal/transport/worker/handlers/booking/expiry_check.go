package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"hauslet/internal/modules/booking/service"
	bookingJob "hauslet/internal/queue/jobs/booking"
)

// BookingExpiryCheckHandler archives expired booking holds.
type BookingExpiryCheckHandler struct {
	bookingSvc service.BookingService
	log        *slog.Logger
	subject    string
}

// NewBookingExpiryCheckHandler constructs an expiry check handler.
func NewBookingExpiryCheckHandler(bookingSvc service.BookingService, log *slog.Logger, subject string) *BookingExpiryCheckHandler {
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

	checkTime := job.GetCheckTime()
	h.log.Info("processing booking expiry check", "check_time", checkTime)

	// Archive expired bookings
	archivedIDs, err := h.bookingSvc.ArchiveExpiredBookings(ctx, checkTime)
	if err != nil {
		return fmt.Errorf("failed to archive expired bookings: %w", err)
	}

	if len(archivedIDs) == 0 {
		h.log.Info("no expired bookings found")
		return nil
	}

	h.log.Info("archived expired bookings", "count", len(archivedIDs))
	for _, id := range archivedIDs {
		h.log.Info("archived booking", "id", id)
	}

	return nil
}
