package finance

import "time"

const PenaltyDebtCollectionJobType = "penalty_debt_collection"

// PenaltyDebtCollectionJob triggers collection of unpaid penalties.
type PenaltyDebtCollectionJob struct {
	// CheckTime is the time to use for reporting (optional, defaults to time.Now())
	CheckTime *time.Time `json:"check_time,omitempty"`
}

// JobType returns the job type identifier.
func (j PenaltyDebtCollectionJob) JobType() string {
	return PenaltyDebtCollectionJobType
}

// GetCheckTime returns the check time, defaulting to time.Now().
func (j PenaltyDebtCollectionJob) GetCheckTime() time.Time {
	if j.CheckTime == nil {
		return time.Now()
	}
	return *j.CheckTime
}
