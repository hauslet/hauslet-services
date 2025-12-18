package ai

import (
	"context"
)

// ---------------------------------------------------------
// INTERFACE
// ---------------------------------------------------------

type AIClient interface {
	// Moderate analyzes content.
	// Returns error ONLY for infrastructure failures (timeout, API down).
	// Logic failures (AI says "I don't know") should be handled in the Result status.
	Moderate(ctx context.Context, input AIModerationInput) (*AIModerationResult, error)

	HealthCheck(ctx context.Context) error
	Provider() string
}
