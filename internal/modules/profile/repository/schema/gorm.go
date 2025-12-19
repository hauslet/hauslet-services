package schema

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

// --- Enums & Constants ---

type AgeGroup string
type Gender string
type Relationship string

const (
	RelationshipFamily    Relationship = "family"
	RelationshipFriend    Relationship = "friend"
	RelationshipColleague Relationship = "colleague"
	RelationshipOther     Relationship = "other"
)

const (
	GenderMale        Gender = "male"
	GenderFemale      Gender = "female"
	GenderOther       Gender = "other"
	GenderUndisclosed Gender = "undisclosed"
)

const (
	AgeGroupChild  AgeGroup = "child"
	AgeGroupTeen   AgeGroup = "teen"
	AgeGroupAdult  AgeGroup = "adult"
	AgeGroupSenior AgeGroup = "senior"
)

// --- Validation Helpers ---

func (g Gender) IsValid() bool {
	switch g {
	case GenderMale, GenderFemale, GenderOther, GenderUndisclosed:
		return true
	}
	return false
}

func (r Relationship) IsValid() bool {
	switch r {
	case RelationshipFamily, RelationshipFriend, RelationshipColleague, RelationshipOther:
		return true
	}
	return false
}

func (a AgeGroup) IsValid() bool {
	switch a {
	case AgeGroupChild, AgeGroupTeen, AgeGroupAdult, AgeGroupSenior:
		return true
	}
	return false
}

// --- Structs ---

type Address struct {
	HouseNumber    *string `json:"house_number,omitempty"`
	Street         *string `json:"street,omitempty"`
	Area           *string `json:"area,omitempty"`
	LGA            *string `json:"lga,omitempty"`
	City           *string `json:"city,omitempty"`
	State          *string `json:"state,omitempty"`
	Country        *string `json:"country,omitempty"`
	PostalCode     *string `json:"postal_code,omitempty"`
	District       *string `json:"district,omitempty"`
	DigitalAddress *string `json:"digital_address,omitempty"`
}

type TravelCompanion struct {
	ID        uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	ProfileID uuid.UUID `gorm:"type:uuid;not null;index" json:"profile_id"` // Foreign Key to Profile
	Name      string    `gorm:"not null" json:"name"`

	// Enforce constraints in DB
	AgeGroup     AgeGroup     `gorm:"not null;check:age_group IN ('child', 'teen', 'adult', 'senior')" json:"age_group"`
	Gender       Gender       `gorm:"not null;default:'undisclosed';check:gender IN ('male', 'female', 'other', 'undisclosed')" json:"gender"`
	Phone        *string      `json:"phone,omitempty"`
	Relationship Relationship `gorm:"not null;check:relationship IN ('family', 'friend', 'colleague', 'other')" json:"relationship"`
	PhotoURL     *string      `json:"photo_url,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Profile struct {
	// Primary Key
	ID        uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	FullName  string     `gorm:"not null;index"`
	BirthDate *time.Time `gorm:"check:birth_date <= now() - interval '18 years'"`
	Gender    Gender     `gorm:"type:text;default:'undisclosed';check:gender IN ('male', 'female', 'other', 'undisclosed')"`
	PhotoURL  *string    `gorm:"type:text"`
	Email     *string    `gorm:"type:text;uniqueIndex"`

	// Foreign Key to Auth Module
	UserID      uuid.UUID      `gorm:"type:uuid;not null;uniqueIndex"`
	UserTypes   pq.StringArray `gorm:"type:text[];not null;default:'{guest}'"`
	IsModerated bool           `gorm:"not null;default:false"`

	// Contact Information
	PhoneNumbers pq.StringArray `gorm:"type:text[];size:2"`

	// Embedded Address (Flattens columns to addr_street, addr_city, etc.)
	Address Address `gorm:"embedded;embeddedPrefix:addr_"`

	// Personal Information
	Occupation   *string
	Education    *string
	Bio          *string
	Skills       pq.StringArray `gorm:"type:text[]"`
	Languages    pq.StringArray `gorm:"type:text[]"`
	Interests    pq.StringArray `gorm:"type:text[]"`
	Hobbies      pq.StringArray `gorm:"type:text[]"`
	FunFact      *string
	ObsessedWith *string

	// Community Settings
	CommunityCommitment bool `gorm:"not null;default:false"`

	// One-to-Many relationship
	TravelCompanions []TravelCompanion `gorm:"foreignKey:ProfileID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"travel_companions"`

	// Verification Information
	PhoneVerified     bool `gorm:"default:false"`
	IDVerified        bool `gorm:"default:false"`
	VerificationDate  *time.Time
	VerificationLevel string

	// Profile Features
	Rating       float64        `gorm:"default:0;index"`
	ReviewsCount int            `gorm:"default:0"`
	Badges       pq.StringArray `gorm:"type:text[]"`
	TrustScore   float64        `gorm:"default:0"`

	// Privacy & Preferences
	BioVisible                 bool `gorm:"not null;default:false"`
	AllowPersonalizedOffers    bool `gorm:"not null;default:false"`
	EnablePerformanceAnalytics bool `gorm:"not null;default:false"`

	// Timestamps
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (Profile) TableName() string {
	return "profiles"
}

// --- Hooks ---

// BeforeSave for TravelCompanion
// Normalizes inputs and ensures Enums are valid before writing to DB
func (tc *TravelCompanion) BeforeSave(tx *gorm.DB) error {
	// 1. Clean strings
	tc.Name = strings.TrimSpace(tc.Name)

	// 2. Handle Gender Defaults & Validation
	tc.Gender = Gender(strings.ToLower(string(tc.Gender)))
	if tc.Gender == "" {
		tc.Gender = GenderUndisclosed
	}
	if !tc.Gender.IsValid() {
		return errors.New("invalid gender for travel companion: must be male, female, other, or undisclosed")
	}

	// 3. Validate Relationship
	tc.Relationship = Relationship(strings.ToLower(string(tc.Relationship)))
	if !tc.Relationship.IsValid() {
		return errors.New("invalid relationship type: must be family, friend, colleague, or other")
	}

	// 4. Validate AgeGroup
	tc.AgeGroup = AgeGroup(strings.ToLower(string(tc.AgeGroup)))
	if !tc.AgeGroup.IsValid() {
		return errors.New("invalid age group: must be child, teen, adult, or senior")
	}

	return nil
}

// BeforeSave for Profile
// Normalizes inputs and ensures Enums are valid before writing to DB
func (p *Profile) BeforeSave(tx *gorm.DB) error {
	// 1. Clean strings
	p.FullName = strings.TrimSpace(p.FullName)

	// 2. Handle Gender Defaults & Validation
	p.Gender = Gender(strings.ToLower(string(p.Gender)))
	if p.Gender == "" {
		p.Gender = GenderUndisclosed
	}
	if !p.Gender.IsValid() {
		return errors.New("invalid gender for profile: must be male, female, other, or undisclosed")
	}

	return nil
}
