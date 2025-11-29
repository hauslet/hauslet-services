package port

import (
	"context"
	"encoding/json"
	"hauslet/cmd/api/server/middleware"
	"hauslet/internal/auth/domain"
	authmiddleware "hauslet/internal/auth/middleware"
	"hauslet/internal/auth/service"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-pkgz/lgr"
	"github.com/redis/go-redis/v9"
)

// HTTPHandler handles HTTP requests for authentication
type HTTPHandler struct {
	authService service.AuthService
	ctx         context.Context
	log         *lgr.Logger
}

// NewHTTPHandler creates a new HTTP handler for auth
func NewHTTPHandler(ctx context.Context, authService service.AuthService, log *lgr.Logger) *HTTPHandler {
	return &HTTPHandler{
		authService: authService,
		ctx:         ctx,
		log:         log,
	}
}

// SetupRoutes configures all auth-related routes
func (h *HTTPHandler) SetupRoutes(r chi.Router) {
	// Mount go-pkgz/auth's built-in authentication routes with metadata capture
	// Provides:
	//   - POST /auth/login              (password authentication with user+passwd fields)
	//   - GET  /auth/google/login       (Google OAuth initiation)
	//   - GET  /auth/google/callback    (Google OAuth callback)
	//   - GET  /auth/logout             (logout and clear session)
	authRoutes, avatarRoutes := h.authService.OAuthService().Handlers()

	// Wrap auth routes with metadata capture middleware
	r.Group(func(r chi.Router) {
		r.Use(authmiddleware.CaptureAuthMetadata(h.authService))
		r.Mount("/auth", authRoutes)
	})

	r.Mount("/avatar", avatarRoutes)

	// Custom registration endpoint (go-pkgz/auth doesn't provide registration)
	r.Post("/register", h.Register)

	// Email verification endpoints (public)
	r.Post("/auth/verify-email", h.VerifyEmail)
	r.Post("/auth/resend-otp", h.ResendOTP)

	// Protected routes (require authentication)
	authMiddleware := h.authService.OAuthService().Middleware()

	r.Group(func(r chi.Router) {
		r.Use(authmiddleware.LogRequestHeaders)
		r.Use(authMiddleware.Auth)

		// User profile management.
		r.Get("/me", h.GetCurrentUser)
		r.Put("/me", h.UpdateCurrentUser)
		r.Post("/change-password", h.ChangePassword)

		// Identity management (OAuth providers, password)
		r.Get("/me/identities", h.GetUserIdentities)
		r.Delete("/me/identities", h.UnlinkIdentity) // Query param: ?id=identity_id
		r.Get("/auth/link/{provider}", h.InitiateLinking)

		// Session management
		r.Get("/me/sessions", h.GetUserSessions)
		r.Delete("/me/sessions", h.RevokeAllSessions)
		r.Delete("/me/session", h.RevokeSession) // Query param: ?id=session_id
	})
}

// SetupRoutesWithRateLimiting configures auth routes with production rate limiting
// Rate limiting only applies when env is "production"
func (h *HTTPHandler) SetupRoutesWithRateLimiting(r chi.Router, redisClient *redis.Client, env string) {
	// Mount go-pkgz/auth's built-in routes with metadata capture
	authRoutes, avatarRoutes := h.authService.OAuthService().Handlers()

	// Wrap auth routes with metadata capture middleware
	r.Group(func(r chi.Router) {
		r.Use(authmiddleware.CaptureAuthMetadata(h.authService))
		r.Mount("/auth", authRoutes)
	})

	r.Mount("/avatar", avatarRoutes)

	// Helper to conditionally apply rate limiting
	applyRateLimit := func(config middleware.RateLimitConfig) func(http.Handler) http.Handler {
		if env == "production" {
			return middleware.RateLimit(config, redisClient)
		}
		// In development, return a no-op middleware
		return func(next http.Handler) http.Handler {
			return next
		}
	}

	// Registration: Aggressive rate limiting (prevent bot signups)
	r.With(applyRateLimit(middleware.RateLimitConfig{
		Requests: 3,
		Window:   15 * time.Minute,
	})).Post("/register", h.Register)

	// Email verification: Moderate rate limiting (prevent brute force)
	r.With(applyRateLimit(middleware.RateLimitConfig{
		Requests: 5,
		Window:   10 * time.Minute,
	})).Post("/auth/verify-email", h.VerifyEmail)

	// Resend OTP: Strict rate limiting (prevent spam)
	r.With(applyRateLimit(middleware.RateLimitConfig{
		Requests: 3,
		Window:   10 * time.Minute,
	})).Post("/auth/resend-otp", h.ResendOTP)

	// Protected routes
	authMiddleware := h.authService.OAuthService().Middleware()
	r.Group(func(r chi.Router) {
		r.Use(authMiddleware.Auth)

		// Password change: Strict rate limiting (security-sensitive)
		r.With(applyRateLimit(middleware.RateLimitConfig{
			Requests: 5,
			Window:   time.Hour,
		})).Post("/change-password", h.ChangePassword)

		// Profile updates: Moderate rate limiting
		r.With(applyRateLimit(middleware.RateLimitConfig{
			Requests: 20,
			Window:   time.Minute,
		})).Put("/me", h.UpdateCurrentUser)

		// Other endpoints (no additional rate limiting)
		r.Get("/me", h.GetCurrentUser)
		r.Get("/me/identities", h.GetUserIdentities)
		r.Delete("/me/identities", h.UnlinkIdentity)
		r.Get("/auth/link/{provider}", h.InitiateLinking)
		r.Get("/me/sessions", h.GetUserSessions)
		r.Delete("/me/sessions", h.RevokeAllSessions)
		r.Delete("/me/session", h.RevokeSession)
	})
}

// sendError sends an error response
func (h *HTTPHandler) sendError(w http.ResponseWriter, message string, statusCode int, field string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	errResponse := domain.ErrorResponse{
		Error:   http.StatusText(statusCode),
		Message: message,
	}

	if field != "" {
		errResponse.Field = field
	}

	json.NewEncoder(w).Encode(errResponse)
}

// sendSuccess sends a success response
func (h *HTTPHandler) sendSuccess(w http.ResponseWriter, data interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}
