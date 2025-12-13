package domain

import (
	"time"

	"github.com/google/uuid"
)

// Address represents a physical address (matching Profile structure)
type Address struct {
	HouseNumber    *string
	Street         *string
	Area           *string
	LGA            *string
	City           *string
	State          *string
	Country        *string
	PostalCode     *string
	District       *string
	DigitalAddress *string
}

// Location represents a geographic coordinate in WGS84 (default SRID 4326)
type Location struct {
	Lat  float64
	Lng  float64
	SRID int
}

// Valid reports whether the coordinates are within valid ranges
func (l *Location) Valid() bool {
	if l == nil {
		return false
	}
	return l.Lat >= -90 && l.Lat <= 90 && l.Lng >= -180 && l.Lng <= 180
}

// Business represents a business entity in the domain model
type Business struct {
	ID          uuid.UUID
	Slug        string
	Name        string
	DisplayName string
	Description *string

	// Business Type
	BusinessType BusinessType

	// Legal Information
	RegistrationNumber *string
	TaxID              *string
	LegalEntityType    *string

	// Contact
	Email        string
	PhoneNumbers []string
	Website      *string

	// Address (embedded structure matching Profile)
	Address  Address
	Location *Location

	// Branding
	LogoURL       *string
	CoverImageURL *string
	BrandColor    *string

	// Settings
	IsVerified bool
	VerifiedAt *time.Time
	IsActive   bool

	// Billing (for future use)
	BillingEmail *string

	// Ownership
	CreatedBy uuid.UUID

	// Metadata
	MemberCount   int
	PropertyCount int
	ListingCount  int

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

// CreateBusinessInput represents input for creating a business
type CreateBusinessInput struct {
	Name               string
	DisplayName        string
	Description        *string
	BusinessType       BusinessType
	RegistrationNumber *string
	TaxID              *string
	LegalEntityType    *string
	Email              string
	PhoneNumbers       []string
	Website            *string
	Address            Address
	Location           *Location
	LogoURL            *string
	CoverImageURL      *string
	BrandColor         *string
	BillingEmail       *string
}

// UpdateBusinessInput represents input for updating a business
type UpdateBusinessInput struct {
	DisplayName   *string
	Description   *string
	Email         *string
	PhoneNumbers  []string
	Website       *string
	Address       *Address
	Location      *Location
	LogoURL       *string
	CoverImageURL *string
	BrandColor    *string
	BillingEmail  *string
}

// IsOwner checks if the given user is an owner of the business
func (b *Business) IsOwner(userID uuid.UUID) bool {
	return b.CreatedBy == userID
}

// CanBeDeleted checks if the business can be deleted
func (b *Business) CanBeDeleted() bool {
	return b.PropertyCount == 0 && b.ListingCount == 0
}

// HasLocation checks if the business has location coordinates
func (b *Business) HasLocation() bool {
	return b.Location != nil && b.Location.Valid()
}
