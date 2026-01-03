package graphql

import (
	"strings"

	"hauslet/internal/modules/profile/domain"
	"hauslet/internal/transport/graph/viewer"
)

func isAdminRole(role string) bool {
	adminRoles := map[string]bool{
		"admin":     true,
		"root":      true,
		"moderator": true, // Moderators might also need full access
	}
	return adminRoles[strings.ToLower(role)]
}

// sanitizeProfileForViewer removes sensitive fields from the profile
// based on the viewer's role and the profile's moderation status.
func sanitizeProfileForViewer(p *domain.Profile, v *viewer.Viewer) *domain.Profile {
	if p == nil {
		return nil
	}

	// Admins and owners see full profile regardless of moderation status
	if v != nil && (v.UserID == p.UserID || isAdminRole(v.Role)) {
		return p
	}

	// Clone the profile to avoid modifying the original
	clone := *p

	// Always hide sensitive contact and privacy fields for non-owners/admins
	clone.PhoneNumbers = nil
	clone.Email = nil
	clone.BirthDate = nil
	clone.Address = nil
	clone.ZipCode = nil
	clone.TravelCompanions = nil
	clone.BioVisible = false
	clone.AllowPersonalizedOffers = false
	clone.EnablePerformanceAnalytics = false

	// Hide verification status fields (only visible to profile owner and admins)
	clone.PhoneVerified = false
	clone.IDVerified = false
	clone.VerificationDate = nil
	clone.VerificationLevel = ""

	// If profile is NOT moderated, hide additional free-text fields that could contain inappropriate content
	if !p.IsModerated {
		// Hide bio and personal free-text fields
		clone.Bio = nil
		clone.FunFact = nil
		clone.ObsessedWith = nil

		// Hide occupation and education
		clone.Occupation = nil
		clone.Education = nil

		// Hide detailed address information
		clone.City = nil
		clone.State = nil
		clone.Country = nil
		clone.HouseNumber = nil
		clone.Street = nil
		clone.Area = nil
		clone.LGA = nil
		clone.District = nil
		clone.DigitalAddress = nil

		// Hide personal lists (could contain inappropriate content)
		clone.Skills = nil
		clone.Languages = nil
		clone.Interests = nil
		clone.Hobbies = nil

		// Show a placeholder for full name instead of actual name
		if clone.FullName != "" {
			// Show only first name or a placeholder
			parts := strings.Fields(clone.FullName)
			if len(parts) > 0 {
				clone.FullName = parts[0] + "."
			} else {
				clone.FullName = "User"
			}
		}

		// Optionally hide photo for unmoderated profiles
		// clone.PhotoURL = nil

		// Hide rating and badges for unmoderated profiles
		clone.Rating = 0
		clone.ReviewsCount = 0
		clone.Badges = nil
		clone.TrustScore = 0

	} else {
		// Profile IS moderated - show most fields but still hide sensitive ones
		// Still hide detailed address information for privacy
		clone.City = nil
		clone.State = nil
		clone.Country = nil
		clone.HouseNumber = nil
		clone.Street = nil
		clone.Area = nil
		clone.LGA = nil
		clone.District = nil
		clone.DigitalAddress = nil

		// For moderated profiles, we can show more but still be cautious
		// Show full name as is
		// Show bio, fun facts, etc.
		// Personal lists (skills, languages, etc.) can be shown
	}

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

// buildProfileUpdates builds a map of updates from the input
func (r *Resolver) buildProfileUpdates(input UpdateProfileInput) map[string]any {
	updates := make(map[string]any)

	// Personal Information
	if input.FullName != nil {
		updates["full_name"] = input.FullName
	}
	if input.BirthDate != nil {
		updates["birth_date"] = input.BirthDate
	}
	if input.Gender != nil {
		updates["gender"] = *input.Gender
	}
	if input.ProfilePhotoURL != nil {
		urlKey := r.urlToKey(*input.ProfilePhotoURL)
		updates["photo_url"] = urlKey
	}

	// Contact Information
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

	// Personal Details
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

	// Community & Privacy Settings
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

	return updates
}
