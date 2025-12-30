package review

const (
	SendReviewRemindersJobType = "review_send_reminders"
)

// SendReviewRemindersJob triggers sending reminders to users who haven't reviewed
type SendReviewRemindersJob struct {
	// ReminderThresholdDays is when to send the reminder (e.g., 3 days before deadline)
	// Defaults to 3 days if not provided
	ReminderThresholdDays *int `json:"reminder_threshold_days,omitempty"`
}

// JobType returns the job type identifier
func (j SendReviewRemindersJob) JobType() string {
	return SendReviewRemindersJobType
}

// GetThresholdDays returns the number of days before deadline to send reminder
func (j SendReviewRemindersJob) GetThresholdDays() int {
	if j.ReminderThresholdDays != nil && *j.ReminderThresholdDays > 0 {
		return *j.ReminderThresholdDays
	}
	return 3 // Default: send reminder 3 days before deadline
}
