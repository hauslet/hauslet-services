package queue

// Job represents any work that can be queued
type Job interface {
	// Type returns the job type identifier
	Type() string

	// Validate checks if the job is valid
	Validate() error
}

// BaseJob provides common fields for all jobs
type BaseJob struct {
	TraceID   string `json:"trace_id,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
}
