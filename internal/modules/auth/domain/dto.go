package domain

import "time"

// RegisterRequest represents the registration request payload
type RegisterRequest struct {
	Email     string     `json:"email" validate:"required,email"`
	Password  string     `json:"password" validate:"required,min=8"`
	Name      string     `json:"name" validate:"required"`
	BirthDate *time.Time `json:"birth_date,omitempty"`
}

// RegisterResponse represents the registration response
type RegisterResponse struct {
	ID      string `json:"id"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Message string `json:"message"`
}

// LoginRequest represents the login request payload
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// LoginResponse represents the login response
type LoginResponse struct {
	Token        string  `json:"token,omitempty"`
	RefreshToken string  `json:"refresh_token,omitempty"`
	User         UserDTO `json:"user"`
	Message      string  `json:"message"`
}

// UserDTO represents a user data transfer object
type UserDTO struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	PrimaryEmail string `json:"primary_email"`
	AvatarURL    string `json:"avatar_url,omitempty"`
	Role         string `json:"role"`
	IsActive     bool   `json:"is_active"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
	Field   string `json:"field,omitempty"`
}

// ChangePasswordRequest represents a password change request
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=8"`
}

// ChangePasswordResponse represents a password change response
type ChangePasswordResponse struct {
	Message string `json:"message"`
}

// ChangeRoleRequest represents a role change request
type ChangeRoleRequest struct {
	Role string `json:"role" validate:"required"`
}

// ChangeRoleResponse represents a role change response
type ChangeRoleResponse struct {
	Message string `json:"message"`
}

// UserResponse represents a user profile response
type UserResponse struct {
	ID        string  `json:"id"`
	Email     string  `json:"email"`
	Name      string  `json:"name"`
	Role      string  `json:"role"`
	AvatarURL *string `json:"avatar_url,omitempty"`
}

// UpdateUserRequest represents a user profile update request
type UpdateUserRequest struct {
	Email string `json:"email,omitempty" validate:"omitempty,email"`
	Name  string `json:"name,omitempty"`
}

// UserIdentityResponse represents a user identity (authentication method)
type UserIdentityResponse struct {
	ID            string `json:"id"`
	Provider      string `json:"provider"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
}

// SessionResponse represents an active session
type SessionResponse struct {
	ID        string    `json:"id"`
	Provider  string    `json:"provider"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	UserAgent string    `json:"user_agent"`
	IP        string    `json:"ip"`
}
