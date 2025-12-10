package graphql

import (
	"context"
	"fmt"

	"hauslet/internal/modules/profile/domain"
	profileservice "hauslet/internal/modules/profile/service"
	"hauslet/internal/transport/graph/loaders"
	"hauslet/internal/transport/graph/model"
	"hauslet/internal/transport/graph/viewer"

	"github.com/go-pkgz/lgr"
	"github.com/google/uuid"
)

// Resolver handles profile-specific GraphQL fields.
type Resolver struct {
	profileService profileservice.ProfileService
	log            *lgr.Logger
}

func NewResolver(profileService profileservice.ProfileService, log *lgr.Logger) *Resolver {
	return &Resolver{profileService: profileService, log: log}
}

// UpdateProfile is the resolver for the updateProfile field.
func (r *Resolver) UpdateProfile(ctx context.Context, input model.UpdateProfileInput) (*domain.Profile, error) {
	v := viewer.FromContext(ctx)
	if v == nil || v.UserID == "" {
		r.log.Logf("WARN Unauthenticated attempt to update profile")
		return nil, fmt.Errorf("unauthenticated")
	}

	// Load current profile for the authenticated user
	profile, err := r.profileService.GetProfileByUserID(ctx, v.UserID)
	if err != nil {
		r.log.Logf("ERROR Failed to get profile for user %s: %v", v.UserID, err)
		return nil, err
	}
	if profile == nil {
		r.log.Logf("WARN Profile not found for user %s", v.UserID)
		return nil, fmt.Errorf("profile not found")
	}

	updates := make(map[string]any)
	if input.FullName != nil {
		updates["full_name"] = input.FullName
	}
	if input.BirthDate != nil {
		updates["birth_date"] = input.BirthDate
	}
	if input.PhoneNumbers != nil {
		updates["phone_numbers"] = input.PhoneNumbers
	}
	if input.Address != nil {
		updates["address"] = input.Address
	}
	if input.City != nil {
		updates["city"] = input.City
	}
	if input.State != nil {
		updates["state"] = input.State
	}
	if input.Country != nil {
		updates["country"] = input.Country
	}
	if input.ZipCode != nil {
		updates["zip_code"] = input.ZipCode
	}
	if input.Occupation != nil {
		updates["occupation"] = input.Occupation
	}
	if input.Education != nil {
		updates["education"] = input.Education
	}
	if input.Bio != nil {
		updates["bio"] = input.Bio
	}
	if input.Skills != nil {
		updates["skills"] = input.Skills
	}
	if input.Languages != nil {
		updates["languages"] = input.Languages
	}
	if input.Interests != nil {
		updates["interests"] = input.Interests
	}
	if input.Hobbies != nil {
		updates["hobbies"] = input.Hobbies
	}
	if input.FunFact != nil {
		updates["fun_fact"] = input.FunFact
	}
	if input.ObsessedWith != nil {
		updates["obsessed_with"] = input.ObsessedWith
	}
	if input.CommunityCommitment != nil {
		updates["community_commitment"] = *input.CommunityCommitment
	}
	if input.BioVisible != nil {
		updates["bio_visible"] = *input.BioVisible
	}
	if input.AllowPersonalizedOffers != nil {
		updates["allow_personalized_offers"] = *input.AllowPersonalizedOffers
	}
	if input.EnablePerformanceAnalytics != nil {
		updates["enable_performance_analytics"] = *input.EnablePerformanceAnalytics
	}

	companionsProvided := input.TravelCompanions != nil
	var companions []domain.TravelCompanion
	if companionsProvided {
		companions = make([]domain.TravelCompanion, 0, len(input.TravelCompanions))
		for _, tc := range input.TravelCompanions {
			if tc == nil {
				continue
			}
			companions = append(companions, domain.TravelCompanion{
				Name:         tc.Name,
				AgeGroup:     tc.AgeGroup,
				Phone:        tc.Phone,
				Relationship: tc.Relationship,
				PhotoURL:     tc.PhotoURL,
			})
		}
	}

	updated := profile
	if len(updates) > 0 {
		updated, err = r.profileService.PatchProfile(ctx, profile.ID.String(), updates)
		if err != nil {
			r.log.Logf("ERROR Failed to patch profile for user %s: %v", v.UserID, err)
			return nil, err
		}
	}

	if companionsProvided {
		if err := r.profileService.UpdateTravelCompanions(ctx, v.UserID, companions); err != nil {
			r.log.Logf("ERROR Failed to update travel companions for user %s: %v", v.UserID, err)
			return nil, err
		}
		if updated != nil {
			updated.TravelCompanions = companions
		}
	}

	r.log.Logf("INFO Profile updated successfully for user %s", v.UserID)
	return sanitizeProfileForViewer(updated, v), nil
}

