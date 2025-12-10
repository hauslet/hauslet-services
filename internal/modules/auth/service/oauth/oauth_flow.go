package oauth

import (
	"context"
	"fmt"
	"strings"
	"time"

	"hauslet/internal/modules/auth/domain"
	"hauslet/internal/modules/auth/repository/schema"

	"github.com/go-pkgz/auth/token"
	"github.com/google/uuid"
)

// ensureActiveOnLogin reactivates a user if allowed or records last login time.
// It returns a fresh copy of the user so callers have up-to-date activation fields.
func ensureActiveOnLogin(ctx context.Context, repo Dependencies, user *schema.User) (*schema.User, error) {
	if user == nil {
		return nil, domain.ErrUserNotFound
	}

	if user.IsActive {
		_ = repo.Repository.UpdateUserLastLogin(ctx, user.ID.String())
		return user, nil
	}

	if !user.ReactivateOnLogin {
		return nil, domain.ErrUserSuspended
	}

	if err := repo.Repository.ReactivateUser(ctx, user.ID.String(), user.ID.String(), true); err != nil {
		return nil, fmt.Errorf("failed to reactivate user: %w", err)
	}

	refreshed, err := repo.Repository.GetUserByID(ctx, user.ID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to reload user after reactivation: %w", err)
	}

	return refreshed, nil
}

func handleOAuthFlow(ctx context.Context, deps Dependencies, claims token.Claims, provider, providerUserID, email, name string, isLinking bool, linkState *LinkState) (*schema.User, error) {
	if isLinking && linkState != nil {
		if linkState.Provider != "" && linkState.Provider != provider {
			deps.Log.Logf("ERROR OAuth Linking: Provider mismatch: expected %s, got %s", linkState.Provider, provider)
			return nil, fmt.Errorf("provider mismatch")
		}

		if deps.LinkIdentity == nil {
			deps.Log.Logf("ERROR OAuth Linking: LinkIdentity function is not configured")
			return nil, fmt.Errorf("link identity not configured")
		}

		if err := deps.LinkIdentity(ctx, linkState, provider, providerUserID, email); err != nil {
			deps.Log.Logf("ERROR OAuth Linking: Failed to link identity: %v", err)
			return nil, err
		}

		user, err := deps.Repository.GetUserByID(ctx, linkState.UserID)
		if err != nil || user == nil {
			deps.Log.Logf("ERROR OAuth Linking: Error fetching user after linking: %v", err)
			return nil, err
		}

		deps.Log.Logf("INFO OAuth Linking: Successfully linked %s to user %s (ID: %s)", provider, maskEmail(user.PrimaryEmail), user.ID)
		return user, nil
	}

	identity, err := deps.Repository.GetUserIdentityByProvider(ctx, provider, providerUserID)
	if err != nil {
		deps.Log.Logf("ERROR OAuth: Error checking identity: %v", err)
		return nil, err
	}

	if identity != nil {
		user, err := deps.Repository.GetUserByID(ctx, identity.UserID.String())
		if err != nil || user == nil {
			deps.Log.Logf("ERROR OAuth: Error fetching user for existing identity: %v", err)
			return nil, err
		}

		if provider == "google" && !strings.Contains(identity.Email, "@") && email != "" {
			deps.Log.Logf("INFO Fixing corrupted Google OAuth email for user %s: %q -> %q", user.ID, identity.Email, email)
			identity.Email = email
		}

		now := time.Now()
		identity.LastUsedAt = &now
		_ = deps.Repository.UpdateUserIdentity(ctx, identity)

		deps.Log.Logf("INFO OAuth: Existing user %s (ID: %s) logged in via %s", maskEmail(user.PrimaryEmail), user.ID, provider)

		return ensureActiveOnLogin(ctx, deps, user)
	}

	user, err := deps.Repository.GetUserByEmail(ctx, email)
	if err != nil {
		deps.Log.Logf("ERROR OAuth: Error checking user by email: %v", err)
		return nil, err
	}

	if user == nil {
		user = &schema.User{
			ID:           uuid.New(),
			Name:         name,
			PrimaryEmail: email,
			Role:         schema.RoleUser,
			IsActive:     true,
		}

		identity = &schema.UserIdentity{
			ID:            uuid.New(),
			UserID:        user.ID,
			Provider:      provider,
			ProviderID:    providerUserID,
			Email:         email,
			EmailVerified: true,
		}

		if err := deps.Repository.CreateUserWithIdentity(ctx, user, identity); err != nil {
			deps.Log.Logf("ERROR OAuth: Error creating user+identity: %v", err)
			return nil, err
		}

		if deps.ProfileHook != nil {
			if err := deps.ProfileHook(ctx, user.ID.String(), user.Name, nil); err != nil {
				deps.Log.Logf("WARN OAuth: failed to create default profile for user %s: %v", user.ID, err)
			}
		}

		deps.Log.Logf("INFO OAuth: Created new user %s (ID: %s) via %s", maskEmail(user.PrimaryEmail), user.ID, provider)

		if deps.SendWelcomeEmail != nil {
			go func(email, name string) {
				if err := deps.SendWelcomeEmail(context.Background(), email, name, ""); err != nil {
					deps.Log.Logf("WARN Failed to send OAuth welcome email to %s: %v", email, err)
				} else {
					deps.Log.Logf("INFO OAuth welcome email sent to %s", email)
				}
			}(user.PrimaryEmail, user.Name)
		}

		return ensureActiveOnLogin(ctx, deps, user)
	}

	// AUTO-LINK: user exists with this email but provider not yet linked
	identity = &schema.UserIdentity{
		ID:            uuid.New(),
		UserID:        user.ID,
		Provider:      provider,
		ProviderID:    providerUserID,
		Email:         email,
		EmailVerified: true, // OAuth providers verify emails
	}

	if err := deps.Repository.CreateUserIdentity(ctx, identity); err != nil {
		deps.Log.Logf("ERROR OAuth: Failed to auto-link %s identity for user %s: %v", provider, user.ID, err)
		return nil, err
	}

	deps.Log.Logf("INFO OAuth: Auto-linked %s identity to existing user %s (ID: %s)", provider, maskEmail(user.PrimaryEmail), user.ID)

	if deps.SendIdentityLinked != nil {
		go func(email, name, provider string) {
			if err := deps.SendIdentityLinked(context.Background(), email, name, provider); err != nil {
				deps.Log.Logf("WARN Failed to send identity-linked notification to %s: %v", email, err)
			}
		}(user.PrimaryEmail, user.Name, provider)
	}

	return ensureActiveOnLogin(ctx, deps, user)
}
