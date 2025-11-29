package queue

import (
	"context"
	"fmt"
)

// Registry maps subjects to handlers
type Registry struct {
	handlers map[string]JobHandler
}

// NewRegistry creates a new handler registry
func NewRegistry() *Registry {
	return &Registry{
		handlers: make(map[string]JobHandler),
	}
}

// Register adds a handler to the registry
func (r *Registry) Register(handler JobHandler) {
	r.handlers[handler.Subject()] = handler
}

// Handle routes a message to the appropriate handler based on subject
func (r *Registry) Handle(ctx context.Context, subject string, data []byte) error {
	handler, ok := r.handlers[subject]
	if !ok {
		return fmt.Errorf("no handler registered for subject: %s", subject)
	}
	return handler.Handle(ctx, data)
}

// Subjects returns all registered subjects
func (r *Registry) Subjects() []string {
	subjects := make([]string, 0, len(r.handlers))
	for subject := range r.handlers {
		subjects = append(subjects, subject)
	}
	return subjects
}

// HandlerCount returns the number of registered handlers
func (r *Registry) HandlerCount() int {
	return len(r.handlers)
}
