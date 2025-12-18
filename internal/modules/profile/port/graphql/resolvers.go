package graphql

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"hauslet/config"
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
	cdnHost        string
}

func NewResolver(profileService profileservice.ProfileService, cfg *config.StorageConfig, log *lgr.Logger) *Resolver {
	return &Resolver{profileService: profileService, cdnHost: cfg.R2.CDNHost, log: log}
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
	if input.Email != nil {
		updates["email"] = input.Email
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
	if input.HouseNumber != nil {
		updates["house_number"] = input.HouseNumber
	}
	if input.Street != nil {
		updates["street"] = input.Street
	}
	if input.Area != nil {
		updates["area"] = input.Area
	}
	if input.Lga != nil {
		updates["lga"] = input.Lga
	}
	if input.District != nil {
		updates["district"] = input.District
	}
	if input.DigitalAddress != nil {
		updates["digital_address"] = input.DigitalAddress
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
	if input.Gender != nil {
		updates["gender"] = *input.Gender
	}
	if input.ProfilePhotoURL != nil {
		urlKey := r.urlToKey(*input.ProfilePhotoURL)
		updates["photo_url"] = urlKey
	}

	updated := profile

	if len(updates) > 0 {
		updated, err = r.profileService.PatchProfile(ctx, profile.ID.String(), updates)
		if err != nil {
			r.log.Logf("ERROR Failed to patch profile for user %s: %v", v.UserID, err)
			return nil, err
		}
	}

	r.log.Logf("INFO Profile updated successfully for user %s", v.UserID)

	url := r.keyToURL(*updated.PhotoURL)
	updated.PhotoURL = &url
	return sanitizeProfileForViewer(updated, v), nil
}

// Profile is the resolver for the profile field.
func (r *Resolver) Profile(ctx context.Context, id uuid.UUID) (*domain.Profile, error) {
	p, err := r.profileService.GetProfileByID(ctx, id.String())
	if err != nil {
		r.log.Logf("ERROR Failed to get profile by ID %s: %v", id, err)
		return nil, err
	}
	url := r.keyToURL(*p.PhotoURL)
	p.PhotoURL = &url
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
	url := r.keyToURL(*p.PhotoURL)
	p.PhotoURL = &url
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
	url := r.keyToURL(*profile.PhotoURL)
	profile.PhotoURL = &url

	return sanitizeProfileForViewer(profile, v), nil
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
	clone.Email = nil
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

func (r *Resolver) urlToKey(rawURL string) string {
	if rawURL == "" {
		return rawURL
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	return strings.TrimPrefix(u.Path, "/")
}

func (r *Resolver) keyToURL(key string) string {
	return fmt.Sprintf("%s/%s", r.cdnHost, strings.TrimPrefix(key, "/"))
}

func (r *Resolver) UploadProfilePhoto(ctx context.Context, userID string, fileName string) (*domain.UploadResult, error) {
	uploadResult, err := r.profileService.UploadProfilePhoto(ctx, userID, fileName)
	if err != nil {
		r.log.Logf("ERROR Failed to upload profile photo for user %s: %v", userID, err)
		return nil, err
	}
	return uploadResult, nil
}

func (r *Resolver) UploadTravelCompanionPhoto(ctx context.Context, companionID uuid.UUID, userID string, fileName string) (*domain.UploadResult, error) {
	uploadResult, err := r.profileService.UploadTravelCompanionPhoto(ctx, companionID, userID, fileName)
	if err != nil {
		r.log.Logf("ERROR Failed to upload travel companion photo for user %s: %v", userID, err)
		return nil, err
	}
	return uploadResult, nil
}

func (r *Resolver) AddTravelCompanion(ctx context.Context, userID string, input model.TravelCompanionInput) (bool, error) {
	companion := domain.TravelCompanion{
		Name:         input.Name,
		AgeGroup:     domain.AgeGroup(input.AgeGroup),
		Gender:       domain.GenderUndisclosed,
		Phone:        input.Phone,
		Relationship: domain.Relationship(input.Relationship),
		PhotoURL:     input.PhotoURL,
	}

	if err := r.profileService.AddTravelCompanion(ctx, userID, companion); err != nil {
		r.log.Logf("ERROR Failed to add travel companion for user %s: %v", userID, err)
		return false, err
	}
	return true, nil
}

func (r *Resolver) UpdateTravelCompanion(ctx context.Context, userID string, companionID string, input model.TravelCompanionInput) (bool, error) {
	companion := domain.TravelCompanion{
		ID:           uuid.Nil,
		Name:         input.Name,
		AgeGroup:     domain.AgeGroup(input.AgeGroup),
		Gender:       domain.GenderUndisclosed,
		Phone:        input.Phone,
		Relationship: domain.Relationship(input.Relationship),
		PhotoURL:     input.PhotoURL,
	}
	if parsed, err := uuid.Parse(companionID); err == nil {
		companion.ID = parsed
	}

	if err := r.profileService.UpdateTravelCompanion(ctx, userID, companion); err != nil {
		r.log.Logf("ERROR Failed to update travel companion %s for user %s: %v", companionID, userID, err)
		return false, err
	}
	return true, nil
}

func (r *Resolver) DeleteTravelCompanion(ctx context.Context, userID string, companionID string) (bool, error) {
	if err := r.profileService.DeleteTravelCompanion(ctx, userID, companionID); err != nil {
		r.log.Logf("ERROR Failed to delete travel companion %s for user %s: %v", companionID, userID, err)
		return false, err
	}
	return true, nil
}

func (r *Resolver) DeleteProfile(ctx context.Context, userID string) (bool, error) {
	if err := r.profileService.DeleteProfile(ctx, userID); err != nil {
		r.log.Logf("ERROR Failed to delete profile for user %s: %v", userID, err)
		return false, err
	}
	return true, nil
}
