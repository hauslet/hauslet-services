package repository

import (
	"context"
	"hauslet/internal/auth/domain"
	"hauslet/internal/auth/repository/schema"
	"time"
)

type AuthRepository interface {
	// User methods
	CreateUser(ctx context.Context, user *schema.User) error
	GetUserByID(ctx context.Context, id string) (*schema.User, error)
	GetUserByEmail(ctx context.Context, email string) (*schema.User, error)
	GetUserWithIdentities(ctx context.Context, id string) (*schema.User, error)
	UpdateUserLastLogin(ctx context.Context, id string) error
	UpdateUser(ctx context.Context, user *schema.User) error
	DeactivateUser(ctx context.Context, userID string, actorID string, reason string, reactivateOnLogin bool) error
	ReactivateUser(ctx context.Context, userID string, actorID string, setLastLogin bool) error
	DeleteUser(ctx context.Context, id string) error
	ListUsers(ctx context.Context, limit, offset int) ([]schema.User, error)

	// UserIdentity methods
	CreateUserIdentity(ctx context.Context, identity *schema.UserIdentity) error
	GetUserIdentityByID(ctx context.Context, id string) (*schema.UserIdentity, error)
	GetUserIdentityByProvider(ctx context.Context, provider, providerID string) (*schema.UserIdentity, error)
	GetUserIdentityByEmail(ctx context.Context, email string) (*schema.UserIdentity, error)
	ListUserIdentitiesByUserID(ctx context.Context, userID string) ([]schema.UserIdentity, error)
	UpdateUserIdentity(ctx context.Context, identity *schema.UserIdentity) error
	DeleteUserIdentity(ctx context.Context, id string) error

	// Session methods
	CreateSession(ctx context.Context, session *domain.Session) error
	GetSessionByID(ctx context.Context, id string) (*domain.Session, error)
	DeleteSession(ctx context.Context, id string) error
	DeleteSessionsByUserID(ctx context.Context, userID string) error
	ListActiveSessions(ctx context.Context, userID string) ([]domain.Session, error)
	UpdateSessionExpiry(ctx context.Context, id string, expiresAt time.Time) error

	// Atomic operations (transactional)
	CreateUserWithIdentity(ctx context.Context, user *schema.User, identity *schema.UserIdentity) error
	GetOrCreateUserByEmail(ctx context.Context, email string, user *schema.User) (*schema.User, bool, error)
	HardDeleteUser(ctx context.Context, id string) error

	// Role management
	GetUsersByRole(ctx context.Context, role schema.UserRole) ([]schema.User, error)
	UpdateUserRole(ctx context.Context, userID string, role schema.UserRole) error
	CountUsersByRole(ctx context.Context, role schema.UserRole) (int64, error)
}
