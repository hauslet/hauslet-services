package sms

import (
	"errors"
	"regexp"
	"time"
)

// E.164 phone number regex pattern
var e164Pattern = regexp.MustCompile(`^\+[1-9]\d{1,14}$`)

// SMSRequest contains the data for sending an SMS
type SMSRequest struct {
	To      string // E.164 format phone number (e.g., +2348012345678)
	Message string // SMS message content
	From    string // Sender ID (optional, uses default if empty)
}

// Validate checks if SMS request has required fields and valid format
func (r *SMSRequest) Validate() error {
	if r.To == "" {
		return errors.New("recipient phone number is required")
	}
	if !e164Pattern.MatchString(r.To) {
		return ErrInvalidPhoneNumber
	}
	if r.Message == "" {
		return errors.New("message content is required")
	}
	if len(r.Message) > 1600 {
		return ErrMessageTooLong // Max 10 SMS segments (160 * 10)
	}
	return nil
}

// SMSResponse contains the result of sending an SMS
type SMSResponse struct {
	Success   bool      // Whether SMS was sent successfully
	MessageID string    // Provider's message ID
	Status    string    // Delivery status (queued, sent, delivered, failed)
	Provider  string    // Provider used (termii, twilio)
	Cost      float64   // Cost in USD (if available)
	Segments  int       // Number of SMS segments used
	SentAt    time.Time // When SMS was sent
	Message   string    // Status message
}

// ProviderStats contains SMS provider performance metrics
type ProviderStats struct {
	Name          string
	SuccessRate   float64       // Last 24 hours
	AvgLatency    time.Duration // Average send time
	AvgCost       float64       // Average cost per SMS
	LastSuccess   *time.Time
	LastFailure   *time.Time
	LastUpdated   time.Time
}
