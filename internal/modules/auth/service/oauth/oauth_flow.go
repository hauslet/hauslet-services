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

func handleOAuthFlow(ctx context.Context, deps Dependencies, _ token.Claims, provider, providerUserID, email, name string, isLinking bool, linkState *LinkState) (*schema.User, error) {
	if isLinking && linkState != nil {
		if linkState.Provider != "" && linkState.Provider != provider {
			deps.Log.Error("OAuth Linking: Provider mismatch", "expected", linkState.Provider, "got", provider)
			return nil, fmt.Errorf("provider mismatch")
		}

		if deps.LinkIdentity == nil {
			deps.Log.Error("OAuth Linking: LinkIdentity function is not configured")
			return nil, fmt.Errorf("link identity not configured")
		}

		if err := deps.LinkIdentity(ctx, linkState, provider, providerUserID, email); err != nil {
			deps.Log.Error("OAuth Linking: Failed to link identity", "error", err)
			return nil, err
		}

		user, err := deps.Repository.GetUserByID(ctx, linkState.UserID)
		if err != nil || user == nil {
			deps.Log.Error("OAuth Linking: Error fetching user after linking", "error", err)
			return nil, err
		}

		deps.Log.Info("OAuth Linking: Successfully linked identity",
			"provider", provider,
			"user", maskEmail(user.PrimaryEmail),
			"user_id", user.ID)
		return user, nil
	}

	identity, err := deps.Repository.GetUserIdentityByProvider(ctx, provider, providerUserID)
	if err != nil {
		deps.Log.Error("OAuth: Error checking identity", "error", err)
		return nil, err
	}

	if identity != nil {
		user, err := deps.Repository.GetUserByID(ctx, identity.UserID.String())
		if err != nil || user == nil {
			deps.Log.Error("OAuth: Error fetching user for existing identity", "error", err)
			return nil, err
		}

		if provider == "google" && !strings.Contains(identity.Email, "@") && email != "" {
			deps.Log.Info("Fixing corrupted Google OAuth email",
				"user_id", user.ID,
				"old_email", identity.Email,
				"new_email", email)
			identity.Email = email
		}

		now := time.Now()
		identity.LastUsedAt = &now
		_ = deps.Repository.UpdateUserIdentity(ctx, identity)

		deps.Log.Info("OAuth: Existing user logged in via provider",
			"email", maskEmail(user.PrimaryEmail),
			"user_id", user.ID,
			"provider", provider)

		return ensureActiveOnLogin(ctx, deps, user)
	}

	user, err := deps.Repository.GetUserByEmail(ctx, email)
	if err != nil {
		deps.Log.Error("OAuth: Error checking user by email", "error", err)
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
			deps.Log.Error("OAuth: Error creating user+identity", "error", err)
			return nil, err
		}

		if deps.ProfileHook != nil {
			if err := deps.ProfileHook(ctx, user.ID.String(), email, user.Name, nil); err != nil {
				deps.Log.Warn("OAuth: failed to create default profile",
					"user", user.ID, "error", err)
			}
		}

		deps.Log.Info("OAuth: Created new user",
			"user", user.ID,
			"provider", provider)

		if deps.SendWelcomeEmail != nil {
			go func(email, name string) {
				if err := deps.SendWelcomeEmail(context.Background(), email, name, ""); err != nil {
					deps.Log.Warn("Failed to send OAuth welcome email", "email", email, "error", err)
				} else {
					deps.Log.Info("OAuth welcome email sent", "email", email)
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
		deps.Log.Error("OAuth: Failed to auto-link identity for user ",
			"provider", provider,
			"user", user.ID, "error", err)
		return nil, err
	}

	deps.Log.Info("OAuth: Auto-linked identity to existing user",
		"provider", provider,
		"user", user.ID)

	if deps.SendIdentityLinked != nil {
		go func(email, name, provider string) {
			if err := deps.SendIdentityLinked(context.Background(), email, name, provider); err != nil {
				deps.Log.Warn("Failed to send identity-linked notification", "email", email, "error", err)
			}
		}(user.PrimaryEmail, user.Name, provider)
	}

	return ensureActiveOnLogin(ctx, deps, user)
}
