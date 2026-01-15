package http

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"hauslet/cmd/api/server/middleware"
	authservice "hauslet/internal/modules/auth/service"
	"hauslet/internal/modules/messaging/domain"
	"hauslet/internal/modules/messaging/service"
	"hauslet/internal/platform/ratelimit"

	"github.com/go-chi/chi/v5"
)

// HTTPHandler exposes messaging endpoints that are not part of GraphQL.
type HTTPHandler struct {
	messagingSvc service.MessagingService
	ctx          context.Context
	log          *slog.Logger
}

// NewHTTPHandler creates a new messaging HTTP handler.
func NewHTTPHandler(ctx context.Context, messagingSvc service.MessagingService, log *slog.Logger) *HTTPHandler {
	return &HTTPHandler{
		messagingSvc: messagingSvc,
		ctx:          ctx,
		log:          log,
	}
}

// SetupRoutes configures public messaging-related HTTP endpoints.
func (h *HTTPHandler) SetupRoutes(r chi.Router, authService authservice.AuthService) {
	authMiddleware := authService.OAuthService().Middleware()

	r.Group(func(r chi.Router) {
		r.Use(authMiddleware.Auth)
		r.Post("/messaging/attachments/upload-url", h.GenerateUploadURL)
	})
}

// SetupRoutesWithRateLimiting configures messaging endpoints with rate limiting.
func (h *HTTPHandler) SetupRoutesWithRateLimiting(r chi.Router, authService authservice.AuthService, limiter ratelimit.Limiter) {
	authMiddleware := authService.OAuthService().Middleware()

	r.Group(func(r chi.Router) {
		r.Use(authMiddleware.Auth)
		r.With(middleware.RateLimitIP(limiter, 60, time.Minute)).
			Post("/messaging/attachments/upload-url", h.GenerateUploadURL)
	})
}

//---- Helper methods for HTTPHandler ----//

func (h *HTTPHandler) handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrUnauthorizedAccess):
		h.sendError(w, err.Error(), http.StatusForbidden, "")
	case errors.Is(err, domain.ErrConversationNotFound):
		h.sendError(w, err.Error(), http.StatusNotFound, "")
	default:
		h.sendError(w, err.Error(), http.StatusBadRequest, "")
	}
}

func (h *HTTPHandler) sendSuccess(w http.ResponseWriter, data any, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(data)
}

func (h *HTTPHandler) sendError(w http.ResponseWriter, message string, statusCode int, field string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	resp := struct {
		Error   string `json:"error"`
		Message string `json:"message"`
		Field   string `json:"field,omitempty"`
	}{
		Error:   http.StatusText(statusCode),
		Message: message,
		Field:   field,
	}

	_ = json.NewEncoder(w).Encode(resp)
}
