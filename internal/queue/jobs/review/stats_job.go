package review

import "github.com/google/uuid"

const (
	RecalculateStatsJobType = "review_recalculate_stats"
)

// RecalculateStatsJob triggers stats recalculation for a listing or host
type RecalculateStatsJob struct {
	TargetType string    `json:"target_type"` // "listing" or "host"
	TargetID   uuid.UUID `json:"target_id"`
}

// JobType returns the job type identifier
func (j RecalculateStatsJob) JobType() string {
	return RecalculateStatsJobType
}
