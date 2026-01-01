package booking

const BookingCheckInOutJobType = "booking_checkin_out"

// BookingCheckInOutJob triggers auto-population of check-in/out timestamps.
type BookingCheckInOutJob struct{}

// JobType returns the job type identifier.
func (j BookingCheckInOutJob) JobType() string {
	return BookingCheckInOutJobType
}

// Validate validates the job parameters.
func (j BookingCheckInOutJob) Validate() error {
	return nil
}
