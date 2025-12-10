package repository

import (
	"context"
	"errors"
	"hauslet/internal/modules/auth/domain"
	"hauslet/internal/modules/auth/repository/schema"
	"hauslet/internal/modules/auth/session"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AuthRepositoryImpl implements AuthRepository using PostgreSQL via GORM
type AuthRepositoryImpl struct {
	db           *gorm.DB
	sessionStore session.SessionStore
}

// NewAuthRepository creates a new PostgreSQL repository
func NewAuthRepository(db *gorm.DB, sessionStore session.SessionStore) *AuthRepositoryImpl {
	return &AuthRepositoryImpl{
		db:           db,
		sessionStore: sessionStore,
	}
}

// ============================================================================
// User methods
// ============================================================================

// CreateUser creates a new user in the database
func (r *AuthRepositoryImpl) CreateUser(ctx context.Context, user *schema.User) error {
	if user == nil {
		return errors.New("user cannot be nil")
	}

	// Generate UUID if not set
	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}

	return r.db.WithContext(ctx).Create(user).Error
}

// GetUserByID retrieves a user by ID
func (r *AuthRepositoryImpl) GetUserByID(ctx context.Context, id string) (*schema.User, error) {
	userID, err := uuid.Parse(id)
	if err != nil {
		return nil, errors.New("invalid user ID format")
	}

	var user schema.User
	err = r.db.WithContext(ctx).
		Where("id = ?", userID).
		First(&user).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &user, nil
}

// GetUserByEmail retrieves a user by email
func (r *AuthRepositoryImpl) GetUserByEmail(ctx context.Context, email string) (*schema.User, error) {
	if email == "" {
		return nil, errors.New("email cannot be empty")
	}

	var user schema.User
	err := r.db.WithContext(ctx).
		Where("primary_email = ?", email).
		First(&user).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &user, nil
}

// GetUserWithIdentities retrieves a user by ID with all identities preloaded
func (r *AuthRepositoryImpl) GetUserWithIdentities(ctx context.Context, id string) (*schema.User, error) {
	userID, err := uuid.Parse(id)
	if err != nil {
		return nil, errors.New("invalid user ID format")
	}

	var user schema.User
	err = r.db.WithContext(ctx).
		Preload("Identities").
		Where("id = ?", userID).
		First(&user).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &user, nil
}

// UpdateUserLastLogin updates the last login timestamp for a user
func (r *AuthRepositoryImpl) UpdateUserLastLogin(ctx context.Context, id string) error {
	userID, err := uuid.Parse(id)
	if err != nil {
		return errors.New("invalid user ID format")
	}

	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&schema.User{}).
		Where("id = ?", userID).
		Update("last_login_at", &now).Error
}

// UpdateUser updates a user's information
func (r *AuthRepositoryImpl) UpdateUser(ctx context.Context, user *schema.User) error {
	if user == nil {
		return errors.New("user cannot be nil")
	}

	if user.ID == uuid.Nil {
		return errors.New("user ID is required for update")
	}

	return r.db.WithContext(ctx).
		Model(&schema.User{}).
		Where("id = ?", user.ID).
		Updates(user).Error
}

// DeactivateUser marks a user as inactive with metadata and optional auto-reactivation
func (r *AuthRepositoryImpl) DeactivateUser(ctx context.Context, userID string, actorID string, reason string, reactivateOnLogin bool) error {
	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		return errors.New("invalid user ID format")
	}

	now := time.Now()
	var reasonPtr *string
	if reason != "" {
		reasonPtr = &reason
	}

	var actorPtr *uuid.UUID
	if actorID != "" {
		parsedActorID, err := uuid.Parse(actorID)
		if err != nil {
			return errors.New("invalid actor ID format")
		}
		actorPtr = &parsedActorID
	}

	updates := map[string]interface{}{
		"is_active":           false,
		"deactivated_at":      &now,
		"deactivated_reason":  reasonPtr,
		"deactivated_by":      actorPtr,
		"reactivate_on_login": reactivateOnLogin,
		"reactivated_at":      nil,
		"reactivated_by":      nil,
	}

	result := r.db.WithContext(ctx).
		Model(&schema.User{}).
		Where("id = ?", parsedUserID).
		Updates(updates)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

// ReactivateUser sets a user back to active and optionally updates last login
func (r *AuthRepositoryImpl) ReactivateUser(ctx context.Context, userID string, actorID string, setLastLogin bool) error {
	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		return errors.New("invalid user ID format")
	}

	var actorPtr *uuid.UUID
	if actorID != "" {
		parsedActorID, err := uuid.Parse(actorID)
		if err != nil {
			return errors.New("invalid actor ID format")
		}
		actorPtr = &parsedActorID
	}

	now := time.Now()
	updates := map[string]interface{}{
		"is_active":           true,
		"deactivated_at":      nil,
		"deactivated_reason":  nil,
		"reactivate_on_login": false,
		"reactivated_at":      &now,
		"reactivated_by":      actorPtr,
	}

	if setLastLogin {
		updates["last_login_at"] = &now
	}

	result := r.db.WithContext(ctx).
		Model(&schema.User{}).
		Where("id = ?", parsedUserID).
		Updates(updates)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

