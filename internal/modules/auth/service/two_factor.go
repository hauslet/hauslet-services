package service

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"hauslet/internal/modules/auth/domain"
	"hauslet/internal/modules/auth/repository/schema"

	"github.com/google/uuid"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"
)

const (
	twoFACodeRedisPrefix    = "2fa_code:"
	twoFASetupRedisPrefix   = "2fa_setup:"
	twoFARateLimitPrefix    = "2fa_ratelimit:"
	twoFAPendingPrefix      = "2fa_pending:"
	twoFACodeTTL            = 5 * time.Minute
	twoFASetupTTL           = 10 * time.Minute
	twoFARateLimitTTL       = 15 * time.Minute
	twoFAPendingTTL         = 5 * time.Minute
	backupCodeCount         = 8
	maxVerificationAttempts = 5
)

// twoFASetupData is the JSON structure stored in Redis during 2FA setup
type twoFASetupData struct {
	Method      string `json:"method"`
	OTPCode     string `json:"otp_code,omitempty"`
	PhoneNumber string `json:"phone_number,omitempty"`
	TOTPSecret  string `json:"totp_secret,omitempty"`
}

// ============================================================================
// AES-GCM Encryption Helpers
// ============================================================================

// encryptTOTPSecret encrypts the TOTP secret using AES-GCM
func (s *AuthServiceImpl) encryptTOTPSecret(plaintext string) (string, error) {
	block, err := aes.NewCipher(s.cfg.EncryptAuthCodeKey)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := aesGCM.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decryptTOTPSecret decrypts the TOTP secret using AES-GCM
func (s *AuthServiceImpl) decryptTOTPSecret(ciphertext string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("failed to decode ciphertext: %w", err)
	}

	block, err := aes.NewCipher(s.cfg.EncryptAuthCodeKey)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := aesGCM.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]
	plaintext, err := aesGCM.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}

// ============================================================================
// Rate Limiting Helpers
// ============================================================================

// check2FARateLimit checks if the user is rate limited for 2FA verification
// Returns an error if rate limited, nil otherwise
func (s *AuthServiceImpl) check2FARateLimit(ctx context.Context, userID string) error {
	key := twoFARateLimitPrefix + userID
	attempts, err := s.redisClient.Get(ctx, key).Int()
	if err != nil && err.Error() != "redis: nil" {
		// Key doesn't exist, no rate limiting
		return nil
	}

	if attempts >= maxVerificationAttempts {
		return errors.New("too many failed verification attempts, please try again in 15 minutes")
	}

	return nil
}

// increment2FAAttempts increments the failed attempt counter
func (s *AuthServiceImpl) increment2FAAttempts(ctx context.Context, userID string) {
	key := twoFARateLimitPrefix + userID
	s.redisClient.Incr(ctx, key)
	s.redisClient.Expire(ctx, key, twoFARateLimitTTL)
}

// reset2FAAttempts clears the failed attempt counter on successful verification
func (s *AuthServiceImpl) reset2FAAttempts(ctx context.Context, userID string) {
	key := twoFARateLimitPrefix + userID
	_ = s.redisClient.Del(ctx, key)
}

// ============================================================================
// Two-Factor Authentication Setup
// ============================================================================

// InitiateSetup2FA starts the 2FA setup process based on the selected method
func (s *AuthServiceImpl) InitiateSetup2FA(ctx context.Context, userID string, method domain.TwoFactorMethod, phoneNumber string) (*domain.SetupResponse, error) {
	// Get user
	user, err := s.GetUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Check if 2FA is already enabled
	existing, err := s.repository.GetUser2FA(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing 2FA: %w", err)
	}
	if existing != nil && existing.IsEnabled {
		return nil, errors.New("2FA is already enabled, disable it first to change method")
	}

	switch method {
	case domain.TwoFactorEmail:
		return s.initiateEmailSetup(ctx, user)
	case domain.TwoFactorSMS:
		if phoneNumber == "" {
			return nil, errors.New("phone number is required for SMS method")
		}
		return s.initiateSMSSetup(ctx, user, phoneNumber)
	case domain.TwoFactorAuthenticator:
		return s.initiateAuthenticatorSetup(ctx, user)
	default:
		return nil, errors.New("invalid 2FA method")
	}
}

