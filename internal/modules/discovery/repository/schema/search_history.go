package schema

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SearchHistory stores user search queries for analytics and personalization
type SearchHistory struct {
	ID              uuid.UUID              `gorm:"type:uuid;primaryKey"`
	UserID          uuid.UUID              `gorm:"type:uuid;not null;index:idx_search_history_user_created"`
	Query           string                 `gorm:"type:text"`
	Filters         map[string]interface{} `gorm:"type:jsonb;serializer:json"`
	ResultCount     int                    `gorm:"not null"`
	ClickedListings []uuid.UUID            `gorm:"type:uuid[]"`
	CreatedAt       time.Time              `gorm:"not null;index:idx_search_history_user_created"`
	UpdatedAt       time.Time
}

// TableName specifies the table name for SearchHistory
func (SearchHistory) TableName() string {
	return "discovery_search_history"
}

// BeforeCreate hook to generate UUID before creating
func (s *SearchHistory) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}
