package http

import (
	"hauslet/internal/modules/auth/domain"

	"github.com/go-pkgz/auth/v2/token"
)

var (
	_ = domain.ErrorResponse{}
	_ = token.User{}
)

// This file is used ONLY for Swagger documentation of external library routes.
// The actual implementation is handled by go-pkgz/auth library which does not provide Swagger docs out of the box.
// These dummy functions allow 'swag' to generate documentation for these routes.

// @Summary Login with Email/Password
// @Description Authenticate using email/username and password.
// @Tags auth
// @Accept x-www-form-urlencoded,json
// @Produce json
// @Param user formData string true "User Email or Username"
// @Param passwd formData string true "Password"
// @Param aud formData string false "Optional Audience (Cookie Domain)"
// @Success 200 {object} token.User
// @Failure 400 {object} domain.ErrorResponse
// @Failure 401 {object} domain.ErrorResponse
// @Router /auth/password/login [post]
func _() {}

// @Summary Get Current User (Auth Library)
// @Description Get the currently authenticated user's details from the session/token.
// @Tags auth
// @Produce json
// @Success 200 {object} token.User
// @Failure 401 {object} domain.ErrorResponse
// @Router /auth/user [get]
func _() {}

// @Summary Logout
// @Description Logout and clear the session cookie/token.
// @Tags auth
// @Produce json
// @Success 200 {string} string "OK"
// @Router /auth/logout [get]
func _() {}

// @Summary List Auth Providers
// @Description List all enabled authentication providers (e.g., google, github, email).
// @Tags auth
// @Produce json
// @Success 200 {array} string
// @Router /auth/list [get]
func _() {}

// @Summary OAuth Login
// @Description Initiate OAuth login flow for a specific provider.
// @Tags auth
// @Param provider path string true "Provider Name (e.g., google, github)"
// @Router /auth/{provider}/login [get]
func _() {}

// @Summary OAuth Callback
// @Description Callback endpoint for OAuth providers. Handled automatically, but documented for completeness.
// @Tags auth
// @Param provider path string true "Provider Name"
// @Param code query string true "Authorization Code"
// @Param state query string true "State"
// @Router /auth/{provider}/callback [get]
func _() {}

// @Summary Get User Avatar
// @Description Proxy to get user avatar from external provider (if configured)
// @Tags auth
// @Param user_hash path string true "User Hash"
// @Router /avatar/{user_hash} [get]
func _() {}
