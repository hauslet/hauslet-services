package service

import (
	"context"
	"hauslet/config"
	"hauslet/internal/modules/auth/domain"
	"hauslet/internal/modules/auth/notification"
	"hauslet/internal/modules/auth/repository"
	"hauslet/internal/platform/redis"
	"log/slog"
	"sync"
	"time"

	"github.com/go-pkgz/auth/v2"
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

	// OTP Management (registration verification)
	GenerateEmailOTP(ctx context.Context, email string) (string, error)
	VerifyEmailOTP(ctx context.Context, email, code string) error
	DeleteEmailOTP(ctx context.Context, email string) error

	// Passwordless Login OTP Management
	GeneratePasswordlessOTP(ctx context.Context, email string) (string, error)
	VerifyPasswordlessOTP(ctx context.Context, email, code string) error
	DeletePasswordlessOTP(ctx context.Context, email string) error

	// Two-Factor Authentication
	InitiateSetup2FA(ctx context.Context, userID string, method domain.TwoFactorMethod, phoneNumber string) (*domain.SetupResponse, error)
	CompleteSetup2FA(ctx context.Context, userID, code string) (*domain.BackupCodesResult, error)
	Send2FACode(ctx context.Context, userID string) error
	Verify2FACode(ctx context.Context, userID, code string) error
	Verify2FABackupCode(ctx context.Context, userID, code string) error
	Disable2FA(ctx context.Context, userID, code string) error
	RegenerateBackupCodes(ctx context.Context, userID, code string) (*domain.BackupCodesResult, error)
	Get2FAStatus(ctx context.Context, userID string) (*domain.TwoFactorStatus, error)

	// Two-Factor Authentication Login Flow
	Create2FAPendingState(ctx context.Context, userID, email, name, provider string, method domain.TwoFactorMethod) (string, error)
	Get2FAPendingState(ctx context.Context, tempToken string) (*domain.Pending2FAState, error)
	Verify2FALogin(ctx context.Context, tempToken, code string) (*domain.User, error)
	Delete2FAPendingState(ctx context.Context, tempToken string) error

	// Verification Email (for resending OTP)
	ResendVerificationEmail(ctx context.Context, email string) error

	// Email Verification (composite: verify OTP + activate identity + cleanup)
	VerifyAndActivateEmail(ctx context.Context, email, code string) error

	// Passwordless Login (composite: verify OTP + get user + cleanup)
	VerifyPasswordlessLogin(ctx context.Context, email, code string) (*domain.User, error)

	// Config accessors
	GetSiteURL() string
}

type AuthServiceImpl struct {
	repository      repository.AuthRepository
	cfg             *config.AuthConfig
	log             *slog.Logger
	requestMetadata *RequestMetadataStore
	notifier        *notification.NotificationService
	// mailClient       *email.Client
	redisClient redis.RedisClient
	// queueClient      *queue.Client
	// queueSubject     string
	linkStateManager *LinkStateManager
	profileHooks     ProfileHooks
	oauthOnce        sync.Once
	oauthService     *auth.Service
}

func NewAuthService(cfg *config.AuthConfig,
	repository repository.AuthRepository,
	log *slog.Logger,
	notifier *notification.NotificationService,
	// emailClient *email.Client,
	redisClient redis.RedisClient,
	// queueClient *queue.Client,
	// queueSubject string,
	profileHooks ProfileHooks) AuthService {
	return &AuthServiceImpl{
		cfg:        cfg,
		repository: repository,
		log:        log,
		notifier:   notifier,
		// mailClient:       emailClient,
		redisClient: redisClient,
		// queueClient:      queueClient,
		// queueSubject:     queueSubject,
		requestMetadata:  NewRequestMetadataStore(),
		linkStateManager: NewLinkStateManager(cfg.EncryptAuthCodeKey, redisClient),
		profileHooks:     profileHooks,
	}
}

// GetSiteURL returns the configured site URL (used as JWT audience).
func (s *AuthServiceImpl) GetSiteURL() string {
	return s.cfg.RedirectURL
}
func (s *AuthServiceImpl) StoreRequestMetadata(email, ip, userAgent string) {
	s.requestMetadata.Set(email, ip, userAgent)
}

// ProfileHooks defines hooks related to user profile management
type ProfileHooks interface {
	// CreateDefaultProfile initializes a profile for a new user
	CreateDefaultProfile(ctx context.Context, userID, email, name string, birthDate *time.Time) error
	// GetProfileAvatarURL returns the fully-qualified avatar URL for a user, if set
	GetProfileAvatarURL(ctx context.Context, userID string) (*string, error)
}
