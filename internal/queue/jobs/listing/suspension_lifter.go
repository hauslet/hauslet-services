package jobs

import "time"

const ListingSuspensionLifterJobType = "listing_suspension_lifter"

// ListingSuspensionLifterJob represents a job to lift expired listing suspensions.
type ListingSuspensionLifterJob struct {
	CheckTime *time.Time `json:"check_time,omitempty"`
}

// Type returns the job type identifier.
func (j ListingSuspensionLifterJob) Type() string {
	return ListingSuspensionLifterJobType
}

// Validate validates the job parameters.
func (j ListingSuspensionLifterJob) Validate() error {
	return nil
}

// GetCheckTime returns the check time, defaulting to time.Now().
func (j ListingSuspensionLifterJob) GetCheckTime() time.Time {
	if j.CheckTime == nil {
		return time.Now()
	}
	return *j.CheckTime
}
