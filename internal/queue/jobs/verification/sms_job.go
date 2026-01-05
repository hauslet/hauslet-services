package verification

import "fmt"

const SMSJobType = "verification_sms"

// SMSJob represents a job to send a verification SMS
type SMSJob struct {
	PhoneNumber string `json:"phone_number"`
	OTP         string `json:"otp"`
	UserID      string `json:"user_id"`
	Provider    string `json:"provider"`    // termii or twilio
	RetryCount  int    `json:"retry_count"` // For provider fallback
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
	if j.OTP == "" {
		return fmt.Errorf("otp is required")
	}
	if j.UserID == "" {
		return fmt.Errorf("user_id is required")
	}
	if j.Provider == "" {
		j.Provider = "termii" // Default to primary provider
	}
	if j.Provider != "termii" && j.Provider != "twilio" {
		return fmt.Errorf("provider must be either 'termii' or 'twilio'")
	}
	return nil
}

// ShouldFallback determines if we should fallback to secondary provider
func (j *SMSJob) ShouldFallback() bool {
	return j.RetryCount > 0 && j.Provider == "termii"
}

// GetFallbackProvider returns the fallback provider
func (j *SMSJob) GetFallbackProvider() string {
	if j.Provider == "termii" {
		return "twilio"
	}
	return "termii"
}
