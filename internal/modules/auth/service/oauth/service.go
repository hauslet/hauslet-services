package oauth

import (
	"context"
	"time"

	"hauslet/config"
	"hauslet/internal/modules/auth/domain"
	"hauslet/internal/modules/auth/repository"

	"github.com/go-pkgz/auth"
	"github.com/go-pkgz/auth/avatar"
	"github.com/go-pkgz/auth/token"
	"github.com/go-pkgz/lgr"
)

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
type ProfileHookFunc func(ctx context.Context, userID string, name string, birthDate *time.Time) error

// Dependencies groups the collaborators required by the OAuth service helpers.
type Dependencies struct {
	Config               *config.AuthConfig
	Repository           repository.AuthRepository
	Log                  *lgr.Logger
	MetadataFetcher      MetadataFetcher
	LinkStateValidator   LinkStateValidator
	AuthenticatePassword PasswordAuthenticator
	LinkIdentity         LinkIdentityFunc
	SendWelcomeEmail     SendWelcomeEmailFunc
	SendIdentityLinked   SendIdentityLinkedEmailFunc
	ProfileHook          ProfileHookFunc
	ProfileAvatarFetcher ProfileAvatarFetcher
}

// NewService builds the OAuth service with configured providers and validator.
func NewService(deps Dependencies) *auth.Service {
	claims := newClaimsEnricher(deps)

	options := auth.Opts{
		Logger: deps.Log,
		SecretReader: token.SecretFunc(func(id string) (string, error) {
			return deps.Config.JWTSecret, nil
		}),
		ClaimsUpd: token.ClaimsUpdFunc(claims.EnrichClaims),

		TokenDuration:     deps.Config.TokenDuration,
		CookieDuration:    deps.Config.CookieDuration,
		Issuer:            "Hauslet",
		URL:               deps.Config.RedirectURL,
		AvatarStore:       avatar.NewLocalFS(deps.Config.AvatarStorePath),
		SendJWTHeader:     false,
		DisableXSRF:       deps.Config.DisableXSRF,
		XSRFIgnoreMethods: []string{"GET"},
		Validator:         NewValidator(deps.Repository, deps.Log),
		JWTCookieDomain:   deps.Config.CookieDomain,
	}

	service := auth.NewService(options)

	setupGoogleProvider(service, deps)
	setupDirectProvider(service, deps)

	if deps.Config.DisableXSRF {
		deps.Log.Logf("WARN XSRF protection is DISABLED - only use in development")
	} else {
		deps.Log.Logf("INFO XSRF protection is ENABLED")
	}

	return service
}
