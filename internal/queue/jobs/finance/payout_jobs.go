package finance

import "time"

const (
	ProcessPayoutsJobType     = "payout_process"
	RetryDisbursementsJobType = "payout_retry"
)

// ProcessPayoutsJob triggers processing of all pending payouts
type ProcessPayoutsJob struct {
	// ProcessTime is the time to use for processing (optional, defaults to time.Now())
	ProcessTime *time.Time `json:"process_time,omitempty"`
}

// JobType returns the job type identifier
func (j ProcessPayoutsJob) JobType() string {
	return ProcessPayoutsJobType
}

// GetProcessTime returns the process time, defaulting to time.Now() if not provided
func (j ProcessPayoutsJob) GetProcessTime() time.Time {
	if j.ProcessTime == nil {
		return time.Now()
	}
	return *j.ProcessTime
}

// RetryDisbursementsJob triggers retry of failed disbursements
type RetryDisbursementsJob struct {
	// RetryTime is the time to use for retry (optional, defaults to time.Now())
	RetryTime *time.Time `json:"retry_time,omitempty"`
}

// JobType returns the job type identifier
func (j RetryDisbursementsJob) JobType() string {
	return RetryDisbursementsJobType
}

// GetRetryTime returns the retry time, defaulting to time.Now() if not provided
func (j RetryDisbursementsJob) GetRetryTime() time.Time {
	if j.RetryTime == nil {
		return time.Now()
	}
	return *j.RetryTime
}
