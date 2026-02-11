package http

import (
	"context"
	"encoding/json"
	"hauslet/cmd/api/server/middleware"
	"hauslet/internal/modules/auth/domain"
	authmiddleware "hauslet/internal/modules/auth/middleware"
	"hauslet/internal/modules/auth/service"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"hauslet/internal/platform/ratelimit"

	"github.com/go-chi/chi/v5"
	authmw "github.com/go-pkgz/auth/v2/middleware"
	"github.com/go-pkgz/auth/v2/token"
)

// HTTPHandler handles HTTP requests for authentication
type HTTPHandler struct {
	authService service.AuthService
	ctx         context.Context
	log         *slog.Logger
}

// NewHTTPHandler creates a new HTTP handler for auth
func NewHTTPHandler(ctx context.Context, authService service.AuthService, log *slog.Logger) *HTTPHandler {
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
	//   - POST /auth/password/login              (password authentication with user+passwd fields)
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
	r.Post("/auth/register", h.Register)
	r.Post("/auth/forgot-password", h.ForgotPassword)
	r.Post("/auth/reset-password", h.ResetPassword)

	// Email verification endpoints (public)
	r.Post("/auth/verify-email", h.VerifyEmail)
	r.Post("/auth/resend-otp", h.ResendOTP)

	// Passwordless login code verification (public)
	r.Post("/auth/passwordless/verify", h.VerifyPasswordlessCode)

	// 2FA login verification (public - user is not yet authenticated)
	r.Post("/auth/2fa/verify", h.Verify2FALogin)
	r.Post("/auth/2fa/resend", h.Resend2FACode)

	// Protected routes (require authentication)
	authMiddleware := h.authService.OAuthService().Middleware()
	updater := h.userUpdater()

	r.Group(func(r chi.Router) {
		// r.Use(authmiddleware.LogRequestHeaders) // Uncomment for debugging header issues
		r.Use(authMiddleware.Auth, authMiddleware.UpdateUser(updater))

		// User profile management.
		r.Get("/me", h.GetCurrentUser)
		r.Put("/me", h.UpdateCurrentUser)
		r.Post("/me/change-password", h.ChangePassword)
		r.Post("/me/add-password", h.AddPassword)

		// Identity management (OAuth providers, password)
		r.Get("/me/identities", h.GetUserIdentities)
		r.Delete("/me/identities", h.UnlinkIdentity) // Query param: ?id=identity_id
		r.Get("/auth/link/{provider}", h.InitiateLinking)

		// Session management
		r.Get("/me/sessions", h.GetUserSessions)
		r.Delete("/me/sessions", h.RevokeAllSessions)
		r.Delete("/me/session", h.RevokeSession) // Query param: ?id=session_id

		// Two-Factor Authentication
		r.Get("/2fa/status", h.Get2FAStatus)
		r.Post("/2fa/setup", h.InitiateSetup2FA)
		r.Post("/2fa/setup/complete", h.CompleteSetup2FA)
		r.Post("/2fa/send-code", h.Send2FACode)
		r.Post("/2fa/verify", h.Verify2FACode)
		r.Post("/2fa/disable", h.Disable2FA)
		r.Post("/2fa/backup-codes", h.RegenerateBackupCodes)
	})

}

