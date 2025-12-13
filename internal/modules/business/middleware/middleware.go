package middleware

import (
	"hauslet/internal/modules/business/service"

	"github.com/go-pkgz/lgr"
)

// Middleware holds all business middleware components
type Middleware struct {
	Auth     *BusinessAuthMiddleware
	GraphQL  *GraphQLAuthHelper
	Property *PropertyAuthHelper
}

// NewMiddleware creates a new middleware instance with all components
func NewMiddleware(businessService service.BusinessService, log *lgr.Logger) *Middleware {
	return &Middleware{
		Auth:     NewBusinessAuthMiddleware(businessService, log),
		GraphQL:  NewGraphQLAuthHelper(businessService),
		Property: NewPropertyAuthHelper(businessService),
	}
}
