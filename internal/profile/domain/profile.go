package domain

import (
	"time"

	"github.com/google/uuid"
)

// TravelCompanion represents a travel companion in the domain model
type TravelCompanion struct {
	Name         string
	AgeGroup     string // e.g., "child", "teen", "adult", "senior"
	Phone        *string
	Relationship string // e.g., "family", "friend", "colleague"
	PhotoURL     *string
}

// Profile represents the core domain model for user profiles
type Profile struct {
	ID        uuid.UUID
	UserID    string // Stored as string for compatibility with upstream auth identifiers
	UserTypes []UserType
	FullName  string
	BirthDate *time.Time

	// Contact Information
	PhoneNumbers []string
	Address      *string
	City         *string
	State        *string
	Country      *string
	ZipCode      *string

	// Personal Information
	Occupation   *string
	Education    *string
	Bio          *string
	Skills       []string
	Languages    []string
	Interests    []string
	Hobbies      []string
	FunFact      *string
	ObsessedWith *string

	// Community Settings
	CommunityCommitment bool
	TravelCompanions    []TravelCompanion

	// Verification Information
	PhoneVerified     bool
	IDVerified        bool
	VerificationDate  *time.Time
	VerificationLevel string // e.g., "basic", "identity", "trusted"

	// Profile Features
	Rating       float64
	ReviewsCount int
	Badges       []Badge
	TrustScore   float64

	// Metadata
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

// UserType represents the type of user in the system
type UserType string

const (
	Host     UserType = "host"
	Guest    UserType = "guest"
	Agent    UserType = "agent"
	LandLord UserType = "landlord"
	CoHost   UserType = "cohost"
)

// UpdateProfileInput contains the data that can be updated in a profile
type UpdateProfileInput struct {
	PhoneNumbers     []*string
	Address          *string
	City             *string
	State            *string
	Country          *string
	ZipCode          *string
	Occupation       *string
	Education        *string
	Bio              *string
	Skills           []string
	Languages        []string
	Interests        []string
	Hobbies          []string
	FunFact          *string
	ObsessedWith     *string
	CommunityComment *bool
	TravelCompanions []TravelCompanion
}

// IsFullyVerified checks if the profile has all verification types completed
func (p *Profile) IsFullyVerified() bool {
	return p.PhoneVerified && p.IDVerified
}

// CanAddPhoneNumber checks if a phone number can be added (max 2 allowed)
func (p *Profile) CanAddPhoneNumber() bool {
	return len(p.PhoneNumbers) < 2
}

// HasUserType checks if the profile has a specific user type
func (p *Profile) HasUserType(userType UserType) bool {
	for _, t := range p.UserTypes {
		if t == userType {
			return true
		}
	}
	return false
}

// HasBadge checks if the profile already has the provided badge.
func (p *Profile) HasBadge(badge Badge) bool {
	for _, b := range p.Badges {
		if b == badge {
			return true
		}
	}
	return false
}

// CalculateTrustScore calculates the trust score based on verification and activity
func (p *Profile) CalculateTrustScore() float64 {
	score := 0.0

	// Verification contributes to trust score
	if p.PhoneVerified {
		score += 0.3
	}
	if p.IDVerified {
		score += 0.4
	}

	// Rating contributes to trust score (max 0.2)
	if p.ReviewsCount > 0 {
		score += (p.Rating / 5.0) * 0.2
	}

	// Badges contribute to trust score (0.02 per badge, max 0.1)
	badgeScore := float64(len(p.Badges)) * 0.02
	if badgeScore > 0.1 {
		badgeScore = 0.1
	}
	score += badgeScore

	return score
}
