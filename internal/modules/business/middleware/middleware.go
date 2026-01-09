package middleware

import (
	"hauslet/internal/modules/business/service"
	"log/slog"
)

// Middleware holds all business middleware components
type Middleware struct {
	// Authentication middleware for business routes
	Auth *BusinessAuthMiddleware
	// GraphQL authentication helper
	GraphQL *GraphQLAuthHelper
	// Property authentication helper
	Property *PropertyAuthHelper
}

// NewMiddleware creates a new middleware instance with all components
func NewMiddleware(businessService service.BusinessService, log *slog.Logger) *Middleware {
	return &Middleware{
		Auth:     NewBusinessAuthMiddleware(businessService, log),
		GraphQL:  NewGraphQLAuthHelper(businessService),
		Property: NewPropertyAuthHelper(businessService),
	}
}
