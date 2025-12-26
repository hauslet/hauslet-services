package booking

import (
	"fmt"
	"time"
)

const BookingExpiryCheckJobType = "booking_expiry_check"

// BookingExpiryCheckJob represents a job to check and archive expired booking holds.
type BookingExpiryCheckJob struct {
	// CheckTime is the time to use as the expiration threshold
	CheckTime time.Time `json:"check_time"`

	// Limit is the maximum number of bookings to process in one batch
	Limit int `json:"limit,omitempty"`
}

// JobType returns the job type identifier
func (j BookingExpiryCheckJob) JobType() string {
	return BookingExpiryCheckJobType
}

// Validate validates the job parameters
func (j BookingExpiryCheckJob) Validate() error {
	if j.CheckTime.IsZero() {
		return fmt.Errorf("check_time is required")
	}
	return nil
}
