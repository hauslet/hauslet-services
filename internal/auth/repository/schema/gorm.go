package schema

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserRole string

const (
	RoleUser      UserRole = "user"
	RoleStaff     UserRole = "staff"
	RoleModerator UserRole = "moderator"
	RoleSupport   UserRole = "support"
	RoleAdmin     UserRole = "admin"
	RoleRoot      UserRole = "root"
)

type User struct {
	ID uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`

	Name string `gorm:"column:name;not null"`

	// Canonical email for the user (for notifications, UI, etc.).
	// Initially set from the first OAuth provider, but can be updated by user.
	PrimaryEmail string `gorm:"column:primary_email;not null;uniqueIndex"`

	Role     UserRole `gorm:"column:role;type:varchar(50);not null;default:'user'"`
	IsActive bool     `gorm:"column:is_active;default:true"`
	// Deactivation metadata (used for user-initiated deactivation or admin suspension)
	DeactivatedAt     *time.Time     `gorm:"column:deactivated_at"`
	DeactivatedReason *string        `gorm:"column:deactivated_reason;type:text"`
	DeactivatedBy     *uuid.UUID     `gorm:"column:deactivated_by;type:uuid"`
	ReactivateOnLogin bool           `gorm:"column:reactivate_on_login;default:false"`
	ReactivatedAt     *time.Time     `gorm:"column:reactivated_at"`
	ReactivatedBy     *uuid.UUID     `gorm:"column:reactivated_by;type:uuid"`
	Identities        []UserIdentity `gorm:"foreignKey:UserID"`

	LastLoginAt *time.Time `gorm:"column:last_login_at"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

type UserIdentity struct {
	ID     uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	UserID uuid.UUID `gorm:"type:uuid;uniqueIndex:idx_user_provider;not null"`
	User   User      `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`

	// OAuth provider name (e.g., "google", "github", "password")
	Provider string `gorm:"column:provider;uniqueIndex:idx_user_provider;uniqueIndex:idx_provider_identity;not null"`
	// Provider's unique ID for this user
	ProviderID string `gorm:"column:provider_id;uniqueIndex:idx_provider_identity;not null"`
	// Email from the OAuth provider (immutable record of what provider gave us)
	Email string `gorm:"column:email;not null;index"`

	EmailVerified bool    `gorm:"column:email_verified;default:false"`
	PasswordHash  *string `gorm:"column:password_hash"` // Only for "password" provider

	LastUsedAt *time.Time `gorm:"column:last_used_at"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
	DeletedAt  gorm.DeletedAt `gorm:"index"`
}
