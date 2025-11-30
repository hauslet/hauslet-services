package graph

import (
	"hauslet/internal/profile/domain"

	"github.com/google/uuid"
)

const (
	defaultMaxListLimit = 50
	defaultListFallback = 20

	defaultBadgesFanOut = 5
	defaultTravelFanOut = 4
)

// clampLimit normalizes the requested limit into a safe, bounded value.
func clampLimit(limit *int, max int, fallback int) int {
	effective := fallback
	if limit != nil {
		effective = *limit
	}
	if effective <= 0 {
		effective = fallback
	}
	if effective > max {
		effective = max
	}
	return effective
}

// NewComplexityRoot sets custom complexity costs to guard expensive list fields.
func NewComplexityRoot(maxList int) ComplexityRoot {
	if maxList <= 0 {
		maxList = defaultMaxListLimit
	}
	fallback := defaultListFallback
	if fallback > maxList {
		fallback = maxList
	}
	clamp := func(limit *int) int {
		return clampLimit(limit, maxList, fallback)
	}

	return ComplexityRoot{
		Query: struct {
			Me                 func(childComplexity int) int
			MyProfile          func(childComplexity int) int
			Profile            func(childComplexity int, id uuid.UUID) int
			ProfileByUserID    func(childComplexity int, userID string) int
			Profiles           func(childComplexity int, limit *int, offset *int) int
			ProfilesByUserType func(childComplexity int, userType domain.UserType, limit *int, offset *int) int
			SearchProfiles     func(childComplexity int, query string, limit *int, offset *int) int
			VerifiedProfiles   func(childComplexity int, level *string, limit *int, offset *int) int
			Version            func(childComplexity int) int
		}{
			MyProfile: func(childComplexity int) int {
				return 1 + childComplexity
			},
			Profiles: func(childComplexity int, limit *int, _ *int) int {
				size := clamp(limit)
				return 1 + childComplexity*size
			},
			SearchProfiles: func(childComplexity int, _ string, limit *int, _ *int) int {
				size := clamp(limit)
				return 1 + childComplexity*size
			},
			ProfilesByUserType: func(childComplexity int, _ domain.UserType, limit *int, _ *int) int {
				size := clamp(limit)
				return 1 + childComplexity*size
			},
			VerifiedProfiles: func(childComplexity int, _ *string, limit *int, _ *int) int {
				size := clamp(limit)
				return 1 + childComplexity*size
			},
		},
		Profile: struct {
			Address             func(childComplexity int) int
			Badges              func(childComplexity int) int
			Bio                 func(childComplexity int) int
			BirthDate           func(childComplexity int) int
			City                func(childComplexity int) int
			CommunityCommitment func(childComplexity int) int
			Country             func(childComplexity int) int
			CreatedAt           func(childComplexity int) int
			Education           func(childComplexity int) int
			FullName            func(childComplexity int) int
			FunFact             func(childComplexity int) int
			Hobbies             func(childComplexity int) int
			ID                  func(childComplexity int) int
			IDVerified          func(childComplexity int) int
			Interests           func(childComplexity int) int
			Languages           func(childComplexity int) int
			ObsessedWith        func(childComplexity int) int
			Occupation          func(childComplexity int) int
			PhoneNumbers        func(childComplexity int) int
			PhoneVerified       func(childComplexity int) int
			Rating              func(childComplexity int) int
			ReviewsCount        func(childComplexity int) int
			Skills              func(childComplexity int) int
			State               func(childComplexity int) int
			TravelCompanions    func(childComplexity int) int
			TrustScore          func(childComplexity int) int
			UpdatedAt           func(childComplexity int) int
			UserID              func(childComplexity int) int
			UserTypes           func(childComplexity int) int
			VerificationDate    func(childComplexity int) int
			VerificationLevel   func(childComplexity int) int
			ZipCode             func(childComplexity int) int
		}{
			Badges: func(childComplexity int) int {
				return 1 + childComplexity*defaultBadgesFanOut
			},
			TravelCompanions: func(childComplexity int) int {
				return 1 + childComplexity*defaultTravelFanOut
			},
		},
	}
}
