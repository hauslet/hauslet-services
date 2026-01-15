package hooks

import (
	"context"
	"fmt"
	messagingdomain "hauslet/internal/modules/messaging/domain"
	profileservice "hauslet/internal/modules/profile/service"
	"strconv"

	"github.com/google/uuid"
)

var _ messagingdomain.ProfileHooks = (*profileHooksAdapter)(nil)

// ProfileHooks defines the contract for user profile data
type ProfileHooks interface {
	// GetProfileData fetches minimal profile metadata for given user IDs
	GetProfileData(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]map[string]string, error)
}

// profileHooksAdapter fetches minimal profile metadata for messaging.
type profileHooksAdapter struct {
	profileSvc profileservice.ProfileService
}

// NewProfileHooksAdapter wires the profile service into messaging.
func NewProfileHooksAdapter(profileSvc profileservice.ProfileService) messagingdomain.ProfileHooks {
	return &profileHooksAdapter{profileSvc: profileSvc}
}

func (a *profileHooksAdapter) GetProfileData(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]map[string]string, error) {
	if a.profileSvc == nil {
		return nil, fmt.Errorf("profile service not configured")
	}

	results := make(map[uuid.UUID]map[string]string, len(userIDs))
	for _, uid := range userIDs {
		profile, err := a.profileSvc.GetProfileByUserID(ctx, uid.String())
		if err != nil {
			return nil, fmt.Errorf("failed to load profile for user %s: %w", uid, err)
		}
		if profile == nil {
			continue
		}

		entry := map[string]string{
			"display_name": profile.FullName,
			"joined_year":  strconv.Itoa(profile.CreatedAt.Year()),
			"avatar_url":   "",
		}
		if profile.PhotoURL != nil {
			entry["avatar_url"] = *profile.PhotoURL
		}

		results[uid] = entry
	}

	return results, nil
}

func (a *profileHooksAdapter) GetUserContact(ctx context.Context, userID uuid.UUID) (*messagingdomain.UserContact, error) {
	if a.profileSvc == nil {
		return nil, fmt.Errorf("profile service not configured")
	}

	profile, err := a.profileSvc.GetProfileByUserID(ctx, userID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to load profile for user %s: %w", userID, err)
	}
	if profile == nil {
		return nil, nil
	}

	contact := &messagingdomain.UserContact{
		UserID: userID,
		Name:   profile.FullName,
	}
	if profile.Email != nil {
		contact.Email = *profile.Email
	}

	return contact, nil
}