func (s *AuthServiceImpl) initiateEmailSetup(ctx context.Context, user *domain.User) (*domain.SetupResponse, error) {
	otpCode, err := generateSecureOTP()
	if err != nil {
		return nil, fmt.Errorf("failed to generate OTP: %w", err)
	}

	// Store setup data as JSON in Redis
	setupData := twoFASetupData{
		Method:  string(domain.TwoFactorEmail),
		OTPCode: otpCode,
	}
	jsonData, err := json.Marshal(setupData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal setup data: %w", err)
	}

	key := twoFASetupRedisPrefix + user.ID.String()
	if err := s.redisClient.Set(ctx, key, string(jsonData), twoFASetupTTL).Err(); err != nil {
		return nil, fmt.Errorf("failed to store setup OTP: %w", err)
	}

	// Send OTP via email
	if err := s.notifier.Send2FASetupEmail(ctx, user.PrimaryEmail, user.Name, otpCode); err != nil {
		return nil, fmt.Errorf("failed to send 2FA setup email: %w", err)
	}

	return &domain.SetupResponse{
		Method: domain.TwoFactorEmail,
	}, nil
}

func (s *AuthServiceImpl) initiateSMSSetup(ctx context.Context, user *domain.User, phoneNumber string) (*domain.SetupResponse, error) {
	otpCode, err := generateSecureOTP()
	if err != nil {
		return nil, fmt.Errorf("failed to generate OTP: %w", err)
	}

	// Store setup data as JSON in Redis (robust against special characters in phone numbers)
	setupData := twoFASetupData{
		Method:      string(domain.TwoFactorSMS),
		OTPCode:     otpCode,
		PhoneNumber: phoneNumber,
	}
	jsonData, err := json.Marshal(setupData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal setup data: %w", err)
	}

	key := twoFASetupRedisPrefix + user.ID.String()
	if err := s.redisClient.Set(ctx, key, string(jsonData), twoFASetupTTL).Err(); err != nil {
		return nil, fmt.Errorf("failed to store setup OTP: %w", err)
	}

	// Send OTP via SMS
	if err := s.notifier.Send2FASetupSMS(ctx, phoneNumber, otpCode); err != nil {
		return nil, fmt.Errorf("failed to send 2FA setup SMS: %w", err)
	}

	return &domain.SetupResponse{
		Method: domain.TwoFactorSMS,
	}, nil
}

func (s *AuthServiceImpl) initiateAuthenticatorSetup(ctx context.Context, user *domain.User) (*domain.SetupResponse, error) {
	// Generate TOTP secret
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "Hauslet",
		AccountName: user.PrimaryEmail,
		SecretSize:  32,
		Algorithm:   otp.AlgorithmSHA1,
		Digits:      otp.DigitsSix,
		Period:      30,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate TOTP secret: %w", err)
	}

	// Store secret in Redis temporarily as JSON until setup is completed
	setupData := twoFASetupData{
		Method:     string(domain.TwoFactorAuthenticator),
		TOTPSecret: key.Secret(),
	}
	jsonData, err := json.Marshal(setupData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal setup data: %w", err)
	}

	redisKey := twoFASetupRedisPrefix + user.ID.String()
	if err := s.redisClient.Set(ctx, redisKey, string(jsonData), twoFASetupTTL).Err(); err != nil {
		return nil, fmt.Errorf("failed to store TOTP secret: %w", err)
	}

	return &domain.SetupResponse{
		Method:    domain.TwoFactorAuthenticator,
		QRCodeURL: key.URL(),
		Secret:    key.Secret(),
	}, nil
}

