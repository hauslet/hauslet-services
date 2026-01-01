package promotions

import (
	"time"
)

const PromotionExpiryJobType = "promotion_expiry"

// PromotionExpiryJob represents a job to expire promotions that have passed their expiry date.
type PromotionExpiryJob struct {
	// CheckTime is the time to use as the expiration threshold (optional, defaults to time.Now())
	CheckTime *time.Time `json:"check_time,omitempty"`

	// Limit is the maximum number of promotions to process in one batch
	Limit int `json:"limit,omitempty"`
}

// JobType returns the job type identifier
func (j PromotionExpiryJob) JobType() string {
	return PromotionExpiryJobType
}

// Validate validates the job parameters
func (j PromotionExpiryJob) Validate() error {
	// CheckTime is optional - will default to time.Now() in handler if nil
	return nil
}

// GetCheckTime returns the check time, defaulting to time.Now() if not provided
func (j PromotionExpiryJob) GetCheckTime() time.Time {
	if j.CheckTime == nil {
		return time.Now()
	}
	return *j.CheckTime
}
