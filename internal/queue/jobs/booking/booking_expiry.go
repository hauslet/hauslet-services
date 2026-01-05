package booking

import (
	"time"
)

const BookingExpiryCheckJobType = "booking_expiry_check"

// BookingExpiryCheckJob represents a job to check and archive expired booking holds.
type BookingExpiryCheckJob struct {
	// CheckTime is the time to use as the expiration threshold (optional, defaults to time.Now())
	CheckTime *time.Time `json:"check_time,omitempty"`

	// Limit is the maximum number of bookings to process in one batch
	Limit int `json:"limit,omitempty"`
}

// JobType returns the job type identifier
func (j BookingExpiryCheckJob) JobType() string {
	return BookingExpiryCheckJobType
}

// Validate validates the job parameters
func (j BookingExpiryCheckJob) Validate() error {
	// CheckTime is optional - will default to time.Now() in handler if nil
	return nil
}

// GetCheckTime returns the check time, defaulting to time.Now() if not provided
func (j BookingExpiryCheckJob) GetCheckTime() time.Time {
	if j.CheckTime == nil {
		return time.Now()
	}
	return *j.CheckTime
}
