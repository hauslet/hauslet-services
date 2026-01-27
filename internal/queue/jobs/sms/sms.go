package jobs

import "fmt"

const SMSJobType = "sms"

// SMSJob represents a generic SMS sending job
type SMSJob struct {
	PhoneNumber string `json:"phone_number"`
	Message     string `json:"message"`
	Provider    string `json:"provider,omitempty"` // Optional: "termii", "twilio"
	TraceID     string `json:"trace_id,omitempty"`
}

// Type returns the job type identifier
func (j *SMSJob) Type() string {
	return SMSJobType
}

// Validate checks if the SMS job is valid
func (j *SMSJob) Validate() error {
	if j.PhoneNumber == "" {
		return fmt.Errorf("phone_number is required")
	}
	if j.Message == "" {
		return fmt.Errorf("message is required")
	}
	return nil
}
