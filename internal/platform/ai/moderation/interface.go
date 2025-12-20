package aimoderation

import (
	"context"
)

// AIClient defines the interface for AI moderation services
type AIClient interface {
	// Moderate analyzes content.
	// Returns error ONLY for infrastructure failures (timeout, API down).
	Moderate(ctx context.Context, input AIModerationInput) (*AIModerationResult, error)

	HealthCheck(ctx context.Context) error
	Provider() string
}