// SetupRoutesWithRateLimiting configures auth routes with rate limiting (caller decides when to use)
func (h *HTTPHandler) SetupRoutesWithRateLimiting(r chi.Router, limiter ratelimit.Limiter) {
	// Mount go-pkgz/auth's built-in routes with metadata capture
	authRoutes, avatarRoutes := h.authService.OAuthService().Handlers()

	// Wrap auth routes with metadata capture middleware
	r.Group(func(r chi.Router) {
		r.Use(authmiddleware.CaptureAuthMetadata(h.authService))
		r.Mount("/auth", authRoutes)
	})

	r.Mount("/avatar", avatarRoutes)

	// Registration: Aggressive rate limiting (prevent bot signups)
	r.With(middleware.RateLimitIP(limiter, 3, 15*time.Minute)).
		Post("/auth/register", h.Register)

	// Email verification: Moderate rate limiting (prevent brute force)
	r.With(middleware.RateLimitIP(limiter, 5, 10*time.Minute)).
		Post("/auth/verify-email", h.VerifyEmail)

	// Resend OTP: Strict rate limiting (prevent spam)
	r.With(middleware.RateLimitIP(limiter, 3, 10*time.Minute)).
		Post("/auth/resend-otp", h.ResendOTP)

	r.With(middleware.RateLimitIP(limiter, 5, 15*time.Minute)).
		Post("/auth/forgot-password", h.ForgotPassword)

	r.With(middleware.RateLimitIP(limiter, 5, 15*time.Minute)).
		Post("/auth/reset-password", h.ResetPassword)

	// Passwordless login code verification: Moderate rate limiting (prevent brute force)
	r.With(middleware.RateLimitIP(limiter, 5, 10*time.Minute)).
		Post("/auth/passwordless/verify", h.VerifyPasswordlessCode)

	// 2FA login verification: Strict rate limiting (brute force target)
	r.With(middleware.RateLimitIP(limiter, 10, 15*time.Minute)).
		Post("/auth/2fa/verify", h.Verify2FALogin)
	r.With(middleware.RateLimitIP(limiter, 3, 10*time.Minute)).
		Post("/auth/2fa/resend", h.Resend2FACode)

	// Protected routes
	authMiddleware := h.authService.OAuthService().Middleware()
	updater := h.userUpdater()

	r.Group(func(r chi.Router) {
		r.Use(authMiddleware.Auth, authMiddleware.UpdateUser(updater))

		// Baseline rate limiting for all authenticated requests
		r.Use(middleware.RateLimitIP(limiter, 60, time.Minute))

		// Password change: Strict rate limiting (security-sensitive)
		r.With(middleware.RateLimitIP(limiter, 5, time.Hour)).
			Post("/me/change-password", h.ChangePassword)

		// Add password: Strict rate limiting (security-sensitive, one-time action)
		r.With(middleware.RateLimitIP(limiter, 3, time.Hour)).
			Post("/me/add-password", h.AddPassword)

		// Profile updates: Moderate rate limiting
		r.With(middleware.RateLimitIP(limiter, 20, time.Minute)).
			Put("/me", h.UpdateCurrentUser)

		// Other endpoints (no additional rate limiting)
		r.Get("/me", h.GetCurrentUser)
		r.Get("/me/identities", h.GetUserIdentities)
		r.Delete("/me/identities", h.UnlinkIdentity)
		r.Get("/auth/link/{provider}", h.InitiateLinking)
		r.Get("/me/sessions", h.GetUserSessions)
		r.Delete("/me/sessions", h.RevokeAllSessions)
		r.Delete("/me/session", h.RevokeSession)

		// Two-Factor Authentication
		r.Route("/2fa", func(r chi.Router) {
			// Read-only: light rate limiting
			r.Get("/status", h.Get2FAStatus)

			// Setup initiation: moderate rate limiting
			r.With(middleware.RateLimitIP(limiter, 5, 10*time.Minute)).
				Post("/setup", h.InitiateSetup2FA)

			// Setup completion: strict rate limiting (brute force target)
			r.With(middleware.RateLimitIP(limiter, 5, 15*time.Minute)).
				Post("/setup/complete", h.CompleteSetup2FA)

			// Send code: strict rate limiting (prevent spam)
			r.With(middleware.RateLimitIP(limiter, 3, 10*time.Minute)).
				Post("/send-code", h.Send2FACode)

			// Verification: strict rate limiting (brute force target)
			r.With(middleware.RateLimitIP(limiter, 10, 15*time.Minute)).
				Post("/verify", h.Verify2FACode)

			// Disable: very strict rate limiting (security-critical)
			r.With(middleware.RateLimitIP(limiter, 3, time.Hour)).
				Post("/disable", h.Disable2FA)

			// Backup codes regeneration: strict rate limiting
			r.With(middleware.RateLimitIP(limiter, 3, time.Hour)).
				Post("/backup-codes", h.RegenerateBackupCodes)
		})
	})
}

// userUpdater enriches token.User with canonical user data from the database.
func (h *HTTPHandler) userUpdater() authmw.UserUpdater {
	return authmw.UserUpdFunc(func(u token.User) token.User {
		uid := u.StrAttr("uid")
		if uid == "" && u.ID != "" {
			if parts := strings.SplitN(u.ID, "_", 2); len(parts) == 2 {
				uid = parts[1]
			}
		}
		if uid == "" {
			return u
		}

		user, err := h.authService.GetUser(context.Background(), uid)
		if err != nil || user == nil {
			return u
		}

		u.SetStrAttr("uid", user.ID.String())
		u.SetStrAttr("email", user.PrimaryEmail)
		u.SetStrAttr("role", string(user.Role))
		u.Email = user.PrimaryEmail
		u.Name = user.Name
		return u
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
