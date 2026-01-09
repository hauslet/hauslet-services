package http

import (
	"context"
	"log/slog"
	"time"

	"hauslet/cmd/api/server/middleware"
	authservice "hauslet/internal/modules/auth/service"
	businessmiddleware "hauslet/internal/modules/business/middleware"
	leadservice "hauslet/internal/modules/leads/service"
	"hauslet/internal/platform/ratelimit"

	"github.com/go-chi/chi/v5"
)

// HTTPHandler exposes lead-specific REST endpoints for external integrations (ads, webhooks, etc.)
type HTTPHandler struct {
	ctx         context.Context
	leadService leadservice.LeadService
	log         *slog.Logger
}

// NewHTTPHandler constructs a lead HTTP handler
func NewHTTPHandler(ctx context.Context, svc leadservice.LeadService, log *slog.Logger) *HTTPHandler {
	return &HTTPHandler{
		ctx:         ctx,
		leadService: svc,
		log:         log,
	}
}

// SetupRoutes registers lead endpoints (both public and authenticated)
func (h *HTTPHandler) SetupRoutes(r chi.Router, authService authservice.AuthService, businessMW *businessmiddleware.Middleware) {
	authMiddleware := authService.OAuthService().Middleware()

	// Consolidated /leads routes - mix of public and authenticated endpoints
	r.Route("/leads", func(r chi.Router) {
		// Public route (no authentication) - for ad integrations
		r.Post("/", h.createLead)

		// Authenticated routes
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware.Auth)
			if businessMW != nil && businessMW.Auth != nil {
				r.Use(businessMW.Auth.WithTenantSlug)
			}

			r.Get("/{leadId}", h.getLead)
		})
	})
}

// SetupRoutesWithRateLimiting registers lead endpoints with rate limiting for production
func (h *HTTPHandler) SetupRoutesWithRateLimiting(r chi.Router, authService authservice.AuthService, limiter ratelimit.Limiter, businessMW *businessmiddleware.Middleware) {
	authMiddleware := authService.OAuthService().Middleware()

	// Consolidated /leads routes with rate limiting - mix of public and authenticated endpoints
	r.Route("/leads", func(r chi.Router) {
		// Public route with aggressive rate limiting (ad webhook protection)
		// Create lead: 20 requests/minute per IP (prevents spam from ad platforms)
		r.With(middleware.RateLimitIP(limiter, 20, time.Minute)).
			Post("/", h.createLead)

		// Authenticated routes with standard rate limiting
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware.Auth)
			if businessMW != nil && businessMW.Auth != nil {
				r.Use(businessMW.Auth.WithTenantSlug)
			}

			// Get lead: 60 requests/minute
			r.With(middleware.RateLimitIP(limiter, 60, time.Minute)).
				Get("/{leadId}", h.getLead)
		})
	})
}