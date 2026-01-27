package service

import (
	"context"
	"fmt"
	"time"

	"hauslet/internal/modules/auth/domain"
	"hauslet/internal/modules/auth/service/oauth"

	"github.com/go-pkgz/auth/v2"
)

func (s *AuthServiceImpl) OAuthService() *auth.Service {
	s.oauthOnce.Do(func() {
		var metadataFetcher oauth.MetadataFetcher
		if s.requestMetadata != nil {
			metadataFetcher = func(key string) *oauth.RequestMetadata {
				if metadata := s.requestMetadata.Get(key); metadata != nil {
					return &oauth.RequestMetadata{IP: metadata.IP, UserAgent: metadata.UserAgent}
				}
				return nil
			}
		}

		var linkStateValidator oauth.LinkStateValidator
		if s.linkStateManager != nil {
			linkStateValidator = func(state string) (*oauth.LinkState, error) {
				linkState, err := s.linkStateManager.ValidateState(state)
				if err != nil {
					return nil, err
				}
				return &oauth.LinkState{
					UserID:      linkState.UserID,
					Provider:    linkState.Provider,
					RedirectURI: linkState.RedirectURI,
				}, nil
			}
		}

		var linkIdentity oauth.LinkIdentityFunc
		if s.linkStateManager != nil {
			linkIdentity = func(ctx context.Context, linkState *oauth.LinkState, provider, providerUserID, email string) error {
				if linkState == nil {
					return fmt.Errorf("link state required")
				}
				return s.linkIdentityToUser(ctx, &LinkState{
					UserID:      linkState.UserID,
					Provider:    linkState.Provider,
					RedirectURI: linkState.RedirectURI,
				}, provider, providerUserID, email)
			}
		}

		var profileAvatarFetcher oauth.ProfileAvatarFetcher
		if s.profileHooks != nil {
			profileAvatarFetcher = func(ctx context.Context, userID string) (*string, error) {
				return s.profileHooks.GetProfileAvatarURL(ctx, userID)
			}
		}

		// 2FA login flow dependencies
		getUser2FA := func(ctx context.Context, userID string) (bool, oauth.TwoFactorMethod, error) {
			status, err := s.Get2FAStatus(ctx, userID)
			if err != nil {
				return false, "", err
			}
			if status == nil || !status.Enabled {
				return false, "", nil
			}
			return true, oauth.TwoFactorMethod(status.Method), nil
		}

		create2FAPendingState := func(ctx context.Context, userID, email, name, provider string, method oauth.TwoFactorMethod) (string, error) {
			return s.Create2FAPendingState(ctx, userID, email, name, provider, domainTwoFactorMethod(method))
		}

		send2FACode := func(ctx context.Context, userID string) error {
			return s.Send2FACode(ctx, userID)
		}

		deps := oauth.Dependencies{
			Config:                s.cfg,
			Repository:            s.repository,
			Log:                   s.log,
			MetadataFetcher:       metadataFetcher,
			LinkStateValidator:    linkStateValidator,
			AuthenticatePassword:  s.AuthenticatePassword,
			LinkIdentity:          linkIdentity,
			SendWelcomeEmail:      s.notifier.SendWelcomeEmail,
			SendIdentityLinked:    s.notifier.SendIdentityLinkedEmail,
			SendPasswordlessEmail: s.notifier.SendPasswordlessLoginEmail,
			GenerateAndStoreOTP:   s.GeneratePasswordlessOTP,
			ProfileHook: func(ctx context.Context, userID, email, name string, birthDate *time.Time) error {
				if s.profileHooks == nil {
					return nil
				}
				return s.profileHooks.CreateDefaultProfile(ctx, userID, email, name, birthDate)
			},
			ProfileAvatarFetcher: profileAvatarFetcher,
			// 2FA dependencies
			GetUser2FA:            getUser2FA,
			Create2FAPendingState: create2FAPendingState,
			Send2FACode:           send2FACode,
		}

		s.oauthService = oauth.NewService(deps)
	})

	return s.oauthService
}

// domainTwoFactorMethod converts oauth.TwoFactorMethod to domain.TwoFactorMethod
func domainTwoFactorMethod(m oauth.TwoFactorMethod) domain.TwoFactorMethod {
	return domain.TwoFactorMethod(m)
}
