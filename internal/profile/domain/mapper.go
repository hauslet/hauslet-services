package domain

import (
	"errors"

	"hauslet/internal/profile/repository/schema"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// MapProfileFromSchema converts a repository profile to a domain profile.
// SECURITY: Excludes soft-delete metadata beyond DeletedAt timestamp.
func MapProfileFromSchema(schemaProfile *schema.Profile) *Profile {
	if schemaProfile == nil {
		return nil
	}

	profile := &Profile{
		ID:                         schemaProfile.ID,
		UserID:                     schemaProfile.UserID.String(),
		UserTypes:                  make([]UserType, len(schemaProfile.UserTypes)),
		FullName:                   schemaProfile.FullName,
		BirthDate:                  schemaProfile.BirthDate,
		PhoneNumbers:               make([]string, len(schemaProfile.PhoneNumbers)),
		Address:                    schemaProfile.Address,
		City:                       schemaProfile.City,
		State:                      schemaProfile.State,
		Country:                    schemaProfile.Country,
		ZipCode:                    schemaProfile.ZipCode,
		Occupation:                 schemaProfile.Occupation,
		Education:                  schemaProfile.Education,
		Bio:                        schemaProfile.Bio,
		Skills:                     schemaProfile.Skills,
		Languages:                  schemaProfile.Languages,
		Interests:                  schemaProfile.Interests,
		Hobbies:                    schemaProfile.Hobbies,
		FunFact:                    schemaProfile.FunFact,
		ObsessedWith:               schemaProfile.ObsessedWith,
		CommunityCommitment:        schemaProfile.CommunityCommitment,
		TravelCompanions:           make([]TravelCompanion, len(schemaProfile.TravelCompanions)),
		PhoneVerified:              schemaProfile.PhoneVerified,
		IDVerified:                 schemaProfile.IDVerified,
		VerificationDate:           schemaProfile.VerificationDate,
		VerificationLevel:          schemaProfile.VerificationLevel,
		Rating:                     schemaProfile.Rating,
		ReviewsCount:               schemaProfile.ReviewsCount,
		Badges:                     make([]Badge, len(schemaProfile.Badges)),
		TrustScore:                 schemaProfile.TrustScore,
		BioVisible:                 schemaProfile.BioVisible,
		AllowPersonalizedOffers:    schemaProfile.AllowPersonalizedOffers,
		EnablePerformanceAnalytics: schemaProfile.EnablePerformanceAnalytics,
		CreatedAt:                  schemaProfile.CreatedAt,
		UpdatedAt:                  schemaProfile.UpdatedAt,
	}

	for i, t := range schemaProfile.UserTypes {
		profile.UserTypes[i] = UserType(t)
	}

	copy(profile.PhoneNumbers, schemaProfile.PhoneNumbers)

	for i, companion := range schemaProfile.TravelCompanions {
		profile.TravelCompanions[i] = MapTravelCompanionFromSchema(companion)
	}

	for i, badge := range schemaProfile.Badges {
		profile.Badges[i] = Badge(badge)
	}

	if schemaProfile.DeletedAt.Valid {
		profile.DeletedAt = &schemaProfile.DeletedAt.Time
	}

	return profile
}

// MapProfileToSchema converts a domain profile to a repository profile.
func MapProfileToSchema(domainProfile *Profile) (*schema.Profile, error) {
	if domainProfile == nil {
		return nil, errors.New("profile cannot be nil")
	}

	if domainProfile.UserID == "" {
		return nil, errors.New("userID cannot be empty")
	}

	userUUID, err := uuid.Parse(domainProfile.UserID)
	if err != nil {
		return nil, errors.New("invalid userID format")
	}

	schemaProfile := &schema.Profile{
		ID:                         domainProfile.ID,
		UserID:                     userUUID,
		UserTypes:                  make(pq.StringArray, len(domainProfile.UserTypes)),
		FullName:                   domainProfile.FullName,
		BirthDate:                  domainProfile.BirthDate,
		PhoneNumbers:               make(pq.StringArray, len(domainProfile.PhoneNumbers)),
		Address:                    domainProfile.Address,
		City:                       domainProfile.City,
		State:                      domainProfile.State,
		Country:                    domainProfile.Country,
		ZipCode:                    domainProfile.ZipCode,
		Occupation:                 domainProfile.Occupation,
		Education:                  domainProfile.Education,
		Bio:                        domainProfile.Bio,
		Skills:                     pq.StringArray(domainProfile.Skills),
		Languages:                  pq.StringArray(domainProfile.Languages),
		Interests:                  pq.StringArray(domainProfile.Interests),
		Hobbies:                    pq.StringArray(domainProfile.Hobbies),
		FunFact:                    domainProfile.FunFact,
		ObsessedWith:               domainProfile.ObsessedWith,
		CommunityCommitment:        domainProfile.CommunityCommitment,
		TravelCompanions:           make([]schema.TravelCompanionProfile, len(domainProfile.TravelCompanions)),
		PhoneVerified:              domainProfile.PhoneVerified,
		IDVerified:                 domainProfile.IDVerified,
		VerificationDate:           domainProfile.VerificationDate,
		VerificationLevel:          domainProfile.VerificationLevel,
		Rating:                     domainProfile.Rating,
		ReviewsCount:               domainProfile.ReviewsCount,
		Badges:                     make([]string, len(domainProfile.Badges)),
		TrustScore:                 domainProfile.TrustScore,
		BioVisible:                 domainProfile.BioVisible,
		AllowPersonalizedOffers:    domainProfile.AllowPersonalizedOffers,
		EnablePerformanceAnalytics: domainProfile.EnablePerformanceAnalytics,
		CreatedAt:                  domainProfile.CreatedAt,
		UpdatedAt:                  domainProfile.UpdatedAt,
	}

	for i, t := range domainProfile.UserTypes {
		schemaProfile.UserTypes[i] = string(t)
	}

	copy(schemaProfile.PhoneNumbers, domainProfile.PhoneNumbers)

	for i, companion := range domainProfile.TravelCompanions {
		schemaProfile.TravelCompanions[i] = MapTravelCompanionToSchema(companion)
	}

	for i, badge := range domainProfile.Badges {
		schemaProfile.Badges[i] = string(badge)
	}

	if domainProfile.DeletedAt != nil {
		schemaProfile.DeletedAt.Time = *domainProfile.DeletedAt
		schemaProfile.DeletedAt.Valid = true
	}

	return schemaProfile, nil
}

// MapProfilesFromSchema converts a slice of schema profiles to domain profiles.
func MapProfilesFromSchema(schemaProfiles []*schema.Profile) []Profile {
	if schemaProfiles == nil {
		return nil
	}

	profiles := make([]Profile, 0, len(schemaProfiles))
	for _, sp := range schemaProfiles {
		if mapped := MapProfileFromSchema(sp); mapped != nil {
			profiles = append(profiles, *mapped)
		}
	}

	return profiles
}

// MapTravelCompanionFromSchema converts schema travel companion to domain.
func MapTravelCompanionFromSchema(schemaCompanion schema.TravelCompanionProfile) TravelCompanion {
	return TravelCompanion{
		Name:         schemaCompanion.Name,
		AgeGroup:     schemaCompanion.AgeGroup,
		Phone:        schemaCompanion.Phone,
		Relationship: schemaCompanion.Relationship,
		PhotoURL:     schemaCompanion.PhotoURL,
	}
}

// MapTravelCompanionToSchema converts domain travel companion to schema.
func MapTravelCompanionToSchema(domainCompanion TravelCompanion) schema.TravelCompanionProfile {
	return schema.TravelCompanionProfile{
		Name:         domainCompanion.Name,
		AgeGroup:     domainCompanion.AgeGroup,
		Phone:        domainCompanion.Phone,
		Relationship: domainCompanion.Relationship,
		PhotoURL:     domainCompanion.PhotoURL,
	}
}
