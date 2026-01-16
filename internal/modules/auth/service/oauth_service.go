package service

import (
	"context"
	"fmt"
	"time"

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

		deps := oauth.Dependencies{
			Config:                s.cfg,
			Repository:            s.repository,
			Log:                   s.log,
			MetadataFetcher:       metadataFetcher,
			LinkStateValidator:    linkStateValidator,
			AuthenticatePassword:  s.AuthenticatePassword,
			LinkIdentity:          linkIdentity,
			SendWelcomeEmail:      s.SendWelcomeEmail,
			SendIdentityLinked:    s.SendIdentityLinkedEmail,
			SendPasswordlessEmail: s.SendPasswordlessLoginEmail,
			GenerateAndStoreOTP:   s.GeneratePasswordlessOTP,
			ProfileHook: func(ctx context.Context, userID, email, name string, birthDate *time.Time) error {
				if s.profileHooks == nil {
					return nil
				}
				return s.profileHooks.CreateDefaultProfile(ctx, userID, email, name, birthDate)
			},
			ProfileAvatarFetcher: profileAvatarFetcher,
		}

		s.oauthService = oauth.NewService(deps)
	})

	return s.oauthService
}
