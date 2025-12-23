package schema

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// -- Validation Errors --
var (
	ErrWishlistNameEmpty   = errors.New("wishlist name cannot be empty")
	ErrWishlistNameTooLong = errors.New("wishlist name cannot exceed 255 characters")
	ErrUserIDRequired      = errors.New("user ID is required for wishlist")
	ErrInvalidSource       = errors.New("invalid wishlist item source")
	ErrMissingIDs          = errors.New("wishlist item requires both WishlistID and ListingID")
)

type WishlistItemSource string

const (
	SourceUserAdded WishlistItemSource = "user_added"
	SourceImported  WishlistItemSource = "imported"
	SourceRecents   WishlistItemSource = "recents"
)

type WishlistItem struct {
	ID         uuid.UUID          `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	WishlistID uuid.UUID          `gorm:"type:uuid;not null;uniqueIndex:idx_wishlist_listing"`
	ListingID  uuid.UUID          `gorm:"type:uuid;not null;uniqueIndex:idx_wishlist_listing"`
	AddedAt    time.Time          `gorm:"autoCreateTime;not null"`
	Source     WishlistItemSource `gorm:"type:varchar(50);not null;default:'user_added'"`
}

// BeforeSave runs before Insert AND Update
func (wi *WishlistItem) BeforeSave(tx *gorm.DB) error {
	// 1. Validate Foreign Keys exist (not nil)
	if wi.WishlistID == uuid.Nil || wi.ListingID == uuid.Nil {
		return ErrMissingIDs
	}

	// 2. Validate Enum Integrity
	switch wi.Source {
	case SourceUserAdded, SourceImported, SourceRecents:
		// Valid
	default:
		return ErrInvalidSource
	}

	return nil
}

type Wishlist struct {
	ID          uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	UserID      uuid.UUID      `gorm:"type:uuid;not null;index"`
	Name        string         `gorm:"type:varchar(255);not null"`
	Description *string        `gorm:"type:text"`
	IsPrivate   bool           `gorm:"default:false"`
	Items       []WishlistItem `gorm:"foreignKey:WishlistID;constraint:OnDelete:CASCADE;"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

// BeforeSave runs before Insert AND Update
func (w *Wishlist) BeforeSave(tx *gorm.DB) error {
	// 1. Sanitize Name (Trim whitespace)
	w.Name = strings.TrimSpace(w.Name)

	// 2. Validate Name Presence
	if w.Name == "" {
		return ErrWishlistNameEmpty
	}

	// 3. Validate Name Length
	if len(w.Name) > 255 {
		return ErrWishlistNameTooLong
	}

	// 4. Validate Owner
	if w.UserID == uuid.Nil {
		return ErrUserIDRequired
	}

	return nil
}
