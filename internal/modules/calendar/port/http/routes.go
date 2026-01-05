package http

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"hauslet/cmd/api/server/middleware"
	authservice "hauslet/internal/modules/auth/service"
	businessmiddleware "hauslet/internal/modules/business/middleware"
	calendarservice "hauslet/internal/modules/calendar/service"
	"hauslet/internal/platform/ratelimit"

	"github.com/go-chi/chi/v5"
)

// HTTPHandler exposes calendar-specific endpoints for hosts.
type HTTPHandler struct {
	ctx             context.Context
	calendarService calendarservice.CalendarService
	log             *slog.Logger
}

// NewHTTPHandler constructs a calendar HTTP handler.
func NewHTTPHandler(ctx context.Context, svc calendarservice.CalendarService, log *slog.Logger) *HTTPHandler {
	return &HTTPHandler{
		ctx:             ctx,
		calendarService: svc,
		log:             log,
	}
}

// SetupRoutes registers calendar endpoints under authenticated routes.
func (h *HTTPHandler) SetupRoutes(r chi.Router, authService authservice.AuthService, businessMW *businessmiddleware.Middleware) {
	authMiddleware := authService.OAuthService().Middleware()

	r.Group(func(r chi.Router) {
		r.Use(authMiddleware.Auth)
		if businessMW != nil && businessMW.Auth != nil {
			r.Use(businessMW.Auth.WithTenantSlug)
		}

		r.Route("/listings/{listingId}/calendar", func(r chi.Router) {
			r.Post("/blocks", h.createBlock)
			r.Delete("/blocks/{blockId}", h.deleteBlock)
		})
	})
}

// SetupRoutesWithRateLimiting registers calendar endpoints with rate limiting for production.
func (h *HTTPHandler) SetupRoutesWithRateLimiting(r chi.Router, authService authservice.AuthService, limiter ratelimit.Limiter, businessMW *businessmiddleware.Middleware) {
	authMiddleware := authService.OAuthService().Middleware()

	// Helper to apply rate limiting
	applyRateLimit := func(config middleware.RateLimitConfig) func(http.Handler) http.Handler {
		policy := middleware.RateLimitPolicy{
			Keys: func(r *http.Request) []ratelimit.LimitKey {
				ip := middleware.ClientIP(r)
				if ip == "" {
					return nil
				}
				return []ratelimit.LimitKey{
					{
						Type:   ratelimit.KeyTypeIP,
						Value:  ip,
						Limit:  int64(config.Requests),
						Window: config.Window,
					},
				}
			},
		}
		return middleware.RateLimitWithLimiter(limiter, policy)
	}

	r.Group(func(r chi.Router) {
		r.Use(authMiddleware.Auth)
		if businessMW != nil && businessMW.Auth != nil {
			r.Use(businessMW.Auth.WithTenantSlug)
		}

		r.Route("/listings/{listingId}/calendar", func(r chi.Router) {
			// Create block: 10 requests/minute
			r.With(applyRateLimit(middleware.RateLimitConfig{
				Requests: 10,
				Window:   time.Minute,
			})).Post("/blocks", h.createBlock)

			// Delete block: 10 requests/minute
			r.With(applyRateLimit(middleware.RateLimitConfig{
				Requests: 10,
				Window:   time.Minute,
			})).Delete("/blocks/{blockId}", h.deleteBlock)
		})
	})
}
