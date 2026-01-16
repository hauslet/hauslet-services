package http

import (
	"time"

	"hauslet/cmd/api/server/middleware"
	authmiddleware "hauslet/internal/modules/auth/middleware"
	"hauslet/internal/platform/ratelimit"

	"github.com/go-chi/chi/v5"
	authmw "github.com/go-pkgz/auth/v2/middleware"
)

// SetupRoutes configures HTTP routes for admin review operations
func (h *AdminHandler) SetupRoutes(r chi.Router, authMiddleware *authmw.Authenticator) {
	r.Group(func(r chi.Router) {
		// Apply authentication and admin role requirement
		r.Use(authMiddleware.Auth, authmiddleware.RBAC("admin"))

		// Review admin operations
		r.Post("/admin/reviews/{id}/publish", h.PublishReview)
		r.Post("/admin/reviews/{id}/hide", h.HideReview)
		r.Post("/admin/reviews/{id}/unhide", h.UnhideReview)
	})
}

// SetupRoutesWithRateLimiting configures HTTP routes with rate limiting
func (h *AdminHandler) SetupRoutesWithRateLimiting(r chi.Router, authMiddleware *authmw.Authenticator, limiter ratelimit.Limiter) {
	r.Group(func(r chi.Router) {
		r.Use(authMiddleware.Auth, authmiddleware.RBAC("admin"))

		// Now this is readable and DRY:
		r.With(middleware.RateLimitIP(limiter, 20, time.Minute)).
			Post("/admin/reviews/{id}/publish", h.PublishReview)

		r.With(middleware.RateLimitIP(limiter, 20, time.Minute)).
			Post("/admin/reviews/{id}/hide", h.HideReview)

		r.With(middleware.RateLimitIP(limiter, 20, time.Minute)).
			Post("/admin/reviews/{id}/unhide", h.UnhideReview)
	})
}
