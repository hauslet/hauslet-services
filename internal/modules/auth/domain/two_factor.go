package domain

import (
	"time"

	"github.com/google/uuid"
)

// TwoFactorMethod represents the available 2FA verification methods
type TwoFactorMethod string

const (
	TwoFactorEmail         TwoFactorMethod = "email"
	TwoFactorSMS           TwoFactorMethod = "sms"
	TwoFactorAuthenticator TwoFactorMethod = "authenticator"
)

// User2FA represents the 2FA settings for a user
type User2FA struct {
	ID                   uuid.UUID
	UserID               uuid.UUID
	Method               TwoFactorMethod
	PhoneNumber          *string // For SMS method
	TOTPSecretEncrypted  *string // For Authenticator method
	IsEnabled            bool
	EnabledAt            *time.Time
	BackupCodesHash      []string
	BackupCodesRemaining int
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// TwoFactorStatus is returned to clients to show 2FA state
type TwoFactorStatus struct {
	Enabled         bool            `json:"enabled"`
	Method          TwoFactorMethod `json:"method,omitempty"`
	BackupCodesLeft int             `json:"backup_codes_left"`
}

// SetupResponse is returned when initiating 2FA setup
type SetupResponse struct {
	Method      TwoFactorMethod `json:"method"`
	QRCodeURL   string          `json:"qr_code_url,omitempty"`   // otpauth:// URL (for authenticator)
	QRCodeImage string          `json:"qr_code_image,omitempty"` // Base64 data URL for QR code image
	Secret      string          `json:"secret,omitempty"`        // Only for authenticator (shown once)
}

// BackupCodesResult is returned after 2FA is enabled or codes regenerated
type BackupCodesResult struct {
	Codes []string `json:"codes"` // Plain text codes (shown once)
}

// AuthResultWith2FA extends auth result when 2FA is required
type AuthResultWith2FA struct {
	Requires2FA bool            `json:"requires_2fa"`
	Method      TwoFactorMethod `json:"method,omitempty"`
	TempToken   string          `json:"temp_token,omitempty"` // Short-lived token for 2FA verification
}

// Pending2FAState is stored in Redis while waiting for 2FA verification
type Pending2FAState struct {
	UserID    string          `json:"user_id"`
	Email     string          `json:"email"`
	Name      string          `json:"name"`
	Method    TwoFactorMethod `json:"method"`
	Provider  string          `json:"provider"` // "password", "email", etc.
	CreatedAt time.Time       `json:"created_at"`
}

// Verify2FALoginRequest is the request body for completing 2FA login
type Verify2FALoginRequest struct {
	TempToken string `json:"temp_token"`
	Code      string `json:"code"`
}
