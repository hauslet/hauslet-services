package schema

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

// TravelCompanionProfile needs JSON tags because it lives inside a JSONB column
type TravelCompanionProfile struct {
	Name         string  `json:"name"`
	AgeGroup     string  `json:"age_group"`
	Phone        *string `json:"phone,omitempty"`
	Relationship string  `json:"relationship"`
	PhotoURL     *string `json:"photo_url,omitempty"`
}

type Profile struct {
	// Primary Key
	ID        uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	FullName  string     `gorm:"not null;index"`
	BirthDate *time.Time `gorm:"check:birth_date <= now() - interval '18 years'"`

	// Foreign Key to Auth Module
	UserID    uuid.UUID      `gorm:"type:uuid;not null;uniqueIndex"`
	UserTypes pq.StringArray `gorm:"type:text[];not null;default:'{guest}'"`

	// Contact Information
	PhoneNumbers pq.StringArray `gorm:"type:text[];size:2"`
	Address      *string
	City         *string
	State        *string
	Country      *string
	ZipCode      *string

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

	TravelCompanions []TravelCompanionProfile `gorm:"type:jsonb;serializer:json"`

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
