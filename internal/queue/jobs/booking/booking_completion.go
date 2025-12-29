package booking

const BookingCompletionJobType = "booking_completion"

// BookingCompletionJob represents a job to check and complete active bookings.
type BookingCompletionJob struct {
	// Limit is the maximum number of bookings to process in one batch
	Limit int `json:"limit,omitempty"`
}

// JobType returns the job type identifier
func (j BookingCompletionJob) JobType() string {
	return BookingCompletionJobType
}

// Validate validates the job parameters
func (j BookingCompletionJob) Validate() error {
	// No validation needed - limit is optional and defaults to 100 in the service
	return nil
}
