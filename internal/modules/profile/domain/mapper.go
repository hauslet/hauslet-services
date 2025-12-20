package domain

import (
	"errors"

	moderationservice "hauslet/internal/modules/moderation/service"
	"hauslet/internal/modules/profile/repository/schema"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type ContentType string
type ModerationStatus string

const (
	ContentTypeProfileBio      ContentType = "profile_bio"
	ContentTypeProfileImage    ContentType = "profile_image"
	ContentTypeTravelCompImage ContentType = "tc_image"
)

const (
	ModerationStatusPending   ModerationStatus = "pending"
	ModerationStatusAccepted  ModerationStatus = "accepted"
	ModerationStatusEscalated ModerationStatus = "escalated"
	ModerationStatusRejected  ModerationStatus = "rejected"
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
		Email:                      schemaProfile.Email,
		BirthDate:                  schemaProfile.BirthDate,
		Gender:                     Gender(schemaProfile.Gender),
		PhotoURL:                   schemaProfile.PhotoURL,
		PhoneNumbers:               make([]string, len(schemaProfile.PhoneNumbers)),
		Address:                    schemaProfile.Address.Street,
		City:                       schemaProfile.Address.City,
		State:                      schemaProfile.Address.State,
		Country:                    schemaProfile.Address.Country,
		IsModerated:                schemaProfile.IsModerated,
		ZipCode:                    schemaProfile.Address.PostalCode,
		HouseNumber:                schemaProfile.Address.HouseNumber,
		Street:                     schemaProfile.Address.Street,
		Area:                       schemaProfile.Address.Area,
		LGA:                        schemaProfile.Address.LGA,
		District:                   schemaProfile.Address.District,
		DigitalAddress:             schemaProfile.Address.DigitalAddress,
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

	gender := domainProfile.Gender
	if gender == "" {
		gender = GenderUndisclosed
	}

	streetVal := domainProfile.Street
	if streetVal == nil {
		streetVal = domainProfile.Address
	}

	schemaProfile := &schema.Profile{
		ID:           domainProfile.ID,
		UserID:       userUUID,
		UserTypes:    make(pq.StringArray, len(domainProfile.UserTypes)),
		FullName:     domainProfile.FullName,
		Email:        domainProfile.Email,
		BirthDate:    domainProfile.BirthDate,
		Gender:       schema.Gender(gender),
		PhotoURL:     domainProfile.PhotoURL,
		PhoneNumbers: make(pq.StringArray, len(domainProfile.PhoneNumbers)),
		Address: schema.Address{
			HouseNumber:    domainProfile.HouseNumber,
			Street:         streetVal,
			Area:           domainProfile.Area,
			LGA:            domainProfile.LGA,
			City:           domainProfile.City,
			State:          domainProfile.State,
			Country:        domainProfile.Country,
			PostalCode:     domainProfile.ZipCode,
			District:       domainProfile.District,
			DigitalAddress: domainProfile.DigitalAddress,
		},
		Occupation:                 domainProfile.Occupation,
		Education:                  domainProfile.Education,
		Bio:                        domainProfile.Bio,
		Skills:                     pq.StringArray(domainProfile.Skills),
		Languages:                  pq.StringArray(domainProfile.Languages),
		Interests:                  pq.StringArray(domainProfile.Interests),
		Hobbies:                    pq.StringArray(domainProfile.Hobbies),
		FunFact:                    domainProfile.FunFact,
		ObsessedWith:               domainProfile.ObsessedWith,
		IsModerated:                domainProfile.IsModerated,
		CommunityCommitment:        domainProfile.CommunityCommitment,
		TravelCompanions:           make([]schema.TravelCompanion, len(domainProfile.TravelCompanions)),
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
func MapTravelCompanionFromSchema(schemaCompanion schema.TravelCompanion) TravelCompanion {
	gender := Gender(schemaCompanion.Gender)
	if gender == "" {
		gender = GenderUndisclosed
	}

	return TravelCompanion{
		ID:           schemaCompanion.ID,
		Name:         schemaCompanion.Name,
		AgeGroup:     AgeGroup(schemaCompanion.AgeGroup),
		Gender:       gender,
		Phone:        schemaCompanion.Phone,
		Relationship: Relationship(schemaCompanion.Relationship),
		PhotoURL:     schemaCompanion.PhotoURL,
	}
}

// MapTravelCompanionToSchema converts domain travel companion to schema.
func MapTravelCompanionToSchema(domainCompanion TravelCompanion) schema.TravelCompanion {
	gender := domainCompanion.Gender
	if gender == "" {
		gender = GenderUndisclosed
	}

	return schema.TravelCompanion{
		ID:           domainCompanion.ID,
		Name:         domainCompanion.Name,
		AgeGroup:     schema.AgeGroup(domainCompanion.AgeGroup),
		Gender:       schema.Gender(gender),
		Phone:        domainCompanion.Phone,
		Relationship: schema.Relationship(domainCompanion.Relationship),
		PhotoURL:     domainCompanion.PhotoURL,
	}
}

type AggregatedModeration struct {
	TargetID uuid.UUID

	Pending   int64
	Accepted  int64
	Rejected  int64
	Escalated int64

	ContentTypes []ContentType
	Reasons      []string
}

func MapModerationAggToDomain(agg moderationservice.AggregatedModeration) *AggregatedModeration {
	contentTypes := make([]ContentType, len(agg.ContentTypes))
	for i, ct := range agg.ContentTypes {
		contentTypes[i] = ContentType(ct)
	}

	reasons := make([]string, len(agg.Reasons))
	copy(reasons, agg.Reasons)

	return &AggregatedModeration{
		TargetID:     agg.TargetID,
		Pending:      agg.Pending,
		Accepted:     agg.Accepted,
		Rejected:     agg.Rejected,
		Escalated:    agg.Escalated,
		ContentTypes: contentTypes,
		Reasons:      reasons,
	}
}

// FinalStatus returns the terminal moderation status for the aggregate.
func (a AggregatedModeration) FinalStatus() ModerationStatus {
	if a.Pending > 0 {
		return ModerationStatusPending
	}
	if a.Rejected > 0 {
		return ModerationStatusRejected
	}
	if a.Escalated > 0 {
		return ModerationStatusEscalated
	}
	if a.Accepted > 0 {
		return ModerationStatusAccepted
	}
	return ModerationStatusPending
}
