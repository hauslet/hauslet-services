package breaker

import "context"

// CircuitBreaker defines the interface for circuit breaker operations
type CircuitBreaker interface {
	// AllowRequest checks if a request is allowed for the provider
	AllowRequest(ctx context.Context, providerName string) (bool, error)

	// RecordSuccess records a successful request
	RecordSuccess(ctx context.Context, providerName string) error

	// RecordFailure records a failed request
	RecordFailure(ctx context.Context, providerName string) error

	// GetState returns the current state of the circuit breaker
	GetState(ctx context.Context, providerName string) (State, error)

	// GetStats returns statistics for the circuit breaker
	GetStats(ctx context.Context, providerName string) (*Stats, error)

	// Reset manually resets the circuit breaker to closed state
	Reset(ctx context.Context, providerName string) error
}
