package http

import (
	"hauslet/cmd/api/server/middleware"
	authservice "hauslet/internal/auth/service"
	"hauslet/internal/platform/redis"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

// SetupRoutes configures all property-related routes
func (h *HTTPHandler) SetupRoutes(r chi.Router, authService authservice.AuthService) {
	authMiddleware := authService.OAuthService().Middleware()

	// Protected routes (require authentication)
	r.Group(func(r chi.Router) {
		r.Use(authMiddleware.Auth)

		// Listing media routes
		r.Route("/api/listings/{id}/media", func(r chi.Router) {
			r.Post("/", h.UploadListingMedia)           // Upload media (get presigned URLs)
			r.Post("/finalize", h.FinalizeListingMedia) // Finalize uploaded media
			r.Delete("/", h.DeleteListingMedia)         // Delete media (bulk)
		})

		r.Route(`/api/listings/{id}/media/{mediaId:[0-9a-fA-F-]{36}}`, func(r chi.Router) {
			r.Patch("/", h.UpdateListingMedia) // Update media metadata
		})
	})
}

// SetupRoutesWithRateLimiting configures property routes with rate limiting (for production)
func (h *HTTPHandler) SetupRoutesWithRateLimiting(r chi.Router, authService authservice.AuthService, redisClient redis.RedisClient) {
	authMiddleware := authService.OAuthService().Middleware()

	// Helper to apply rate limiting
	applyRateLimit := func(config middleware.RateLimitConfig) func(http.Handler) http.Handler {
		return middleware.RateLimit(config, redisClient)
	}

	// Protected routes (require authentication)
	r.Group(func(r chi.Router) {
		r.Use(authMiddleware.Auth)

		// Listing media routes
		r.Route("/api/listings/{id}/media", func(r chi.Router) {
			// Upload media: 20 requests/minute
			r.With(applyRateLimit(middleware.RateLimitConfig{
				Requests: 20,
				Window:   time.Minute,
			})).Post("/", h.UploadListingMedia)

			// Finalize media: 20 requests/minute
			r.With(applyRateLimit(middleware.RateLimitConfig{
				Requests: 20,
				Window:   time.Minute,
			})).Post("/finalize", h.FinalizeListingMedia)

			// Delete media: 20 requests/minute
			r.With(applyRateLimit(middleware.RateLimitConfig{
				Requests: 20,
				Window:   time.Minute,
			})).Delete("/", h.DeleteListingMedia)
		})

		r.Route(`/api/listings/{id}/media/{mediaId:[0-9a-fA-F-]{36}}`, func(r chi.Router) {
			// Update media: 30 requests/minute
			r.With(applyRateLimit(middleware.RateLimitConfig{
				Requests: 30,
				Window:   time.Minute,
			})).Patch("/", h.UpdateListingMedia)
		})
	})
}
