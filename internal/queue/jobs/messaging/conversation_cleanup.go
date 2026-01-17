package messaging

import "time"

const ConversationCleanupJobType = "conversation_cleanup"

// ConversationCleanupJob represents a job to archive stale conversations and delete old archived ones.
type ConversationCleanupJob struct {
	// CheckTime is the time to use for calculating cutoff dates (optional, defaults to time.Now())
	CheckTime *time.Time `json:"check_time,omitempty"`
}

// Validate checks the job parameters.
func (j ConversationCleanupJob) Validate() error {
	return nil
}

// GetCheckTime returns the check time, defaulting to time.Now() if not provided.
func (j ConversationCleanupJob) GetCheckTime() time.Time {
	if j.CheckTime == nil {
		return time.Now()
	}
	return *j.CheckTime
}
