package hooks

import (
	"context"
	"strings"
	"time"

	"hauslet/internal/modules/profile/domain"
	"hauslet/internal/modules/profile/service"
)

// AuthHooksAdapter translates calls from the Auth module into Profile domain logic.
type AuthHooksAdapter struct {
	svc     service.ProfileService
	cdnHost string
}

// NewAuthHooksAdapter creates the adapter
func NewAuthHooksAdapter(svc service.ProfileService, cdnHost string) *AuthHooksAdapter {
	return &AuthHooksAdapter{svc: svc, cdnHost: cdnHost}
}

// CreateDefaultProfile satisfies the Auth module's "ProfileHooks" interface
func (a *AuthHooksAdapter) CreateDefaultProfile(ctx context.Context, userID, email, name string, birthDate *time.Time) error {
	// 1. Translate "scalars" (strings/dates) into "Domain Object"
	newProfile := domain.Profile{
		UserID:    userID,
		UserTypes: []domain.UserType{domain.Guest},
		FullName:  name,
		BirthDate: birthDate,
		Email:     &email,
	}

	// 2. Call the internal service
	_, err := a.svc.CreateProfile(ctx, newProfile)
	return err
}

// GetProfileAvatarURL returns the CDN URL for the user's profile photo (if any).
func (a *AuthHooksAdapter) GetProfileAvatarURL(ctx context.Context, userID string) (*string, error) {
	profile, err := a.svc.GetProfileByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if profile == nil || profile.PhotoURL == nil || *profile.PhotoURL == "" {
		return nil, nil
	}

	url := a.keyToURL(*profile.PhotoURL)
	return &url, nil
}

// keyToURL converts an object key to a CDN-backed URL.
func (a *AuthHooksAdapter) keyToURL(key string) string {
	if key == "" {
		return key
	}
	return a.cdnHost + "/" + strings.TrimPrefix(key, "/")
}
