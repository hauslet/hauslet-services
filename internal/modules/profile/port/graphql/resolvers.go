package graphql

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strings"

	"hauslet/config"
	"hauslet/internal/modules/profile/domain"
	profileservice "hauslet/internal/modules/profile/service"
	"hauslet/internal/transport/graph/loaders"

	"hauslet/internal/transport/graph/viewer"

	"github.com/google/uuid"
)

// Resolver handles profile-specific GraphQL fields.
type Resolver struct {
	profileService profileservice.ProfileService
	log            *slog.Logger
	cdnHost        string
}

func NewResolver(profileService profileservice.ProfileService, cfg *config.StorageConfig, log *slog.Logger) *Resolver {
	return &Resolver{profileService: profileService, cdnHost: cfg.R2.CDNHost, log: log}
}

// UpdateProfile is the resolver for the updateProfile field.
func (r *Resolver) UpdateProfile(ctx context.Context, input UpdateProfileInput) (*domain.Profile, error) {

	userID, err := viewer.GetUserIDFromContext(ctx)
	if err != nil {
		r.log.Warn("Unauthenticated attempt to update profile")
		return nil, err
	}
	// Load current profile for the authenticated user
	profile, err := r.profileService.GetProfileByUserID(ctx, userID.String())
	if err != nil {
		r.log.Error("Failed to get profile for user", "user_id", userID, "error", err)
		return nil, err
	}
	if profile == nil {
		r.log.Warn("Profile not found for user", "user_id", userID)
		return nil, fmt.Errorf("profile not found")
	}

	// Build updates from input
	updates := r.buildProfileUpdates(input)

	// If no updates, return current profile
	if len(updates) == 0 {
		r.log.Info("No updates provided for user", "user_id", userID)
		return sanitizeProfileForViewer(profile, userID.String()), nil
	}

	// Apply updates
	updated, err := r.profileService.PatchProfile(ctx, profile.ID.String(), updates)
	if err != nil {
		r.log.Error("Failed to patch profile for user", "user_id", userID, "error", err)
		return nil, err
	}

	r.log.Info("Profile updated successfully for user", "user_id", userID)
	// Convert photo key back to URL for response
	if updated.PhotoURL != nil && *updated.PhotoURL != "" {
		url := r.keyToURL(*updated.PhotoURL)
		updated.PhotoURL = &url
	}

	return sanitizeProfileForViewer(updated, userID.String()), nil
}

// SelectSupplyRoles assigns supply-side roles for the authenticated user.
func (r *Resolver) SelectSupplyRoles(ctx context.Context, userTypes []domain.UserType) (*domain.Profile, error) {

	userID, err := viewer.GetUserIDFromContext(ctx)
	if err != nil {
		r.log.Warn("Unauthenticated attempt to update profile")
		return nil, err
	}
	updated, err := r.profileService.SelectSupplyRoles(ctx, userID.String(), userTypes)
	if err != nil {
		r.log.Error("Failed to select supply roles", "user_id", userID, "error", err)
		return nil, err
	}

	if updated != nil && updated.PhotoURL != nil && *updated.PhotoURL != "" {
		url := r.keyToURL(*updated.PhotoURL)
		updated.PhotoURL = &url
	}

	r.log.Info("Supply roles updated successfully for user", "user_id", userID)
	return sanitizeProfileForViewer(updated, userID.String()), nil
}

// Profile is the resolver for the profile field.
func (r *Resolver) Profile(ctx context.Context, id uuid.UUID) (*domain.Profile, error) {
	p, err := r.profileService.GetProfileByID(ctx, id.String())
	if err != nil {
		r.log.Error("Failed to get profile by ID", "id", id.String(), "error", err)
		return nil, err
	}
	if p != nil && p.PhotoURL != nil && *p.PhotoURL != "" {
		url := r.keyToURL(*p.PhotoURL)
		p.PhotoURL = &url
	}
	return sanitizeProfileForViewer(p, ""), nil
}

