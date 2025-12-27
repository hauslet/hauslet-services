package payments

import (
	"encoding/json"
	"fmt"
	"time"
)

const PaymentWebhookJobType = "payment_webhook"

// PaymentWebhookJob represents a job to process a payment provider webhook.
type PaymentWebhookJob struct {
	Provider   string          `json:"provider"`
	EventType  string          `json:"event_type"`
	Reference  string          `json:"reference"`
	Payload    json.RawMessage `json:"payload"`
	ReceivedAt time.Time       `json:"received_at"`
}

// JobType returns the job type identifier.
func (j PaymentWebhookJob) JobType() string {
	return PaymentWebhookJobType
}

// Validate validates the job parameters.
func (j PaymentWebhookJob) Validate() error {
	if j.Provider == "" {
		return fmt.Errorf("provider is required")
	}
	if j.EventType == "" {
		return fmt.Errorf("event_type is required")
	}
	if j.Reference == "" {
		return fmt.Errorf("reference is required")
	}
	if len(j.Payload) == 0 {
		return fmt.Errorf("payload is required")
	}
	if j.ReceivedAt.IsZero() {
		return fmt.Errorf("received_at is required")
	}
	return nil
}
