package schema

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MediaType string

const (
	MediaTypeImage    MediaType = "image"
	MediaTypeVideo    MediaType = "video"
	MediaTypeTour360  MediaType = "360_tour"
	MediaTypeDocument MediaType = "document"
)

type Thumbnail struct {
	Key      string `json:"key"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	Size     int64  `json:"size_bytes"`
	MimeType string `json:"mime_type"`
}

type ThumbnailMap map[string]Thumbnail

func (tm ThumbnailMap) Value() (driver.Value, error) {
	return json.Marshal(tm)
}

func (tm *ThumbnailMap) Scan(value interface{}) error {
	return json.Unmarshal(value.([]byte), tm)
}

type ListingMedia struct {
	ID        uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	ListingID uuid.UUID `gorm:"type:uuid;not null;index"`

	Key  string    `gorm:"not null"`
	Type MediaType `gorm:"size:20;default:'image'"`

	Thumbnails ThumbnailMap `gorm:"type:jsonb;serializer:json"`

	Group    *string `gorm:"type:varchar(100);index"`
	Caption  *string `gorm:"size:100"`
	MimeType *string `gorm:"size:50"`

	SizeBytes int64 `gorm:"default:0"`
	Duration  *int  `gorm:"default:null"`

	IsPrimary    bool `gorm:"default:false;index"`
	IsGroupCover bool `gorm:"default:false"`
	Order        int  `gorm:"default:0;index"`

	ImageEmbedding       *VectorEmbedding `gorm:"type:vector(512)"`
	EmbeddingModel       *string          `gorm:"type:varchar(100)"`
	EmbeddingVersion     *string          `gorm:"type:varchar(50)"`
	EmbeddingGeneratedAt *time.Time

	// Moderation tracking
	LastModeratedAt *time.Time `gorm:"index"` // Tracks when this media was last moderated

	UrlGeneratedAt time.Time `gorm:"index"`
	Uploaded       bool      `gorm:"default:false;index"`
	UploadedAt     time.Time `gorm:"index"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (m *ListingMedia) BeforeSave(tx *gorm.DB) error {
	if tx != nil && tx.Statement != nil {
		rv := tx.Statement.ReflectValue
		if rv.IsValid() {
			for rv.Kind() == reflect.Pointer && !rv.IsNil() {
				rv = rv.Elem()
			}
			if rv.IsValid() && rv.Kind() == reflect.Struct {
				if !tx.Statement.Changed("Type") {
					return nil
				}
			}
		}
	}

	switch m.Type {
	case MediaTypeImage, MediaTypeVideo, MediaTypeTour360, MediaTypeDocument:
		return nil
	default:
		return fmt.Errorf("invalid media type: %s", m.Type)
	}
}
