package promotions

import (
	"time"
)

const SubscriptionBillingJobType = "subscription_billing"

// SubscriptionBillingJob represents a job to process subscription billing for due subscriptions.
type SubscriptionBillingJob struct {
	// CheckTime is the time to use for checking due subscriptions (optional, defaults to time.Now())
	CheckTime *time.Time `json:"check_time,omitempty"`

	// Limit is the maximum number of subscriptions to process in one batch
	Limit int `json:"limit,omitempty"`
}

// JobType returns the job type identifier
func (j SubscriptionBillingJob) JobType() string {
	return SubscriptionBillingJobType
}

// Validate validates the job parameters
func (j SubscriptionBillingJob) Validate() error {
	// CheckTime is optional - will default to time.Now() in handler if nil
	return nil
}

// GetCheckTime returns the check time, defaulting to time.Now() if not provided
func (j SubscriptionBillingJob) GetCheckTime() time.Time {
	if j.CheckTime == nil {
		return time.Now()
	}
	return *j.CheckTime
}
