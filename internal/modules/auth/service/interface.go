package service

import (
	"context"
	"hauslet/config"
	"hauslet/internal/modules/auth/domain"
	"hauslet/internal/modules/auth/repository"
	"hauslet/internal/platform/email"
	"hauslet/internal/platform/queue"
	"hauslet/internal/platform/redis"
	"sync"
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
	UpdateIdentityVerified(ctx context.Context, email string) error
	InitiateIdentityLinking(userID, provider, redirectURI string) (string, error)
	RequestPasswordReset(ctx context.Context, email string) error
	ResetPassword(ctx context.Context, email, token, newPassword string) error

	// Session management
	GetUserSessions(ctx context.Context, userID string) ([]domain.Session, error)
	RevokeSession(ctx context.Context, sessionID string) error
	RevokeAllUserSessions(ctx context.Context, userID string) error
	ExtendSession(ctx context.Context, sessionID string, duration time.Duration) error

	// Password authentication (for email/password login)
	LinkPasswordIdentity(ctx context.Context, userID, email, password string) (*domain.UserIdentity, error)
	CreatePasswordUser(ctx context.Context, email, password, name string, birthDate *time.Time) (*domain.User, error)
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

	// Email operations
	SendWelcomeEmail(ctx context.Context, email, name string, otpCode string) error
	SendIdentityLinkedEmail(ctx context.Context, emailAddr string, name string, provider string) error
	SendPasswordChangedEmail(ctx context.Context, emailAddr string, name string) error
	SendPasswordResetEmail(ctx context.Context, emailAddr string, name string, token string, ttlMinutes int) error

	// OTP Management
	GenerateEmailOTP(ctx context.Context, email string) (string, error)
	VerifyEmailOTP(ctx context.Context, email, code string) error
	DeleteEmailOTP(ctx context.Context, email string) error
}

type AuthServiceImpl struct {
	repository       repository.AuthRepository
	cfg              *config.AuthConfig
	log              *lgr.Logger
	requestMetadata  *RequestMetadataStore
	mailClient       *email.Client
	redisClient      redis.RedisClient
	queueClient      *queue.Client
	queueSubject     string
	linkStateManager *LinkStateManager
	profileHooks     ProfileHooks
	oauthOnce        sync.Once
	oauthService     *auth.Service
}

func NewAuthService(cfg *config.AuthConfig,
	repository repository.AuthRepository,
	log *lgr.Logger,
	emailClient *email.Client,
	redisClient redis.RedisClient,
	queueClient *queue.Client,
	queueSubject string,
	profileHooks ProfileHooks) AuthService {
	return &AuthServiceImpl{
		cfg:              cfg,
		repository:       repository,
		log:              log,
		mailClient:       emailClient,
		redisClient:      redisClient,
		queueClient:      queueClient,
		queueSubject:     queueSubject,
		requestMetadata:  NewRequestMetadataStore(),
		linkStateManager: NewLinkStateManager(cfg.EncryptAuthCodeKey),
		profileHooks:     profileHooks,
	}
}

// ProfileHooks defines hooks related to user profile management
type ProfileHooks interface {
	// CreateDefaultProfile initializes a profile for a new user
	CreateDefaultProfile(ctx context.Context, userID string, name string, birthDate *time.Time) error
}