// CompleteSetup2FA verifies the code and enables 2FA
func (s *AuthServiceImpl) CompleteSetup2FA(ctx context.Context, userID, code string) (*domain.BackupCodesResult, error) {
	// Get pending setup from Redis
	redisKey := twoFASetupRedisPrefix + userID
	value, err := s.redisClient.Get(ctx, redisKey).Result()
	if err != nil {
		return nil, errors.New("no pending 2FA setup found, please initiate setup first")
	}

	// Parse JSON setup data
	var setupData twoFASetupData
	if err := json.Unmarshal([]byte(value), &setupData); err != nil {
		return nil, errors.New("invalid setup state")
	}

	method := domain.TwoFactorMethod(setupData.Method)
	var twoFA *schema.User2FA

	switch method {
	case domain.TwoFactorEmail:
		if code != setupData.OTPCode {
			return nil, errors.New("invalid verification code")
		}
		twoFA = &schema.User2FA{
			UserID: uuid.MustParse(userID),
			Method: schema.TwoFactorEmail,
		}

	case domain.TwoFactorSMS:
		if code != setupData.OTPCode {
			return nil, errors.New("invalid verification code")
		}
		twoFA = &schema.User2FA{
			UserID:      uuid.MustParse(userID),
			Method:      schema.TwoFactorSMS,
			PhoneNumber: &setupData.PhoneNumber,
		}

	case domain.TwoFactorAuthenticator:
		// Validate TOTP code with skew to allow for clock drift
		valid, err := totp.ValidateCustom(code, setupData.TOTPSecret, time.Now(), totp.ValidateOpts{
			Period:    30,
			Skew:      1, // Allows +/- 30 seconds
			Digits:    otp.DigitsSix,
			Algorithm: otp.AlgorithmSHA1,
		})
		if err != nil || !valid {
			return nil, errors.New("invalid authenticator code")
		}

		// Encrypt the TOTP secret before storing
		encryptedSecret, err := s.encryptTOTPSecret(setupData.TOTPSecret)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt TOTP secret: %w", err)
		}

		twoFA = &schema.User2FA{
			UserID:              uuid.MustParse(userID),
			Method:              schema.TwoFactorAuthenticator,
			TOTPSecretEncrypted: &encryptedSecret,
		}

	default:
		return nil, errors.New("invalid 2FA method")
	}

	// Generate backup codes
	codes, hashes, err := generateBackupCodes(backupCodeCount)
	if err != nil {
		return nil, fmt.Errorf("failed to generate backup codes: %w", err)
	}
	now := time.Now()

	twoFA.IsEnabled = true
	twoFA.EnabledAt = &now
	twoFA.BackupCodesHash = hashes
	twoFA.BackupCodesRemaining = backupCodeCount

	// Check if updating existing or creating new
	existing, _ := s.repository.GetUser2FA(ctx, userID)
	if existing != nil {
		twoFA.ID = existing.ID
		if err := s.repository.UpdateUser2FA(ctx, twoFA); err != nil {
			return nil, fmt.Errorf("failed to update 2FA settings: %w", err)
		}
	} else {
		if err := s.repository.CreateUser2FA(ctx, twoFA); err != nil {
			return nil, fmt.Errorf("failed to enable 2FA: %w", err)
		}
	}

	// Delete setup data from Redis
	_ = s.redisClient.Del(ctx, redisKey)

	s.log.Info("2FA enabled", "user_id", userID, "method", method)

	return &domain.BackupCodesResult{
		Codes: codes,
	}, nil
}

// ============================================================================
// Two-Factor Authentication Verification
// ============================================================================

// Send2FACode sends a 2FA code for Email/SMS methods (called during login)
func (s *AuthServiceImpl) Send2FACode(ctx context.Context, userID string) error {
	twoFA, err := s.repository.GetUser2FA(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get 2FA settings: %w", err)
	}
	if twoFA == nil || !twoFA.IsEnabled {
		return errors.New("2FA is not enabled")
	}

	// Only Email and SMS need code sending
	if twoFA.Method == schema.TwoFactorAuthenticator {
		return nil // Authenticator doesn't need server-side code
	}

	user, err := s.GetUser(ctx, userID)
	if err != nil {
		return err
	}

	code, err := generateSecureOTP()
	if err != nil {
		return fmt.Errorf("failed to generate OTP: %w", err)
	}
	redisKey := twoFACodeRedisPrefix + userID

	if err := s.redisClient.Set(ctx, redisKey, code, twoFACodeTTL).Err(); err != nil {
		return fmt.Errorf("failed to store 2FA code: %w", err)
	}

	switch twoFA.Method {
	case schema.TwoFactorEmail:
		return s.notifier.Send2FAVerificationEmail(ctx, user.PrimaryEmail, user.Name, code)
	case schema.TwoFactorSMS:
		if twoFA.PhoneNumber == nil {
			return errors.New("phone number not configured")
		}
		// Send via SMS client
		s.log.Info("2FA SMS code", "phone", *twoFA.PhoneNumber, "code", code)
		return nil
	}

	return nil
}

