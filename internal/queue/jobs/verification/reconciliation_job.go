package verification

import (
	"fmt"
	"time"
)

const ReconciliationJobType = "verification_reconciliation"

// ReconciliationJob represents a job to reconcile verification data
// Fixes drift between verification sessions and profile module
type ReconciliationJob struct {
	BatchSize   int       `json:"batch_size"`
	WindowStart time.Time `json:"window_start,omitempty"` // Optional: only reconcile within time window
	WindowEnd   time.Time `json:"window_end,omitempty"`
	TraceID     string    `json:"trace_id,omitempty"`
}

// Type returns the job type identifier
func (j *ReconciliationJob) Type() string {
	return ReconciliationJobType
}

// Validate checks if the reconciliation job is valid
func (j *ReconciliationJob) Validate() error {
	if j.BatchSize <= 0 {
		j.BatchSize = 100 // Default batch size
	}
	if j.BatchSize > 1000 {
		return fmt.Errorf("batch_size cannot exceed 1000")
	}
	if !j.WindowStart.IsZero() && !j.WindowEnd.IsZero() {
		if j.WindowEnd.Before(j.WindowStart) {
			return fmt.Errorf("window_end must be after window_start")
		}
	}
	return nil
}
