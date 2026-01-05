package schema

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UserPreferences stores user preferences for personalization
type UserPreferences struct {
	ID                 uuid.UUID                `gorm:"type:uuid;primaryKey"`
	UserID             uuid.UUID                `gorm:"type:uuid;uniqueIndex;not null"`
	PreferredLocations []string                 `gorm:"type:text[]"`
	PreferredTypes     []string                 `gorm:"type:text[]"`
	PriceRange         map[string]interface{}   `gorm:"type:jsonb;serializer:json"`
	BedroomRange       map[string]interface{}   `gorm:"type:jsonb;serializer:json"`
	SavedFilters       []map[string]interface{} `gorm:"type:jsonb;serializer:json"`
	CreatedAt          time.Time                `gorm:"not null"`
	UpdatedAt          time.Time
}

// TableName specifies the table name for UserPreferences
func (UserPreferences) TableName() string {
	return "discovery_user_preferences"
}

// BeforeCreate hook to generate UUID before creating
func (u *UserPreferences) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}
