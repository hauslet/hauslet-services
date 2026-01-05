package calendar

const (
	ShowingReminderJobType   = "calendar_showing_reminders"
	OpenHouseReminderJobType = "calendar_open_house_reminders"
)

// ReminderJob triggers sending calendar reminders.
type ReminderJob struct {
	// ReminderMinutes is how long before the event to notify (e.g., 60 or 1440).
	ReminderMinutes *int `json:"reminder_minutes,omitempty"`

	// WindowMinutes defines the time window to match events around the target time.
	WindowMinutes *int `json:"window_minutes,omitempty"`
}

func (j ReminderJob) GetReminderMinutes() int {
	if j.ReminderMinutes != nil && *j.ReminderMinutes > 0 {
		return *j.ReminderMinutes
	}
	return 60
}

func (j ReminderJob) GetWindowMinutes() int {
	if j.WindowMinutes != nil && *j.WindowMinutes > 0 {
		return *j.WindowMinutes
	}
	return 30
}
