package schema

import (
	"errors"
	"strings"
	"time"

	propertySchema "hauslet/internal/modules/property/repository/schema"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

// --- Enums ---

type BusinessType string

const (
	TypePropertyManagement BusinessType = "property_management"
	TypeRealEstateAgency   BusinessType = "real_estate_agency"
	TypePropertyDeveloper  BusinessType = "property_developer"
	TypeInvestmentFirm     BusinessType = "investment_firm"
	TypeIndividual         BusinessType = "individual"
)

type MemberRole string

const (
	RoleOwner      MemberRole = "owner"
	RoleAdmin      MemberRole = "admin"
	RoleMember     MemberRole = "member"
	RoleViewer     MemberRole = "viewer"
	RoleAccountant MemberRole = "accountant"
)

type InvitationStatus string

const (
	InvitationPending  InvitationStatus = "pending"
	InvitationAccepted InvitationStatus = "accepted"
	InvitationDeclined InvitationStatus = "declined"
	InvitationExpired  InvitationStatus = "expired"
	InvitationRevoked  InvitationStatus = "revoked"
)

// --- Validation Helpers ---

func (bt BusinessType) IsValid() bool {
	switch bt {
	case TypePropertyManagement, TypeRealEstateAgency, TypePropertyDeveloper, TypeInvestmentFirm, TypeIndividual:
		return true
	}
	return false
}

func (mr MemberRole) IsValid() bool {
	switch mr {
	case RoleOwner, RoleAdmin, RoleMember, RoleViewer, RoleAccountant:
		return true
	}
	return false
}

func (is InvitationStatus) IsValid() bool {
	switch is {
	case InvitationPending, InvitationAccepted, InvitationDeclined, InvitationExpired, InvitationRevoked:
		return true
	}
	return false
}

// --- Structs ---

// Address mirrors the Profile Address structure
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

// Business represents a business entity in the database
type Business struct {
	ID          uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Slug        string    `gorm:"uniqueIndex;not null"`
	Name        string    `gorm:"not null;index"`
	DisplayName string    `gorm:"not null"`
	Description *string   `gorm:"type:text"`

	BusinessType BusinessType `gorm:"size:50;not null"`

	// Legal Information
	RegistrationNumber *string
	TaxID              *string
	LegalEntityType    *string

	// Contact
	Email        string         `gorm:"not null"`
	PhoneNumbers pq.StringArray `gorm:"type:text[]"`
	Website      *string

	// Embedded Address (Flattens columns to addr_street, addr_city, etc.)
	// Matches Profile structure exactly
	Address  Address                        `gorm:"embedded;embeddedPrefix:addr_"`
	Location *propertySchema.GeographyPoint `gorm:"type:geography(Point,4326)"`

	// Branding
	LogoURL       *string
	CoverImageURL *string
	BrandColor    *string

	// Settings
	IsVerified bool       `gorm:"default:false"`
	VerifiedAt *time.Time
	IsActive   bool `gorm:"default:true"`

	// Billing (for future use)
	BillingEmail *string

	// Ownership
	CreatedBy uuid.UUID `gorm:"type:uuid;not null;index"`

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (Business) TableName() string {
	return "businesses"
}

// BeforeSave hook to validate Business before saving
func (b *Business) BeforeSave(tx *gorm.DB) error {
	// Normalize strings
	b.Slug = strings.ToLower(strings.TrimSpace(b.Slug))
	b.Name = strings.TrimSpace(b.Name)
	b.DisplayName = strings.TrimSpace(b.DisplayName)
	b.Email = strings.ToLower(strings.TrimSpace(b.Email))

	// Validate business type
	if !b.BusinessType.IsValid() {
		return errors.New("invalid business type")
	}

	// Validate slug format (alphanumeric and hyphens only)
	if !isValidSlug(b.Slug) {
		return errors.New("invalid slug format: must contain only lowercase letters, numbers, and hyphens")
	}

	return nil
}

// BusinessMember represents a member of a business
type BusinessMember struct {
	ID         uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	BusinessID uuid.UUID `gorm:"type:uuid;not null;index;uniqueIndex:idx_business_user"`
	UserID     uuid.UUID `gorm:"type:uuid;not null;index;uniqueIndex:idx_business_user"`

	Role MemberRole `gorm:"size:50;not null"`

	// JSONB for flexible permissions
	Permissions map[string]bool `gorm:"type:jsonb;serializer:json"`

	IsActive  bool       `gorm:"default:true"`
	InvitedBy *uuid.UUID `gorm:"type:uuid"`
	JoinedAt  time.Time  `gorm:"not null"`

	CreatedAt time.Time
	UpdatedAt time.Time

	// Relationships
	Business Business `gorm:"foreignKey:BusinessID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

func (BusinessMember) TableName() string {
	return "business_members"
}

// BeforeSave hook to validate BusinessMember before saving
func (bm *BusinessMember) BeforeSave(tx *gorm.DB) error {
	// Validate role
	if !bm.Role.IsValid() {
		return errors.New("invalid member role")
	}

	// Ensure JoinedAt is set
	if bm.JoinedAt.IsZero() {
		bm.JoinedAt = time.Now()
	}

	return nil
}

// BusinessInvitation represents an invitation to join a business
type BusinessInvitation struct {
	ID         uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	BusinessID uuid.UUID `gorm:"type:uuid;not null;index"`

	Email  string     `gorm:"not null;index"`
	UserID *uuid.UUID `gorm:"type:uuid;index"`

	Role        MemberRole      `gorm:"size:50;not null"`
	Permissions map[string]bool `gorm:"type:jsonb;serializer:json"`

	InvitedBy  uuid.UUID        `gorm:"type:uuid;not null"`
	Token      string           `gorm:"uniqueIndex;not null"`
	Status     InvitationStatus `gorm:"size:20;default:'pending';index"`

	ExpiresAt  time.Time  `gorm:"not null;index"`
	AcceptedAt *time.Time

	CreatedAt time.Time
	UpdatedAt time.Time

	// Relationships
	Business Business `gorm:"foreignKey:BusinessID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

func (BusinessInvitation) TableName() string {
	return "business_invitations"
}

// BeforeSave hook to validate BusinessInvitation before saving
func (bi *BusinessInvitation) BeforeSave(tx *gorm.DB) error {
	// Normalize email
	bi.Email = strings.ToLower(strings.TrimSpace(bi.Email))

	// Validate role
	if !bi.Role.IsValid() {
		return errors.New("invalid member role for invitation")
	}

	// Validate status
	if !bi.Status.IsValid() {
		return errors.New("invalid invitation status")
	}

	// Ensure token is set
	if bi.Token == "" {
		return errors.New("invitation token cannot be empty")
	}

	// Validate email format (basic check)
	if !strings.Contains(bi.Email, "@") {
		return errors.New("invalid email format")
	}

	return nil
}

// --- Helper Functions ---

// isValidSlug checks if a slug contains only lowercase letters, numbers, and hyphens
func isValidSlug(slug string) bool {
	if slug == "" {
		return false
	}

	for _, char := range slug {
		if !((char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '-') {
			return false
		}
	}

	// Ensure it doesn't start or end with a hyphen
	return !strings.HasPrefix(slug, "-") && !strings.HasSuffix(slug, "-")
}
