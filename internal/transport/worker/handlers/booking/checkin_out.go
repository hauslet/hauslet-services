package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"hauslet/internal/modules/booking/service"
	bookingJob "hauslet/internal/queue/jobs/booking"
)

// BookingCheckInOutHandler auto-populates actual check-in/out timestamps.
type BookingCheckInOutHandler struct {
	bookingSvc service.BookingService
	log        *slog.Logger
	subject    string
}

// NewBookingCheckInOutHandler constructs a check-in/out handler.
func NewBookingCheckInOutHandler(bookingSvc service.BookingService, log *slog.Logger, subject string) *BookingCheckInOutHandler {
	return &BookingCheckInOutHandler{
		bookingSvc: bookingSvc,
		log:        log,
		subject:    subject,
	}
}

func (h *BookingCheckInOutHandler) JobType() string {
	return bookingJob.BookingCheckInOutJobType
}

func (h *BookingCheckInOutHandler) Subject() string {
	return h.subject
}

func (h *BookingCheckInOutHandler) Handle(ctx context.Context, data []byte) error {
	var job bookingJob.BookingCheckInOutJob
	if err := json.Unmarshal(data, &job); err != nil {
		return fmt.Errorf("unmarshal booking check-in/out job: %w", err)
	}
	if err := job.Validate(); err != nil {
		return fmt.Errorf("invalid booking check-in/out job: %w", err)
	}

	checkIns, checkOuts, err := h.bookingSvc.AutoPopulateCheckInOut(ctx)
	if err != nil {
		return err
	}

	if h.log != nil {
		h.log.Info("booking check-in/out auto-populate completed", "checkins", checkIns, "checkouts", checkOuts)
	}

	return nil
}
