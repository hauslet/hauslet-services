package graphql

import (
	"hauslet/internal/modules/review/service"

	"github.com/go-pkgz/lgr"
)

// Resolver handles GraphQL queries for review module
type Resolver struct {
	reviewService service.ReviewService
	log           *lgr.Logger
}

// NewResolver creates a new GraphQL resolver for reviews
func NewResolver(
	reviewService service.ReviewService,
	log *lgr.Logger,
) *Resolver {
	return &Resolver{
		reviewService: reviewService,
		log:           log,
	}
}
