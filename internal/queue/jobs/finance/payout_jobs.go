package finance

import "time"

const (
	ProcessPayoutsJobType     = "payout_process"
	RetryDisbursementsJobType = "payout_retry"
)

// ProcessPayoutsJob triggers processing of all pending payouts
type ProcessPayoutsJob struct {
	ProcessTime time.Time `json:"process_time"`
}

// JobType returns the job type identifier
func (j ProcessPayoutsJob) JobType() string {
	return ProcessPayoutsJobType
}

// RetryDisbursementsJob triggers retry of failed disbursements
type RetryDisbursementsJob struct {
	RetryTime time.Time `json:"retry_time"`
}

// JobType returns the job type identifier
func (j RetryDisbursementsJob) JobType() string {
	return RetryDisbursementsJobType
}
