package service

import (
	"context"
	"fmt"
	"net/url"

	"hauslet/internal/modules/auth/repository/schema"

	"time"

	"github.com/google/uuid"
)

// InitiateIdentityLinking generates OAuth URL for linking a provider to authenticated user.
// We redirect through go-pkgz/auth's login endpoint with the link state embedded in the `from` parameter.
// This ensures compatibility with go-pkgz/auth's handshake token flow.
func (s *AuthServiceImpl) InitiateIdentityLinking(userID, provider, redirectURI string) (string, error) {
	// Validate provider
	if provider != "google" {
		return "", fmt.Errorf("unsupported provider: %s", provider)
	}

	// Validate user exists
	ctx := context.Background()
	user, err := s.repository.GetUserByID(ctx, userID)
	if err != nil || user == nil {
		return "", fmt.Errorf("user not found")
	}

	// Generate state token (e.g., "link.{base64_payload}.{signature}")
	stateToken, err := s.linkStateManager.GenerateState(userID, provider, redirectURI)
	if err != nil {
		return "", fmt.Errorf("failed to generate state: %w", err)
	}

	// Register pending link so we can detect linking during OAuth callback
	// The claims enricher will check for pending links by email
	linkState := &LinkState{
		UserID:      userID,
		Provider:    provider,
		RedirectURI: redirectURI,
		ExpiresAt:   time.Now().Add(10 * time.Minute),
	}
	if err := s.linkStateManager.RegisterPendingLink(ctx, user.PrimaryEmail, linkState); err != nil {
		return "", fmt.Errorf("failed to register pending link: %w", err)
	}
	s.log.Info("Registered pending link", "user_id", userID, "email", user.PrimaryEmail, "provider", provider)

	// Embed the link state in the `from` parameter
	// go-pkgz/auth preserves this through the handshake and stores it in claims.Handshake.From
	// We'll detect linking in the claims enricher by checking for "link_state=" prefix in From
	fromURL := fmt.Sprintf("%s?link_state=%s", redirectURI, url.QueryEscape(stateToken))

	// Build OAuth URL using go-pkgz/auth's login endpoint (for proper handshake token handling)
	oauthURL := fmt.Sprintf(
		"%s/auth/google/login?from=%s",
		s.cfg.ServerURL,
		url.QueryEscape(fromURL),
	)

	return oauthURL, nil
}

// linkIdentityToUser links an OAuth identity to an existing authenticated user
func (s *AuthServiceImpl) linkIdentityToUser(ctx context.Context, linkState *LinkState, provider, providerUserID, email string) error {
	// Get the user to link to
	user, err := s.repository.GetUserByID(ctx, linkState.UserID)
	if err != nil || user == nil {
		return fmt.Errorf("user not found: %s", linkState.UserID)
	}

	// Check if this provider identity already exists
	existingIdentity, err := s.repository.GetUserIdentityByProvider(ctx, provider, providerUserID)
	if err != nil {
		return fmt.Errorf("failed to check existing identity: %w", err)
	}

	// Reject if provider already linked to a different user
	if existingIdentity != nil && existingIdentity.UserID.String() != linkState.UserID {
		return fmt.Errorf("provider already linked to another account")
	}

	// If already linked to same user, idempotent success
	if existingIdentity != nil && existingIdentity.UserID.String() == linkState.UserID {
		s.log.Info("Identity Linking: Provider already linked to user", "provider", provider, "user_id", user.ID)
		return nil
	}

	// Create new identity
	identity := &schema.UserIdentity{
		ID:            uuid.New(),
		UserID:        user.ID,
		Provider:      provider,
		ProviderID:    providerUserID,
		Email:         email,
		EmailVerified: true, // OAuth providers verify emails
	}

	if err := s.repository.CreateUserIdentity(ctx, identity); err != nil {
		return fmt.Errorf("failed to create identity: %w", err)
	}

	s.log.Info("Identity Linking: Linked identity", "provider", provider, "user_id", user.ID)

	// Send security notification email
	if err := s.notifier.SendIdentityLinkedEmail(context.Background(), user.PrimaryEmail, user.Name, provider); err != nil {
		s.log.Warn("Failed to send identity linked notification", "user_email", user.PrimaryEmail, "error", err)
	}

	return nil
}
