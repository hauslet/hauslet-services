package domain

import "errors"

// Authentication errors
var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrUserNotFound       = errors.New("user not found")
	ErrUserAlreadyExists  = errors.New("user with this email already exists")
	ErrUserDeactivated    = errors.New("user account is deactivated")
	ErrInvalidToken       = errors.New("invalid or expired token")
	ErrSessionNotFound    = errors.New("session not found")
	ErrSessionExpired     = errors.New("session has expired")
)

// Authorization errors
var (
	ErrUnauthorized            = errors.New("unauthorized")
	ErrForbidden               = errors.New("forbidden: insufficient permissions")
	ErrCannotModifyRoot        = errors.New("cannot modify root user")
	ErrCannotDeactivateRoot    = errors.New("cannot deactivate root user")
	ErrCannotDemoteRoot        = errors.New("cannot demote root user through UpdateUser - this operation is not allowed")
	ErrOnlyRootCanPromote      = errors.New("only root users can promote to admin")
	ErrInsufficientPermissions = errors.New("insufficient permissions to perform this action")
)

// Validation errors
var (
	ErrEmailRequired       = errors.New("email is required")
	ErrPasswordRequired    = errors.New("password is required")
	ErrPasswordTooShort    = errors.New("password must be at least 8 characters long")
	ErrNameRequired        = errors.New("name is required")
	ErrInvalidEmail        = errors.New("invalid email format")
	ErrWeakPassword        = errors.New("password is too weak")
	ErrPasswordsDoNotMatch = errors.New("passwords do not match")
)

// Identity errors
var (
	ErrIdentityNotFound         = errors.New("identity not found")
	ErrCannotUnlinkLastIdentity = errors.New("cannot unlink the last identity: user must have at least one login method")
	ErrPasswordAuthNotEnabled   = errors.New("user does not have password authentication enabled")
	ErrInvalidOldPassword       = errors.New("invalid old password")
)

// Role errors
var (
	ErrInvalidRole           = errors.New("invalid role")
	ErrCannotChangeOwnRole   = errors.New("users cannot change their own roles")
	ErrCannotModifyAdminRole = errors.New("only root users can modify admin roles")
)

// General errors
var (
	ErrNilUser             = errors.New("user cannot be nil")
	ErrInvalidRequest      = errors.New("invalid request")
	ErrInternalServerError = errors.New("internal server error")
)

// ValidationError represents a field-specific validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

// NewValidationError creates a new validation error
func NewValidationError(field, message string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Message: message,
	}
}
