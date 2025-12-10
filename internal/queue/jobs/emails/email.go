package jobs

import "fmt"

const EmailJobType = "email"

// EmailJob represents an email sending job
type EmailJob struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	HTML    string `json:"html"`
	TraceID string `json:"trace_id,omitempty"`
}

// Type returns the job type identifier
func (j *EmailJob) Type() string {
	return EmailJobType
}

// Validate checks if the email job is valid
func (j *EmailJob) Validate() error {
	if j.To == "" {
		return fmt.Errorf("to address is required")
	}
	if j.Subject == "" {
		return fmt.Errorf("subject is required")
	}
	if j.HTML == "" {
		return fmt.Errorf("HTML body is required")
	}
	return nil
}