// Verify2FACode verifies a 2FA code during login
func (s *AuthServiceImpl) Verify2FACode(ctx context.Context, userID, code string) error {
	// Check rate limiting first
	if err := s.check2FARateLimit(ctx, userID); err != nil {
		return err
	}

	twoFA, err := s.repository.GetUser2FA(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get 2FA settings: %w", err)
	}
	if twoFA == nil || !twoFA.IsEnabled {
		return errors.New("2FA is not enabled")
	}

	switch twoFA.Method {
	case schema.TwoFactorAuthenticator:
		if twoFA.TOTPSecretEncrypted == nil {
			return errors.New("authenticator not configured")
		}

		// Decrypt secret before validation
		secret, err := s.decryptTOTPSecret(*twoFA.TOTPSecretEncrypted)
		if err != nil {
			s.log.Error("failed to decrypt TOTP secret", "error", err)
			return errors.New("authenticator configuration error")
		}

		// Validate with skew to allow for clock drift (+/- 30 seconds)
		valid, err := totp.ValidateCustom(code, secret, time.Now(), totp.ValidateOpts{
			Period:    30,
			Skew:      1, // Allows previous or next 30-second window
			Digits:    otp.DigitsSix,
			Algorithm: otp.AlgorithmSHA1,
		})
		if err != nil || !valid {
			s.increment2FAAttempts(ctx, userID)
			return errors.New("invalid authenticator code")
		}

		s.reset2FAAttempts(ctx, userID)
		return nil

	case schema.TwoFactorEmail, schema.TwoFactorSMS:
		redisKey := twoFACodeRedisPrefix + userID
		storedCode, err := s.redisClient.Get(ctx, redisKey).Result()
		if err != nil {
			return errors.New("verification code expired, please request a new one")
		}
		if code != storedCode {
			s.increment2FAAttempts(ctx, userID)
			return errors.New("invalid verification code")
		}
		// Delete code after successful verification
		_ = s.redisClient.Del(ctx, redisKey)
		s.reset2FAAttempts(ctx, userID)
		return nil
	}

	return errors.New("unknown 2FA method")
}

// Verify2FABackupCode verifies a backup code using atomic database operations
func (s *AuthServiceImpl) Verify2FABackupCode(ctx context.Context, userID, code string) error {
	// Check rate limiting first
	if err := s.check2FARateLimit(ctx, userID); err != nil {
		return err
	}

	twoFA, err := s.repository.GetUser2FA(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get 2FA settings: %w", err)
	}
	if twoFA == nil || !twoFA.IsEnabled {
		return errors.New("2FA is not enabled")
	}

	// Find the matching backup code hash
	var matchedHash string
	for _, hash := range twoFA.BackupCodesHash {
		if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(code)); err == nil {
			matchedHash = hash
			break
		}
	}

	if matchedHash == "" {
		s.increment2FAAttempts(ctx, userID)
		return errors.New("invalid backup code")
	}

	// Use atomic operation to consume the backup code
	if err := s.repository.ConsumeBackupCode(ctx, userID, matchedHash); err != nil {
		// If atomic operation fails (e.g., code already used), return error
		s.increment2FAAttempts(ctx, userID)
		return errors.New("backup code already used or invalid")
	}

	s.reset2FAAttempts(ctx, userID)
	s.log.Info("backup code used", "user_id", userID)

	return nil
}

// ============================================================================
// Two-Factor Authentication Management
// ============================================================================

// Disable2FA disables 2FA after verifying the current code
func (s *AuthServiceImpl) Disable2FA(ctx context.Context, userID, code string) error {
	// Verify code first
	if err := s.Verify2FACode(ctx, userID, code); err != nil {
		// Try backup code if regular code fails
		if bkErr := s.Verify2FABackupCode(ctx, userID, code); bkErr != nil {
			return errors.New("invalid verification code")
		}
	}

	if err := s.repository.DeleteUser2FA(ctx, userID); err != nil {
		return fmt.Errorf("failed to disable 2FA: %w", err)
	}

	s.log.Info("2FA disabled", "user_id", userID)
	return nil
}

// RegenerateBackupCodes generates new backup codes after verifying current code
func (s *AuthServiceImpl) RegenerateBackupCodes(ctx context.Context, userID, code string) (*domain.BackupCodesResult, error) {
	// Verify code first
	if err := s.Verify2FACode(ctx, userID, code); err != nil {
		return nil, err
	}

	twoFA, err := s.repository.GetUser2FA(ctx, userID)
	if err != nil || twoFA == nil {
		return nil, errors.New("2FA is not enabled")
	}

	// Generate new backup codes
	codes, hashes, err := generateBackupCodes(backupCodeCount)
	if err != nil {
		return nil, fmt.Errorf("failed to generate backup codes: %w", err)
	}
	twoFA.BackupCodesHash = hashes
	twoFA.BackupCodesRemaining = backupCodeCount

	if err := s.repository.UpdateUser2FA(ctx, twoFA); err != nil {
		return nil, fmt.Errorf("failed to update backup codes: %w", err)
	}

	return &domain.BackupCodesResult{
		Codes: codes,
	}, nil
}

