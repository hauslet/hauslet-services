package service

import (
	"context"
	"hauslet/config"
	"hauslet/internal/auth/domain"
	"hauslet/internal/auth/repository"
	"time"

	"github.com/go-pkgz/auth"
	"github.com/go-pkgz/lgr"
)

type AuthService interface {
	// OAuth configuration
	OAuthService() *auth.Service

	// User management
	GetUser(ctx context.Context, userID string) (*domain.User, error)
	GetUserByEmail(ctx context.Context, email string) (*domain.User, error)
	UpdateUser(ctx context.Context, user *domain.User) error
	DeactivateUser(ctx context.Context, userID string) error
	ListUsers(ctx context.Context, limit, offset int) ([]domain.User, error)

	// Identity management
	ListUserIdentities(ctx context.Context, userID string) ([]domain.UserIdentity, error)
	UnlinkIdentity(ctx context.Context, identityID string) error

	// Session management
	GetUserSessions(ctx context.Context, userID string) ([]domain.Session, error)
	RevokeSession(ctx context.Context, sessionID string) error
	RevokeAllUserSessions(ctx context.Context, userID string) error
	ExtendSession(ctx context.Context, sessionID string, duration time.Duration) error

	// Password authentication (for email/password login)
	CreatePasswordUser(ctx context.Context, email, password, name string) (*domain.User, error)
	AuthenticatePassword(ctx context.Context, email, password string) (*domain.User, error)
	ChangePassword(ctx context.Context, userID, oldPassword, newPassword string) error

	// Root and Admin management
	CreateRootUser(ctx context.Context, email, password, name string) (*domain.User, error)
	EnsureRootUserExists(ctx context.Context, email, password, name string) (*domain.User, error)
	PromoteToAdmin(ctx context.Context, actorID, targetUserID string) error
	DemoteFromAdmin(ctx context.Context, actorID, targetUserID string) error
	PromoteToRoot(ctx context.Context, actorID, targetUserID string) error
	ListRootUsers(ctx context.Context) ([]domain.User, error)
	ListAdminUsers(ctx context.Context) ([]domain.User, error)
	GetRootCount(ctx context.Context) (int64, error)

	// Role management
	ChangeUserRole(ctx context.Context, actorID, targetUserID string, newRole domain.UserRole) error
	GetUsersByRole(ctx context.Context, role domain.UserRole) ([]domain.User, error)

	// Request metadata (for session creation)
	StoreRequestMetadata(email, ip, userAgent string)
}

type AuthServiceImpl struct {
	repository      repository.AuthRepository
	cfg             *config.AuthConfig
	log             *lgr.Logger
	requestMetadata *RequestMetadataStore
}

func NewAuthService(cfg *config.AuthConfig, repository repository.AuthRepository, log *lgr.Logger) AuthService {
	return &AuthServiceImpl{
		cfg:             cfg,
		repository:      repository,
		log:             log,
		requestMetadata: NewRequestMetadataStore(),
	}
}