// Profile is the resolver for the profile field.
func (r *Resolver) Profile(ctx context.Context, id uuid.UUID) (*domain.Profile, error) {
	p, err := r.profileService.GetProfileByID(ctx, id.String())
	if err != nil {
		r.log.Logf("ERROR Failed to get profile by ID %s: %v", id, err)
		return nil, err
	}
	return sanitizeProfileForViewer(p, viewer.FromContext(ctx)), nil
}

// ProfileByUserID is the resolver for the profileByUserId field.
func (r *Resolver) ProfileByUserID(ctx context.Context, userID string) (*domain.Profile, error) {
	if l := loaders.For(ctx); l != nil && l.Profile != nil {
		if p, err := l.Profile.Load(ctx, userID); err == nil {
			return sanitizeProfileForViewer(p, viewer.FromContext(ctx)), nil
		}
	}

	p, err := r.profileService.GetProfileByUserID(ctx, userID)
	if err != nil {
		r.log.Logf("ERROR Failed to get profile by user ID %s: %v", userID, err)
		return nil, err
	}
	return sanitizeProfileForViewer(p, viewer.FromContext(ctx)), nil
}

// Profiles is the resolver for the profiles field.
func (r *Resolver) Profiles(ctx context.Context, limit *int, offset *int) ([]*domain.Profile, error) {
	l := 20
	o := 0
	if limit != nil {
		l = *limit
	}
	if offset != nil {
		o = *offset
	}

	profiles, err := r.profileService.ListProfiles(ctx, l, o)
	if err != nil {
		r.log.Logf("ERROR Failed to list profiles: %v", err)
		return nil, err
	}
	return sanitizeProfilesForViewer(profiles, viewer.FromContext(ctx)), nil
}

// SearchProfiles is the resolver for the searchProfiles field.
func (r *Resolver) SearchProfiles(ctx context.Context, query string, limit *int, offset *int) ([]*domain.Profile, error) {
	l := 20
	o := 0
	if limit != nil {
		l = *limit
	}
	if offset != nil {
		o = *offset
	}

	profiles, err := r.profileService.SearchProfiles(ctx, query, l, o)
	if err != nil {
		r.log.Logf("ERROR Failed to search profiles with query '%s': %v", query, err)
		return nil, err
	}
	return sanitizeProfilesForViewer(profiles, viewer.FromContext(ctx)), nil
}

// MyProfile is the resolver for the myProfile field.
func (r *Resolver) MyProfile(ctx context.Context) (*domain.Profile, error) {
	v := viewer.FromContext(ctx)
	if v == nil || v.UserID == "" {
		r.log.Logf("WARN Unauthenticated attempt to access myProfile")
		return nil, fmt.Errorf("unauthenticated")
	}

	profile, err := r.profileService.GetProfileByUserID(ctx, v.UserID)
	if err != nil {
		r.log.Logf("ERROR Failed to get profile for user %s: %v", v.UserID, err)
		return nil, err
	}

	return sanitizeProfileForViewer(profile, v), nil
}

// Badges resolves the detailed badge objects for a profile.
func (r *Resolver) Badges(ctx context.Context, obj *domain.Profile) ([]*domain.BadgeDetails, error) {
	badges := make([]*domain.BadgeDetails, 0, len(obj.Badges))
	for _, b := range obj.Badges {
		details := domain.GetBadgeDetails(b)
		badge := details // copy to avoid pointer reuse in loop
		badges = append(badges, &badge)
	}
	return badges, nil
}

func isAdminRole(role string) bool {
	return role == "admin" || role == "root"
}

func sanitizeProfileForViewer(p *domain.Profile, v *viewer.Viewer) *domain.Profile {
	if p == nil {
		return nil
	}
	// Admins and owners see full profile.
	if v != nil && (v.UserID == p.UserID || isAdminRole(v.Role)) {
		return p
	}

	clone := *p
	clone.PhoneNumbers = nil
	clone.Address = nil
	clone.ZipCode = nil
	clone.TravelCompanions = nil
	clone.BioVisible = false
	clone.AllowPersonalizedOffers = false
	clone.EnablePerformanceAnalytics = false
	return &clone
}

// sanitizeProfilesForViewer applies sanitization to a slice of profiles.
func sanitizeProfilesForViewer(profiles []domain.Profile, v *viewer.Viewer) []*domain.Profile {
	if len(profiles) == 0 {
		return []*domain.Profile{}
	}
	out := make([]*domain.Profile, 0, len(profiles))
	for i := range profiles {
		if p := sanitizeProfileForViewer(&profiles[i], v); p != nil {
			out = append(out, p)
		}
	}
	return out
}
