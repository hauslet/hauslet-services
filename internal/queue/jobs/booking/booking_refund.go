package booking

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

const BookingRefundJobType = "booking_refund"

// BookingRefundJob represents a job to process a booking refund.
type BookingRefundJob struct {
	BookingID  uuid.UUID `json:"booking_id"`
	PaymentID  uuid.UUID `json:"payment_id"`
	Amount     int64     `json:"amount"`
	Reason     string    `json:"reason,omitempty"`
	RefundedBy uuid.UUID `json:"refunded_by"`
	RequestedAt time.Time `json:"requested_at"`
}

// JobType returns the job type identifier.
func (j BookingRefundJob) JobType() string {
	return BookingRefundJobType
}

// Validate validates the job parameters.
func (j BookingRefundJob) Validate() error {
	if j.BookingID == uuid.Nil {
		return fmt.Errorf("booking_id is required")
	}
	if j.PaymentID == uuid.Nil {
		return fmt.Errorf("payment_id is required")
	}
	if j.Amount <= 0 {
		return fmt.Errorf("amount must be greater than zero")
	}
	if j.RefundedBy == uuid.Nil {
		return fmt.Errorf("refunded_by is required")
	}
	if j.RequestedAt.IsZero() {
		return fmt.Errorf("requested_at is required")
	}
	return nil
}
