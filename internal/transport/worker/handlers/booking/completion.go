package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"hauslet/internal/modules/booking/service"
	bookingJob "hauslet/internal/queue/jobs/booking"
)

// BookingCompletionHandler marks active bookings as completed when eligible.
type BookingCompletionHandler struct {
	bookingSvc service.BookingService
	log        *slog.Logger
	subject    string
}

// NewBookingCompletionHandler constructs a completion handler.
func NewBookingCompletionHandler(bookingSvc service.BookingService, log *slog.Logger, subject string) *BookingCompletionHandler {
	return &BookingCompletionHandler{
		bookingSvc: bookingSvc,
		log:        log,
		subject:    subject,
	}
}

func (h *BookingCompletionHandler) JobType() string {
	return bookingJob.BookingCompletionJobType
}

func (h *BookingCompletionHandler) Subject() string {
	return h.subject
}

func (h *BookingCompletionHandler) Handle(ctx context.Context, data []byte) error {
	var job bookingJob.BookingCompletionJob
	if err := json.Unmarshal(data, &job); err != nil {
		return fmt.Errorf("unmarshal booking completion job: %w", err)
	}
	if err := job.Validate(); err != nil {
		return fmt.Errorf("invalid booking completion job: %w", err)
	}

	h.log.Info("processing booking completion check")

	// Complete eligible bookings
	err := h.bookingSvc.CompleteBookings(ctx)
	if err != nil {
		return fmt.Errorf("failed to complete bookings: %w", err)
	}

	h.log.Info("booking completion check finished successfully")

	return nil
}
