package review

import "time"

const (
	PublishStandoffsJobType = "review_publish_standoffs"
)

// PublishStandoffsJob triggers publication of reviews stuck in standoff
type PublishStandoffsJob struct {
	// StandoffThresholdDays is the number of days after which to publish standoff reviews
	// Defaults to 14 days if not provided
	StandoffThresholdDays *int `json:"standoff_threshold_days,omitempty"`
}

// JobType returns the job type identifier
func (j PublishStandoffsJob) JobType() string {
	return PublishStandoffsJobType
}

// GetThresholdTime returns the cutoff time for standoff expiration
func (j PublishStandoffsJob) GetThresholdTime() time.Time {
	days := 14 // Default: 14 days
	if j.StandoffThresholdDays != nil && *j.StandoffThresholdDays > 0 {
		days = *j.StandoffThresholdDays
	}
	return time.Now().AddDate(0, 0, -days)
}