// DeleteUser soft deletes a user (sets DeletedAt)
func (r *AuthRepositoryImpl) DeleteUser(ctx context.Context, id string) error {
	userID, err := uuid.Parse(id)
	if err != nil {
		return errors.New("invalid user ID format")
	}

	return r.db.WithContext(ctx).
		Delete(&schema.User{}, userID).Error
}

// ListUsers retrieves a paginated list of users
func (r *AuthRepositoryImpl) ListUsers(ctx context.Context, limit, offset int) ([]schema.User, error) {
	var users []schema.User

	err := r.db.WithContext(ctx).
		Limit(limit).
		Offset(offset).
		Order("created_at DESC").
		Find(&users).Error

	if err != nil {
		return nil, err
	}

	return users, nil
}

// ============================================================================
// UserIdentity methods
// ============================================================================

// CreateUserIdentity creates a new user identity
func (r *AuthRepositoryImpl) CreateUserIdentity(ctx context.Context, identity *schema.UserIdentity) error {
	if identity == nil {
		return errors.New("identity cannot be nil")
	}

	// Generate UUID if not set
	if identity.ID == uuid.Nil {
		identity.ID = uuid.New()
	}

	return r.db.WithContext(ctx).Create(identity).Error
}

// GetUserIdentityByID retrieves a user identity by its ID
func (r *AuthRepositoryImpl) GetUserIdentityByID(ctx context.Context, id string) (*schema.UserIdentity, error) {
	identityID, err := uuid.Parse(id)
	if err != nil {
		return nil, errors.New("invalid identity ID format")
	}

	var identity schema.UserIdentity
	err = r.db.WithContext(ctx).
		Where("id = ?", identityID).
		First(&identity).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &identity, nil
}

// GetUserIdentityByProvider retrieves a user identity by provider and provider ID
func (r *AuthRepositoryImpl) GetUserIdentityByProvider(ctx context.Context, provider, providerID string) (*schema.UserIdentity, error) {
	if provider == "" || providerID == "" {
		return nil, errors.New("provider and providerID cannot be empty")
	}

	var identity schema.UserIdentity
	err := r.db.WithContext(ctx).
		Where("provider = ? AND provider_id = ?", provider, providerID).
		First(&identity).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &identity, nil
}

// GetUserIdentityByEmail retrieves the first user identity with the given email
func (r *AuthRepositoryImpl) GetUserIdentityByEmail(ctx context.Context, email string) (*schema.UserIdentity, error) {
	if email == "" {
		return nil, errors.New("email cannot be empty")
	}

	var identity schema.UserIdentity
	err := r.db.WithContext(ctx).
		Where("email = ?", email).
		First(&identity).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &identity, nil
}

// ListUserIdentitiesByUserID retrieves all identities for a user
func (r *AuthRepositoryImpl) ListUserIdentitiesByUserID(ctx context.Context, userID string) ([]schema.UserIdentity, error) {
	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		return nil, errors.New("invalid user ID format")
	}

	var identities []schema.UserIdentity
	err = r.db.WithContext(ctx).
		Where("user_id = ?", parsedUserID).
		Order("created_at DESC").
		Find(&identities).Error

	if err != nil {
		return nil, err
	}

	return identities, nil
}

// UpdateUserIdentity updates a user identity
func (r *AuthRepositoryImpl) UpdateUserIdentity(ctx context.Context, identity *schema.UserIdentity) error {
	if identity == nil {
		return errors.New("identity cannot be nil")
	}

	if identity.ID == uuid.Nil {
		return errors.New("identity ID is required for update")
	}

	return r.db.WithContext(ctx).
		Model(&schema.UserIdentity{}).
		Where("id = ?", identity.ID).
		Updates(identity).Error
}

// DeleteUserIdentity soft deletes a user identity
func (r *AuthRepositoryImpl) DeleteUserIdentity(ctx context.Context, id string) error {
	identityID, err := uuid.Parse(id)
	if err != nil {
		return errors.New("invalid identity ID format")
	}

	return r.db.WithContext(ctx).
		Delete(&schema.UserIdentity{}, identityID).Error
}

// ============================================================================
// Session methods (sessions stored in Redis)
// ============================================================================

func (r *AuthRepositoryImpl) CreateSession(ctx context.Context, session *domain.Session) error {
	if session == nil {
		return errors.New("session cannot be nil")
	}

	return r.sessionStore.Store(session.ID, session)
}

