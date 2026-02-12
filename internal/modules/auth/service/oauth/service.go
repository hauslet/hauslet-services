package oauth

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"hauslet/config"
	"hauslet/internal/modules/auth/domain"
	"hauslet/internal/modules/auth/repository"

	"github.com/go-pkgz/auth/v2"
	"github.com/go-pkgz/auth/v2/avatar"
	"github.com/go-pkgz/auth/v2/token"
)

// slogAdapter wraps *slog.Logger to implement go-pkgz/auth/logger.L interface.
type slogAdapter struct {
	logger *slog.Logger
}

// Logf implements the logger.L interface required by go-pkgz/auth.
func (a *slogAdapter) Logf(format string, args ...any) {
	a.logger.Info(fmt.Sprintf(format, args...))
}

// newSlogAdapter creates a logger adapter from *slog.Logger.
func newSlogAdapter(logger *slog.Logger) *slogAdapter {
	return &slogAdapter{logger: logger}
}

// RequestMetadata carries optional request context for session enrichment.
type RequestMetadata struct {
	IP        string
	UserAgent string
}

// MetadataFetcher retrieves request metadata for a given key (usually email).
type MetadataFetcher func(key string) *RequestMetadata

// LinkState describes minimal data needed during OAuth linking validation.
type LinkState struct {
	UserID      string
	Provider    string
	RedirectURI string
}

// LinkStateValidator validates a signed link state token and returns decoded data.
type LinkStateValidator func(token string) (*LinkState, error)

// PasswordAuthenticator verifies a password credential.
type PasswordAuthenticator func(ctx context.Context, email, password string) (*domain.User, error)

// LinkIdentityFunc links an OAuth identity to an existing account.
type LinkIdentityFunc func(ctx context.Context, state *LinkState, provider, providerUserID, email string) error

// SendWelcomeEmailFunc sends a welcome email for newly created OAuth users.
type SendWelcomeEmailFunc func(ctx context.Context, email, name, otpCode string) error

// SendIdentityLinkedEmailFunc sends a security notification when a new identity is linked.
type SendIdentityLinkedEmailFunc func(ctx context.Context, email, name, provider string) error

// ProfileAvatarFetcher fetches a fully-qualified avatar URL for the user.
type ProfileAvatarFetcher func(ctx context.Context, userID string) (*string, error)

// ProfileHookFunc creates a default profile for new users.
type ProfileHookFunc func(ctx context.Context, userID, email, name string, birthDate *time.Time) error

// SendPasswordlessEmailFunc sends a passwordless login email with both a 6-digit code and magic link.
type SendPasswordlessEmailFunc func(ctx context.Context, email, otpCode, magicLink string, ttlMinutes int) error

// GenerateAndStoreOTPFunc generates a 6-digit OTP and stores it for later verification.
type GenerateAndStoreOTPFunc func(ctx context.Context, email string) (string, error)

// TwoFactorMethod represents 2FA method type
type TwoFactorMethod string

// GetUser2FAFunc retrieves 2FA settings for a user
type GetUser2FAFunc func(ctx context.Context, userID string) (enabled bool, method TwoFactorMethod, err error)

// Create2FAPendingStateFunc creates a pending 2FA state and returns temp token
type Create2FAPendingStateFunc func(ctx context.Context, userID, email, name, provider string, method TwoFactorMethod) (tempToken string, err error)

// Send2FACodeFunc sends a 2FA verification code to the user
type Send2FACodeFunc func(ctx context.Context, userID string) error

// Pending2FAResult is returned when 2FA verification is required
type Pending2FAResult struct {
	Required  bool
	TempToken string
	Method    TwoFactorMethod
}

// GetPendingLinkFunc retrieves a pending link state by target email
type GetPendingLinkFunc func(email string) *LinkState

// Dependencies groups the collaborators required by the OAuth service helpers.
type Dependencies struct {
	Config                *config.AuthConfig
	Repository            repository.AuthRepository
	Log                   *slog.Logger
	MetadataFetcher       MetadataFetcher
	LinkStateValidator    LinkStateValidator
	GetPendingLink        GetPendingLinkFunc
	AuthenticatePassword  PasswordAuthenticator
	LinkIdentity          LinkIdentityFunc
	SendWelcomeEmail      SendWelcomeEmailFunc
	SendIdentityLinked    SendIdentityLinkedEmailFunc
	SendPasswordlessEmail SendPasswordlessEmailFunc
	GenerateAndStoreOTP   GenerateAndStoreOTPFunc
	ProfileHook           ProfileHookFunc
	ProfileAvatarFetcher  ProfileAvatarFetcher
	// 2FA login flow dependencies
	GetUser2FA            GetUser2FAFunc
	Create2FAPendingState Create2FAPendingStateFunc
	Send2FACode           Send2FACodeFunc
}

// NewService builds the OAuth service with configured providers and validator.
func NewService(deps Dependencies) *auth.Service {
	claims := newClaimsEnricher(deps)

	sameSite := http.SameSiteLaxMode
	secureCookie := false

	if deps.Config.Env == "development" {
		sameSite = http.SameSiteLaxMode
		secureCookie = false
	} else if deps.Config.CookieDomain == "" {
		sameSite = http.SameSiteNoneMode
		secureCookie = true
	}

	options := auth.Opts{
		Logger: newSlogAdapter(deps.Log),
		SecretReader: token.SecretFunc(func(id string) (string, error) {
			return deps.Config.JWTSecret, nil
		}),
		ClaimsUpd: token.ClaimsUpdFunc(claims.EnrichClaims),

		TokenDuration:     deps.Config.TokenDuration,
		CookieDuration:    deps.Config.CookieDuration,
		Issuer:            "Hauslet",
		URL:               deps.Config.ServerURL,
		AvatarStore:       avatar.NewLocalFS(deps.Config.AvatarStorePath),
		SendJWTHeader:     false,
		DisableXSRF:       deps.Config.DisableXSRF,
		SameSiteCookie:    sameSite,
		SecureCookies:     secureCookie,
		XSRFIgnoreMethods: []string{"GET"},
		Validator:         NewValidator(deps.Repository, deps.Log),
		JWTCookieDomain:   deps.Config.CookieDomain,
	}

	service := auth.NewService(options)

	setupGoogleProvider(service, deps)
	setupDirectProvider(service, deps)
	setupEmailProvider(service, deps)

	if deps.Config.DisableXSRF {
		deps.Log.Warn("XSRF protection is DISABLED - only use in development")
	} else {
		deps.Log.Info(" XSRF protection is ENABLED")
	}

	return service
}
