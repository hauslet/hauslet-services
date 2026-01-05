package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	bookingdomain "hauslet/internal/modules/booking/domain"
	bookingrepository "hauslet/internal/modules/booking/repository"
	paymentdomain "hauslet/internal/modules/payments/domain"
	paymentservice "hauslet/internal/modules/payments/service"
	bookingJob "hauslet/internal/queue/jobs/booking"
)

// BookingRefundHandler processes refund jobs for cancelled bookings.
type BookingRefundHandler struct {
	bookingRepo bookingrepository.BookingRepository
	paymentSvc  paymentservice.PaymentService
	log         *slog.Logger
	subject     string
}

// NewBookingRefundHandler constructs a refund handler.
func NewBookingRefundHandler(
	bookingRepo bookingrepository.BookingRepository,
	paymentSvc paymentservice.PaymentService,
	log *slog.Logger,
	subject string,
) *BookingRefundHandler {
	return &BookingRefundHandler{
		bookingRepo: bookingRepo,
		paymentSvc:  paymentSvc,
		log:         log,
		subject:     subject,
	}
}

func (h *BookingRefundHandler) JobType() string {
	return bookingJob.BookingRefundJobType
}

func (h *BookingRefundHandler) Subject() string {
	return h.subject
}

func (h *BookingRefundHandler) Handle(ctx context.Context, data []byte) error {
	var job bookingJob.BookingRefundJob
	if err := json.Unmarshal(data, &job); err != nil {
		return fmt.Errorf("unmarshal booking refund job: %w", err)
	}
	if err := job.Validate(); err != nil {
		return fmt.Errorf("invalid booking refund job: %w", err)
	}

	h.log.Info("processing booking refund",
		"booking", job.BookingID,
		"payment", job.PaymentID,
		"amount", job.Amount,
	)

	schemaBooking, err := h.bookingRepo.GetBookingByID(ctx, job.BookingID)
	if err != nil {
		return fmt.Errorf("get booking %s: %w", job.BookingID, err)
	}
	if schemaBooking == nil {
		return fmt.Errorf("booking %s not found", job.BookingID)
	}

	booking := bookingdomain.MapBookingFromSchema(schemaBooking)
	if booking.RefundProcessedAt != nil {
		h.log.Info("booking refund already processed", "booking", job.BookingID)
		return nil
	}

	if booking.LastPaymentID != nil && *booking.LastPaymentID != job.PaymentID {
		h.log.Warn("refund payment mismatch",
			"booking", job.BookingID,
			"payment", job.PaymentID,
			"expected", *booking.LastPaymentID,
		)
	}

	refundInput := paymentdomain.RefundPaymentInput{
		PaymentID:  job.PaymentID,
		Amount:     &job.Amount,
		Reason:     job.Reason,
		RefundedBy: job.RefundedBy,
	}

	refundedPayment, err := h.paymentSvc.RefundPayment(ctx, refundInput)
	if err != nil {
		if errors.Is(err, paymentdomain.ErrCannotRefundPayment) {
			existing, getErr := h.paymentSvc.GetPayment(ctx, job.PaymentID)
			if getErr != nil {
				return fmt.Errorf("get payment after refund failure: %w", getErr)
			}
			if existing.RefundedAt != nil || existing.RefundedAmount > 0 {
				return h.updateBookingRefund(ctx, booking, existing, job)
			}
		}
		return fmt.Errorf("refund payment: %w", err)
	}

	return h.updateBookingRefund(ctx, booking, refundedPayment, job)
}

func (h *BookingRefundHandler) updateBookingRefund(
	ctx context.Context,
	booking *bookingdomain.Booking,
	payment *paymentdomain.Payment,
	job bookingJob.BookingRefundJob,
) error {
	now := time.Now()
	ref := payment.ID.String()

	booking.RefundAmount = payment.RefundedAmount
	booking.RefundReference = &ref

	if booking.RefundInitiatedAt == nil {
		requestedAt := job.RequestedAt
		booking.RefundInitiatedAt = &requestedAt
	}

	if payment.RefundedAt != nil {
		booking.RefundProcessedAt = payment.RefundedAt
	} else {
		booking.RefundProcessedAt = &now
	}

	if booking.RefundReason == nil && job.Reason != "" {
		reason := job.Reason
		booking.RefundReason = &reason
	}

	booking.UpdatedAt = now

	if err := h.bookingRepo.UpdateBooking(ctx, bookingdomain.MapBookingFromDomain(booking)); err != nil {
		return fmt.Errorf("update booking refund %s: %w", booking.ID, err)
	}

	h.log.Info("booking refund recorded", "booking", booking.ID, "amount", booking.RefundAmount)
	return nil
}
