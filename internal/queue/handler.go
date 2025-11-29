package queue

import "context"

// JobHandler processes a specific job type
type JobHandler interface {
	// Handle processes the job
	Handle(ctx context.Context, data []byte) error

	// JobType returns the type this handler processes
	JobType() string

	// Subject returns the NATS subject this handler listens to
	Subject() string
}
