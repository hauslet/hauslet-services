package breaker

import "time"

// State represents the circuit breaker state
type State string

const (
	StateClosed   State = "closed"    // Normal operation
	StateOpen     State = "open"      // Failing, requests blocked
	StateHalfOpen State = "half_open" // Testing if service recovered
)

// String returns the string representation
func (s State) String() string {
	return string(s)
}

// Stats contains circuit breaker statistics
type Stats struct {
	ProviderName  string
	State         State
	FailureCount  int64
	SuccessCount  int64
	LastFailureAt *time.Time
	LastSuccessAt *time.Time
	OpenedAt      *time.Time
	LastCheckedAt time.Time
}

// StateTransition represents a state change event
type StateTransition struct {
	ProviderName string
	FromState    State
	ToState      State
	Reason       string
	TransitionAt time.Time
}
