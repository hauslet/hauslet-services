package domain

import (
	"fmt"
	"regexp"
	"strings"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

// ValidateRegistration validates a registration request
func (r *RegisterRequest) Validate() error {
	if err := validateEmail(r.Email); err != nil {
		return NewValidationError("email", err.Error())
	}

	if err := validatePassword(r.Password); err != nil {
		return NewValidationError("password", err.Error())
	}

	if err := validateName(r.Name); err != nil {
		return NewValidationError("name", err.Error())
	}

	return nil
}

// ValidateLogin validates a login request
func (r *LoginRequest) Validate() error {
	if err := validateEmail(r.Email); err != nil {
		return NewValidationError("email", err.Error())
	}

	if r.Password == "" {
		return NewValidationError("password", ErrPasswordRequired.Error())
	}

	return nil
}

// ValidateUpdate validates an update user request
func (r *UpdateUserRequest) Validate() error {
	// At least one field must be provided
	if r.Email == "" && r.Name == "" {
		return fmt.Errorf("at least one field must be provided")
	}

	// Validate email if provided
	if r.Email != "" {
		if err := validateEmail(r.Email); err != nil {
			return NewValidationError("email", err.Error())
		}
	}

	// Validate name if provided
	if r.Name != "" {
		if err := validateName(r.Name); err != nil {
			return NewValidationError("name", err.Error())
		}
	}

	return nil
}

// ValidateChangePassword validates a password change request
func (r *ChangePasswordRequest) Validate() error {
	if r.OldPassword == "" {
		return NewValidationError("old_password", "current password is required")
	}

	if r.NewPassword == "" {
		return NewValidationError("new_password", "new password is required")
	}

	if err := validatePassword(r.NewPassword); err != nil {
		return NewValidationError("new_password", err.Error())
	}

	if r.OldPassword == r.NewPassword {
		return NewValidationError("new_password", "new password must be different from current password")
	}

	return nil
}

// Helper validation functions

func validateEmail(email string) error {
	if email == "" {
		return ErrEmailRequired
	}

	email = strings.TrimSpace(email)
	if !emailRegex.MatchString(email) {
		return fmt.Errorf("invalid email format")
	}

	return nil
}

func validatePassword(password string) error {
	if password == "" {
		return ErrPasswordRequired
	}

	if len(password) < 8 {
		return ErrPasswordTooShort
	}

	// Optional: Add more complex password requirements
	// - Must contain at least one uppercase letter
	// - Must contain at least one lowercase letter
	// - Must contain at least one digit
	// - Must contain at least one special character

	return nil
}

func validateName(name string) error {
	if name == "" {
		return ErrNameRequired
	}

	name = strings.TrimSpace(name)
	if len(name) < 2 {
		return fmt.Errorf("name must be at least 2 characters")
	}

	if len(name) > 100 {
		return fmt.Errorf("name must not exceed 100 characters")
	}

	return nil
}
