package sms

import "context"

// Provider defines the interface that all SMS providers must implement
type Provider interface {
	// Send sends an SMS message
	Send(ctx context.Context, req SMSRequest) (*SMSResponse, error)

	// HealthCheck verifies provider API is reachable
	HealthCheck(ctx context.Context) error

	// Provider returns the provider name
	Provider() string
}
