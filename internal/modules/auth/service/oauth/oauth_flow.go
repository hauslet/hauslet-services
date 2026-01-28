package oauth

import (
	"context"
	"fmt"
	"time"

	"hauslet/internal/modules/auth/domain"
	"hauslet/internal/modules/auth/repository/schema"

	"github.com/go-pkgz/auth/v2/token"
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

// handleOAuthFlow processes OAuth provider authentication (Google, etc.).
// It either links the identity to an existing account (if linking) or finds/creates
// a user account for the OAuth identity, sending welcome emails to new users.
func handleOAuthFlow(ctx context.Context,
	deps Dependencies, claims token.Claims,
	provider, providerUserID,
	email, name string,
	isLinking bool,
	linkState *LinkState,
) (*schema.User, error) {
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

		now := time.Now()
		identity.LastUsedAt = &now
		_ = deps.Repository.UpdateUserIdentity(ctx, identity)

		// Check if user has 2FA enabled - must verify before granting session
		if deps.GetUser2FA != nil {
			enabled, method, err := deps.GetUser2FA(ctx, user.ID.String())
			if err != nil {
				deps.Log.Warn("OAuth: Failed to check 2FA status", "user_id", user.ID, "error", err)
				// Continue without 2FA on error (fail open for this check)
			} else if enabled {
				deps.Log.Info("OAuth: 2FA required for user", "user_id", user.ID, "method", method, "provider", provider)

				// Create pending 2FA state
				if deps.Create2FAPendingState != nil {
					tempToken, err := deps.Create2FAPendingState(ctx, user.ID.String(), user.PrimaryEmail, user.Name, provider, method)
					if err != nil {
						deps.Log.Error("OAuth: Failed to create 2FA pending state", "user_id", user.ID, "error", err)
						return nil, fmt.Errorf("failed to initiate 2FA: %w", err)
					}

					deps.Log.Info("OAuth: 2FA pending state created", "user_id", user.ID, "method", method)

					// Send 2FA code for email/SMS methods
					if method != TwoFactorMethod("authenticator") && deps.Send2FACode != nil {
						if err := deps.Send2FACode(ctx, user.ID.String()); err != nil {
							deps.Log.Warn("OAuth: Failed to send 2FA code", "user_id", user.ID, "error", err)
						}
					}

					// Set 2FA pending attributes on claims - will be read by claims enricher
					claims.User.SetStrAttr("2fa_pending", "true")
					claims.User.SetStrAttr("2fa_temp_token", tempToken)
					claims.User.SetStrAttr("2fa_method", string(method))
					claims.User.SetStrAttr("2fa_email", maskEmail(user.PrimaryEmail))

					return nil, ErrTwoFactorRequired
				}
			}
		}

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

	// SECURITY: Block auto-linking for existing accounts.
	// User must log in with their existing credentials first, then manually link the provider.
	// This prevents account takeover if someone gains access to a user's OAuth provider.
	deps.Log.Warn("OAuth: Blocked auto-link attempt - account already exists",
		"email", maskEmail(email),
		"provider", provider)

	return nil, fmt.Errorf("an account with this email already exists. Please log in with your existing credentials first, then link your %s account from settings", provider)
}
