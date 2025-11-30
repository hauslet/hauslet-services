package graph

import (
	"context"
	"fmt"

	"hauslet/internal/profile/domain"
)

func isAdminRole(role string) bool {
	return role == "admin" || role == "root"
}
func ensureMutationAllowed(ctx context.Context, targetUserID string) error {
	viewer := viewerFromContext(ctx)
	if viewer == nil || viewer.UserID == "" {
		return fmt.Errorf("unauthenticated")
	}
	if viewer.UserID != targetUserID && !isAdminRole(viewer.Role) {
		return fmt.Errorf("forbidden")
	}
	return nil
}
func sanitizeProfileForViewer(p *domain.Profile, viewer *Viewer) *domain.Profile {
	if p == nil {
		return nil
	}
	// Admins and owners see full profile.
	if viewer != nil && (viewer.UserID == p.UserID || isAdminRole(viewer.Role)) {
		return p
	}

	clone := *p
	clone.PhoneNumbers = nil
	clone.Address = nil
	clone.ZipCode = nil
	clone.TravelCompanions = nil
	return &clone
}

// sanitizeProfilesForViewer applies sanitization to a slice of profiles.
func sanitizeProfilesForViewer(profiles []domain.Profile, viewer *Viewer) []*domain.Profile {
	if len(profiles) == 0 {
		return []*domain.Profile{}
	}
	out := make([]*domain.Profile, 0, len(profiles))
	for i := range profiles {
		if p := sanitizeProfileForViewer(&profiles[i], viewer); p != nil {
			out = append(out, p)
		}
	}
	return out
}
