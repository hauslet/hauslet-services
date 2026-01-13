package schema

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// --- PROPERTY (The Physical Asset) ---
type Property struct {
	ID       uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	PublicID string    `gorm:"type:varchar(8);uniqueIndex;not null"`

	// Basic info
	UnitNumber string `gorm:"type:varchar(50)"`
	Address    string `gorm:"type:varchar(500);not null"`
	City       string `gorm:"type:varchar(100);index"`
	State      string `gorm:"type:varchar(100);index"`
	PostalCode string `gorm:"type:varchar(20)"`

	// Enums
	Country CountryCode `gorm:"type:varchar(50);default:'NG'"`

	// PostGIS geospatial location
	Location *GeographyPoint `gorm:"type:geography(Point,4326);index:idx_properties_location_gist,type:gist" json:"location,omitempty"`

	// Classification
	PropertyClass     PropertyClass     `gorm:"type:varchar(50);default:'residential';index"`
	PropertyType      PropertyType      `gorm:"type:varchar(50);default:'apartment';index"`
	FurnishingType    FurnishingType    `gorm:"type:varchar(50);default:'furnished'"`
	PropertyCondition PropertyCondition `gorm:"type:varchar(50);default:'used'"`

	// Details
	Bedrooms      *int `gorm:"type:int"`
	Bathrooms     *int `gorm:"type:int"`
	Toilets       *int `gorm:"type:int"`
	HalfBathrooms *int `gorm:"type:int"`
	Floors        *int `gorm:"type:int"`
	Units         int  `gorm:"type:int;default:1"`

	OwnerID uuid.UUID `gorm:"type:uuid;not null;index;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`

	SquareMeters float64  `gorm:"type:float;default:0"`
	FloorArea    *float64 `gorm:"type:float"`

	// JSON fields
	Amenities          []AmenitiesType `gorm:"type:jsonb;serializer:json"`
	FeaturesCommercial []AmenitiesType `gorm:"type:jsonb;serializer:json"`

	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

// Hook to validate Property Enums
func (p *Property) BeforeSave(tx *gorm.DB) error {
	for _, a := range p.Amenities {
		for _, item := range a.Items {
			if !IsValidAmenity(item) {
				return fmt.Errorf("invalid amenity: %s", item)
			}
		}
	}
	return nil
}

// BeforeCreate hook to generate PublicID
func (p *Property) BeforeCreate(tx *gorm.DB) error {
	if p.PublicID == "" {
		for range 10 {
			publicID, err := generateRandomString("H", 8)
			if err != nil {
				return fmt.Errorf("failed to generate public ID: %w", err)
			}

			var count int64
			if err := tx.Model(&Property{}).Where("public_id = ?", publicID).Count(&count).Error; err != nil {
				return fmt.Errorf("check public ID: %w", err)
			}

			if count == 0 {
				p.PublicID = publicID
				return nil
			}
		}
		return fmt.Errorf("failed to generate unique public ID")
	}
	return nil
}
