package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-pkgz/auth/token"
)

// InitiateLinking initiates OAuth provider linking flow
// @Summary Initiate identity linking
// @Description Start OAuth flow to link a provider (google) to authenticated user account
// @Tags user
// @Param provider path string true "Provider name (google)"
// @Param redirect_uri query string false "URI to redirect after linking (default: /settings)"
// @Produce json
// @Success 302 {string} string "Redirect to OAuth provider"
// @Failure 400 {object} domain.ErrorResponse
// @Failure 401 {object} domain.ErrorResponse
// @Failure 500 {object} domain.ErrorResponse
// @Router /auth/link/{provider} [get]
// @Security BearerAuth
func (h *HTTPHandler) InitiateLinking(w http.ResponseWriter, r *http.Request) {
	// Extract authenticated user
	userInfo, err := token.GetUserInfo(r)
	if err != nil {
		h.log.Logf("ERROR Failed to get user info: %v", err)
		h.sendError(w, "Unauthorized", http.StatusUnauthorized, "")
		return
	}

	userID := userInfo.StrAttr("uid")
	if userID == "" {
		userID = userInfo.ID
	}

	// Get provider from URL
	provider := chi.URLParam(r, "provider")
	if provider == "" {
		h.sendError(w, "Provider is required", http.StatusBadRequest, "provider")
		return
	}

	// Get redirect URI (default to settings page)
	redirectURI := r.URL.Query().Get("redirect_uri")
	if redirectURI == "" {
		redirectURI = "/settings"
	}

	// Generate OAuth linking URL
	oauthURL, err := h.authService.InitiateIdentityLinking(userID, provider, redirectURI)
	if err != nil {
		h.log.Logf("ERROR Failed to initiate linking for user %s: %v", userID, err)
		h.sendError(w, "Failed to initiate linking", http.StatusInternalServerError, "")
		return
	}

	h.log.Logf("INFO User %s initiating %s linking", userID, provider)

	// Redirect to OAuth provider
	http.Redirect(w, r, oauthURL, http.StatusTemporaryRedirect)
}
