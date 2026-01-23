package schema

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// Interaction represents the database schema for interactions table
// Note: This table uses declarative partitioning by created_at
type Interaction struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey"`
	UserID    *uuid.UUID `gorm:"type:uuid;index:idx_interactions_user"`
	SessionID string     `gorm:"type:varchar(64);not null;index:idx_interactions_session"`

	// Core Data
	InteractionType string     `gorm:"type:varchar(32);not null"`
	EntityType      string     `gorm:"type:varchar(32);not null;index:idx_interactions_lookup;index:idx_interactions_entity_time,priority:1"`
	EntityID        *uuid.UUID `gorm:"type:uuid;index:idx_interactions_lookup;index:idx_interactions_entity_time,priority:2"`

	// Context - JSONB for flexible metadata
	Context datatypes.JSON `gorm:"type:jsonb;default:'{}'"`

	// Metadata
	DeviceType string `gorm:"type:varchar(16)"`
	Platform   string `gorm:"type:varchar(16)"`
	IPHash     string `gorm:"type:varchar(64)"` // Anonymized
	UserAgent  string `gorm:"type:text"`
	Referrer   string `gorm:"type:text"`

	// Technical
	IsBot     bool      `gorm:"default:false"`
	CreatedAt time.Time `gorm:"not null;index:idx_interactions_created;primaryKey;index:idx_interactions_entity_time,priority:3"` // Part of composite primary key for partitioning
}

// TableName returns the table name for GORM
func (Interaction) TableName() string {
	return "interactions"
}

// InteractionAggregate represents the database schema for interaction_aggregates table
type InteractionAggregate struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	EntityType  string    `gorm:"type:varchar(32);not null;uniqueIndex:idx_aggregate_unique"`
	EntityID    uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_aggregate_unique"`
	PeriodType  string    `gorm:"type:varchar(10);not null;uniqueIndex:idx_aggregate_unique"`
	PeriodStart time.Time `gorm:"not null;uniqueIndex:idx_aggregate_unique"`

	// Metrics
	ViewsTotal      int64 `gorm:"default:0"`
	ViewsUnique     int64 `gorm:"default:0"`
	SavesTotal      int64 `gorm:"default:0"`
	UnsavesTotal    int64 `gorm:"default:0"`
	SharesTotal     int64 `gorm:"default:0"`
	ContactsTotal   int64 `gorm:"default:0"`
	BookingRequests int64 `gorm:"default:0"`

	// Derived Metrics
	AvgTimeOnPageSec int     `gorm:"default:0"`
	EngagementScore  float64 `gorm:"default:0"`

	CreatedAt time.Time `gorm:"not null"`
	UpdatedAt time.Time `gorm:"not null"`
}

// TableName returns the table name for GORM
func (InteractionAggregate) TableName() string {
	return "interaction_aggregates"
}