// ProfileByUserID is the resolver for the profileByUserId field.
func (r *Resolver) ProfileByUserID(ctx context.Context, userID string) (*domain.Profile, error) {
	if l := loaders.For(ctx); l != nil && l.Profile != nil {
		if p, err := l.Profile.Load(ctx, userID); err == nil {
			return sanitizeProfileForViewer(p, userID), nil
		}
	}

	p, err := r.profileService.GetProfileByUserID(ctx, userID)
	if err != nil {
		r.log.Error("Failed to get profile by user ID", "user_id", userID, "error", err)
		return nil, err
	}
	if p != nil && p.PhotoURL != nil && *p.PhotoURL != "" {
		url := r.keyToURL(*p.PhotoURL)
		p.PhotoURL = &url
	}
	return sanitizeProfileForViewer(p, userID), nil
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
		r.log.Error("Failed to list profiles", "error", err)
		return nil, err
	}
	return sanitizeProfilesForViewer(profiles, ""), nil
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
		r.log.Error("Failed to search profiles with query", "query", query, "error", err)
		return nil, err
	}
	return sanitizeProfilesForViewer(profiles, ""), nil
}

// MyProfile is the resolver for the myProfile field.
func (r *Resolver) MyProfile(ctx context.Context) (*domain.Profile, error) {

	userID, err := viewer.GetUserIDFromContext(ctx)
	if err != nil {
		r.log.Warn("Unauthenticated attempt to access myProfile")
		return nil, err
	}

	profile, err := r.profileService.GetProfileByUserID(ctx, userID.String())
	if err != nil {
		r.log.Error("Failed to get profile for user", "user_id", userID, "error", err)
		return nil, err
	}
	if profile != nil && profile.PhotoURL != nil && *profile.PhotoURL != "" {
		url := r.keyToURL(*profile.PhotoURL)
		profile.PhotoURL = &url
	}

	return sanitizeProfileForViewer(profile, userID.String()), nil
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
		r.log.Error("Failed to upload profile photo for user", "user_id", userID, "error", err)
		return nil, err
	}
	return uploadResult, nil
}

func (r *Resolver) UploadTravelCompanionPhoto(ctx context.Context, companionID uuid.UUID, userID string, fileName string) (*domain.UploadResult, error) {
	uploadResult, err := r.profileService.UploadTravelCompanionPhoto(ctx, companionID, userID, fileName)
	if err != nil {
		r.log.Error("Failed to upload travel companion photo for user", "user_id", userID, "error", err)
		return nil, err
	}
	return uploadResult, nil
}

func (r *Resolver) AddTravelCompanion(ctx context.Context, userID string, input TravelCompanionInput) (bool, error) {
	companion := domain.TravelCompanion{
		Name:         input.Name,
		AgeGroup:     domain.AgeGroup(input.AgeGroup),
		Gender:       domain.GenderUndisclosed,
		Phone:        input.Phone,
		Relationship: domain.Relationship(input.Relationship),
		PhotoURL:     input.PhotoURL,
	}

	if err := r.profileService.AddTravelCompanion(ctx, userID, companion); err != nil {
		r.log.Error("Failed to add travel companion for user", "user_id", userID, "error", err)
		return false, err
	}
	return true, nil
}

func (r *Resolver) UpdateTravelCompanion(ctx context.Context, userID string, companionID string, input TravelCompanionInput) (bool, error) {
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
		r.log.Error("Failed to update travel companion", "companion_id", companionID, "user_id", userID, "error", err)
		return false, err
	}
	return true, nil
}

func (r *Resolver) DeleteTravelCompanion(ctx context.Context, userID string, companionID string) (bool, error) {
	if err := r.profileService.DeleteTravelCompanion(ctx, userID, companionID); err != nil {
		r.log.Error("Failed to delete travel companion", "companion_id", companionID, "user_id", userID, "error", err)
		return false, err
	}
	return true, nil
}

func (r *Resolver) DeleteProfile(ctx context.Context, userID string) (bool, error) {
	if err := r.profileService.DeleteProfile(ctx, userID); err != nil {
		r.log.Error("Failed to delete profile for user", "user_id", userID, "error", err)
		return false, err
	}
	return true, nil
}
