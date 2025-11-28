package domain

import (
	"time"

	"github.com/google/uuid"
)

// UserRole represents user authorization level
type UserRole string

const (
	RoleUser      UserRole = "user"
	RoleStaff     UserRole = "staff"
	RoleModerator UserRole = "moderator"
	RoleSupport   UserRole = "support"
	RoleAdmin     UserRole = "admin"
	RoleRoot      UserRole = "root"
)

// User represents a user in the system (domain model - safe for business logic)
// SECURITY: Does not expose sensitive fields like password hashes or soft deletes
type User struct {
	ID           uuid.UUID  `json:"id"`
	Name         string     `json:"name"`
	PrimaryEmail string     `json:"primary_email"`
	Role         UserRole   `json:"role"`
	IsActive     bool       `json:"is_active"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`

	// Connected OAuth identities (without sensitive data)
	Identities []UserIdentity `json:"identities,omitempty"`
}

// UserIdentity represents an authentication method for a user
// SECURITY: Does not expose PasswordHash or ProviderID (internal identifiers)
type UserIdentity struct {
	ID            uuid.UUID  `json:"id"`
	UserID        uuid.UUID  `json:"user_id"`
	Provider      string     `json:"provider"`       // "google", "github", "password"
	Email         string     `json:"email"`          // Email from this provider
	EmailVerified bool       `json:"email_verified"` // Whether provider verified this email
	LastUsedAt    *time.Time `json:"last_used_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

// PublicUser represents user data safe to expose to API clients
// SECURITY: Minimal exposure - only data needed for UI display
type PublicUser struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	PrimaryEmail string    `json:"primary_email"`
	CreatedAt    time.Time `json:"created_at"`
}

// AuthenticatedUser represents a user with session context
// Used internally for authorization checks
type AuthenticatedUser struct {
	User
	SessionID string `json:"session_id"`
}
