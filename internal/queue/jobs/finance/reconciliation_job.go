package finance

import "time"

const (
	ReconciliationJobType = "finance_reconciliation"
)

// ReconciliationJob triggers daily financial reconciliation
type ReconciliationJob struct {
	// ReconciliationTime is the time to use for reconciliation (optional, defaults to time.Now())
	ReconciliationTime *time.Time `json:"reconciliation_time,omitempty"`
}

// JobType returns the job type identifier
func (j ReconciliationJob) JobType() string {
	return ReconciliationJobType
}

// GetReconciliationTime returns the reconciliation time, defaulting to time.Now() if not provided
func (j ReconciliationJob) GetReconciliationTime() time.Time {
	if j.ReconciliationTime == nil {
		return time.Now()
	}
	return *j.ReconciliationTime
}
