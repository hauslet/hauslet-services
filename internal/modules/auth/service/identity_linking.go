package service

import (
	"context"
	"fmt"

	"hauslet/internal/modules/auth/repository/schema"

	"github.com/google/uuid"
)

// InitiateIdentityLinking generates OAuth URL for linking a provider to authenticated user
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

	// Generate state token
	stateToken, err := s.linkStateManager.GenerateState(userID, provider, redirectURI)
	if err != nil {
		return "", fmt.Errorf("failed to generate state: %w", err)
	}

	// Build OAuth URL with state
	oauthURL := fmt.Sprintf(
		"https://accounts.google.com/o/oauth2/v2/auth?client_id=%s&redirect_uri=%s&response_type=code&scope=profile email&state=%s",
		s.cfg.GoogleClientID,
		s.cfg.RedirectURL,
		stateToken,
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
