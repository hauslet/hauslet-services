package service

import (
	"hauslet/internal/modules/leads/domain"
	"regexp"
	"strings"
)

// LeadValidator handles validation of lead input data
type LeadValidator struct {
	emailRegex *regexp.Regexp
	phoneRegex *regexp.Regexp
}

// NewLeadValidator creates a new instance of LeadValidator
func NewLeadValidator() *LeadValidator {
	return &LeadValidator{
		// RFC 5322 simplified email regex
		emailRegex: regexp.MustCompile(`^[a-zA-Z0-9.!#$%&'*+/=?^_` + "`" + `{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$`),
		// E.164 phone number format (international)
		phoneRegex: regexp.MustCompile(`^\+?[1-9]\d{6,14}$`),
	}
}

// ValidateEmail validates email format and returns normalized email
func (v *LeadValidator) ValidateEmail(email string) (string, error) {
	email = strings.TrimSpace(strings.ToLower(email))

	if email == "" {
		return "", domain.ErrInvalidEmail
	}

	if len(email) > 255 {
		return "", domain.ErrInvalidEmail
	}

	if !v.emailRegex.MatchString(email) {
		return "", domain.ErrInvalidEmail
	}

	return email, nil
}

// ValidatePhoneNumber validates phone number format (optional field)
func (v *LeadValidator) ValidatePhoneNumber(phone *string) error {
	if phone == nil {
		return nil // Phone is optional
	}

	// Clean phone number (remove spaces, dashes, parentheses)
	cleaned := strings.TrimSpace(*phone)
	cleaned = strings.ReplaceAll(cleaned, " ", "")
	cleaned = strings.ReplaceAll(cleaned, "-", "")
	cleaned = strings.ReplaceAll(cleaned, "(", "")
	cleaned = strings.ReplaceAll(cleaned, ")", "")
	cleaned = strings.ReplaceAll(cleaned, ".", "")

	if cleaned == "" {
		return nil // Empty phone is acceptable
	}

	if !v.phoneRegex.MatchString(cleaned) {
		return domain.ErrInvalidPhoneNumber
	}

	return nil
}

// ValidateMessage validates message content
func (v *LeadValidator) ValidateMessage(message string) error {
	message = strings.TrimSpace(message)

	if len(message) < 10 {
		return domain.ErrMessageTooShort
	}

	if len(message) > 5000 {
		return domain.ErrMessageTooLong
	}

	return nil
}

// ValidateName validates the name field
func (v *LeadValidator) ValidateName(name string) error {
	name = strings.TrimSpace(name)

	if name == "" {
		return domain.ErrInvalidInput
	}

	if len(name) > 255 {
		return domain.ErrInvalidInput
	}

	return nil
}

// ValidateCreateLeadInput validates all fields of CreateLeadInput
func (v *LeadValidator) ValidateCreateLeadInput(input CreateLeadInput) error {
	// Validate name
	if err := v.ValidateName(input.Name); err != nil {
		return err
	}

	// Validate email
	if _, err := v.ValidateEmail(input.Email); err != nil {
		return err
	}

	// Validate phone (optional)
	if err := v.ValidatePhoneNumber(input.PhoneNumber); err != nil {
		return err
	}

	// Validate message
	if err := v.ValidateMessage(input.Message); err != nil {
		return err
	}

	// Validate source
	if !input.Source.IsValid() {
		return domain.ErrInvalidInput
	}

	return nil
}