func (r *AuthRepositoryImpl) GetSessionByID(ctx context.Context, id string) (*domain.Session, error) {
	if id == "" {
		return nil, errors.New("session ID cannot be empty")
	}

	session, exists := r.sessionStore.Load(id)
	if !exists {
		return nil, nil
	}

	return session, nil
}

func (r *AuthRepositoryImpl) DeleteSession(ctx context.Context, id string) error {
	if id == "" {
		return errors.New("session ID cannot be empty")
	}

	return r.sessionStore.Delete(id)
}

func (r *AuthRepositoryImpl) DeleteSessionsByUserID(ctx context.Context, userID string) error {
	if userID == "" {
		return errors.New("user ID cannot be empty")
	}

	return r.sessionStore.DeleteByUserID(userID)
}

func (r *AuthRepositoryImpl) ListActiveSessions(ctx context.Context, userID string) ([]domain.Session, error) {
	if userID == "" {
		return nil, errors.New("user ID cannot be empty")
	}

	sessions, err := r.sessionStore.ListActiveByUserID(userID)
	if err != nil {
		return nil, err
	}

	// Convert []*domain.Session to []domain.Session
	result := make([]domain.Session, len(sessions))
	for i, session := range sessions {
		if session != nil {
			result[i] = *session
		}
	}

	return result, nil
}

func (r *AuthRepositoryImpl) UpdateSessionExpiry(ctx context.Context, id string, expiresAt time.Time) error {
	if id == "" {
		return errors.New("session ID cannot be empty")
	}

	return r.sessionStore.UpdateExpiry(id, expiresAt)
}

// ============================================================================
// Atomic operations
// ============================================================================

// CreateUserWithIdentity creates a user and identity in a transaction
func (r *AuthRepositoryImpl) CreateUserWithIdentity(ctx context.Context, user *schema.User, identity *schema.UserIdentity) error {
	if user == nil || identity == nil {
		return errors.New("user and identity cannot be nil")
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Generate user ID if not set
		if user.ID == uuid.Nil {
			user.ID = uuid.New()
		}

		// Create user
		if err := tx.Create(user).Error; err != nil {
			return err
		}

		// Set UserID on identity
		identity.UserID = user.ID

		// Generate identity ID if not set
		if identity.ID == uuid.Nil {
			identity.ID = uuid.New()
		}

		// Create identity
		if err := tx.Create(identity).Error; err != nil {
			return err
		}

		return nil
	})
}

// GetOrCreateUserByEmail gets an existing user by email or creates a new one
// Returns (user, wasCreated, error)
func (r *AuthRepositoryImpl) GetOrCreateUserByEmail(ctx context.Context, email string, user *schema.User) (*schema.User, bool, error) {
	if email == "" {
		return nil, false, errors.New("email cannot be empty")
	}

	if user == nil {
		return nil, false, errors.New("user cannot be nil")
	}

	// Try to get existing user
	existingUser, err := r.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, false, err
	}

	// User exists, return it
	if existingUser != nil {
		return existingUser, false, nil
	}

	// User doesn't exist, create it
	user.PrimaryEmail = email
	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}

	if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
		return nil, false, err
	}

	return user, true, nil
}

// HardDeleteUser permanently deletes a user and their identities from the database
func (r *AuthRepositoryImpl) HardDeleteUser(ctx context.Context, id string) error {
	userID, err := uuid.Parse(id)
	if err != nil {
		return errors.New("invalid user ID format")
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Unscoped().Where("user_id = ?", userID).Delete(&schema.UserIdentity{}).Error; err != nil {
			return err
		}

		if err := tx.Unscoped().Where("id = ?", userID).Delete(&schema.User{}).Error; err != nil {
			return err
		}

		return nil
	})
}

// ============================================================================
// Role Management
// ============================================================================

// GetUsersByRole retrieves all users with a specific role
func (r *AuthRepositoryImpl) GetUsersByRole(ctx context.Context, role schema.UserRole) ([]schema.User, error) {
	var users []schema.User

	err := r.db.WithContext(ctx).
		Where("role = ?", role).
		Order("created_at ASC").
		Find(&users).Error

	if err != nil {
		return nil, err
	}

	return users, nil
}

// UpdateUserRole updates a user's role
func (r *AuthRepositoryImpl) UpdateUserRole(ctx context.Context, userID string, role schema.UserRole) error {
	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		return errors.New("invalid user ID format")
	}

	return r.db.WithContext(ctx).
		Model(&schema.User{}).
		Where("id = ?", parsedUserID).
		Update("role", role).Error
}

// CountUsersByRole counts how many users have a specific role
func (r *AuthRepositoryImpl) CountUsersByRole(ctx context.Context, role schema.UserRole) (int64, error) {
	var count int64

	err := r.db.WithContext(ctx).
		Model(&schema.User{}).
		Where("role = ?", role).
		Count(&count).Error

	if err != nil {
		return 0, err
	}

	return count, nil
}
