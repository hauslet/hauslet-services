package http

import (
	"context"
	"encoding/json"
	"hauslet/internal/modules/property/domain"
	"hauslet/internal/modules/property/service"
	"net/http"
	"strings"

	"github.com/go-pkgz/auth/token"
	"github.com/go-pkgz/lgr"
	"github.com/google/uuid"
)

// HTTPHandler handles HTTP requests for property operations
type HTTPHandler struct {
	propertyService service.Service
	ctx             context.Context
	log             *lgr.Logger
}

// NewHTTPHandler creates a new HTTP handler for property operations
func NewHTTPHandler(ctx context.Context, propertyService service.Service, log *lgr.Logger) *HTTPHandler {
	return &HTTPHandler{
		propertyService: propertyService,
		ctx:             ctx,
		log:             log,
	}
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

// extractUserID extracts the user ID from the JWT token in the request
func (h *HTTPHandler) extractUserID(r *http.Request) (uuid.UUID, error) {
	userInfo, err := token.GetUserInfo(r)
	if err != nil {
		return uuid.Nil, domain.ErrUnauthorized
	}

	userID := userInfo.StrAttr("uid")
	if userID == "" {
		// Fallback to ID field if uid attribute is not set
		userID = userInfo.ID
		// Extract the UUID part if ID is in "provider_uuid" format
		if parts := strings.SplitN(userID, "_", 2); len(parts) == 2 {
			userID = parts[1]
		}
	}

	if userID == "" {
		return uuid.Nil, domain.ErrUnauthorized
	}

	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		return uuid.Nil, domain.ErrUnauthorized
	}

	return parsedUserID, nil
}
