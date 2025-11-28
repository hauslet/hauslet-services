package domain

import (
	"time"

	"github.com/google/uuid"
)

type Session struct {
	ID        string    `json:"id"` // opaque session ID (the one in the cookie / header)
	UserID    uuid.UUID `json:"user_id"`
	UserAgent string    `json:"user_agent,omitempty"`
	IP        string    `json:"ip,omitempty"`

	// OAuth tokens for the current session (if authenticated via OAuth provider)
	// These are stored in Redis (not DB) for better security and performance
	Provider     string     `json:"provider,omitempty"`      // "google", "github", etc. (empty for local auth)
	AccessToken  string     `json:"access_token,omitempty"`  // Provider access token
	RefreshToken string     `json:"refresh_token,omitempty"` // Provider refresh token
	TokenExpiry  *time.Time `json:"token_expiry,omitempty"`  // When provider token expires

	// When this session should be considered expired
	ExpiresAt time.Time `json:"expires_at"`

	// Optional metadata
	CreatedAt time.Time `json:"created_at"`
}
