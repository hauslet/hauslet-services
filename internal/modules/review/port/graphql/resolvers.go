package graphql

import (
	"hauslet/internal/modules/review/service"
	"log/slog"
)

// Resolver handles GraphQL queries for review module
type Resolver struct {
	reviewService service.ReviewService
	log           *slog.Logger
}

// NewResolver creates a new GraphQL resolver for reviews
func NewResolver(
	reviewService service.ReviewService,
	log *slog.Logger,
) *Resolver {
	return &Resolver{
		reviewService: reviewService,
		log:           log,
	}
}