// Get2FAStatus returns the current 2FA status for a user
func (s *AuthServiceImpl) Get2FAStatus(ctx context.Context, userID string) (*domain.TwoFactorStatus, error) {
	twoFA, err := s.repository.GetUser2FA(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get 2FA status: %w", err)
	}

	if twoFA == nil {
		return &domain.TwoFactorStatus{
			Enabled: false,
		}, nil
	}

	return &domain.TwoFactorStatus{
		Enabled:         twoFA.IsEnabled,
		Method:          domain.TwoFactorMethod(twoFA.Method),
		BackupCodesLeft: twoFA.BackupCodesRemaining,
	}, nil
}

// ============================================================================
// Two-Factor Authentication Login Flow
// ============================================================================

// Create2FAPendingState generates a temporary token and stores pending 2FA state in Redis
func (s *AuthServiceImpl) Create2FAPendingState(ctx context.Context, userID, email, name, provider string, method domain.TwoFactorMethod) (string, error) {
	// Generate a cryptographically secure random token
	tokenBytes := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, tokenBytes); err != nil {
		return "", fmt.Errorf("failed to generate temp token: %w", err)
	}
	tempToken := base64.URLEncoding.EncodeToString(tokenBytes)

	// Create pending state
	state := domain.Pending2FAState{
		UserID:    userID,
		Email:     email,
		Name:      name,
		Method:    method,
		Provider:  provider,
		CreatedAt: time.Now(),
	}

	// Serialize to JSON
	jsonData, err := json.Marshal(state)
	if err != nil {
		return "", fmt.Errorf("failed to marshal pending state: %w", err)
	}

	// Store in Redis
	key := twoFAPendingPrefix + tempToken
	if err := s.redisClient.Set(ctx, key, string(jsonData), twoFAPendingTTL).Err(); err != nil {
		return "", fmt.Errorf("failed to store pending state: %w", err)
	}

	s.log.Info("2FA pending state created", "user_id", userID, "method", method)
	return tempToken, nil
}

// Get2FAPendingState retrieves pending 2FA state from Redis
func (s *AuthServiceImpl) Get2FAPendingState(ctx context.Context, tempToken string) (*domain.Pending2FAState, error) {
	if tempToken == "" {
		return nil, errors.New("temp token is required")
	}

	key := twoFAPendingPrefix + tempToken
	value, err := s.redisClient.Get(ctx, key).Result()
	if err != nil {
		return nil, errors.New("invalid or expired temp token")
	}

	var state domain.Pending2FAState
	if err := json.Unmarshal([]byte(value), &state); err != nil {
		return nil, errors.New("invalid pending state")
	}

	return &state, nil
}

// Delete2FAPendingState removes pending 2FA state from Redis
func (s *AuthServiceImpl) Delete2FAPendingState(ctx context.Context, tempToken string) error {
	key := twoFAPendingPrefix + tempToken
	return s.redisClient.Del(ctx, key).Err()
}

// Verify2FALogin verifies the 2FA code and returns the user if successful
func (s *AuthServiceImpl) Verify2FALogin(ctx context.Context, tempToken, code string) (*domain.User, error) {
	// Get pending state
	state, err := s.Get2FAPendingState(ctx, tempToken)
	if err != nil {
		return nil, err
	}

	// Verify the 2FA code
	if err := s.Verify2FACode(ctx, state.UserID, code); err != nil {
		// Try backup code if regular code fails
		if bkErr := s.Verify2FABackupCode(ctx, state.UserID, code); bkErr != nil {
			return nil, errors.New("invalid verification code")
		}
	}

	// Delete pending state
	_ = s.Delete2FAPendingState(ctx, tempToken)

	// Get the user
	user, err := s.GetUser(ctx, state.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Update last login
	_ = s.repository.UpdateUserLastLogin(ctx, state.UserID)

	s.log.Info("2FA login verified", "user_id", state.UserID)
	return user, nil
}
