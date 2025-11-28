package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"hauslet/internal/auth/domain"
	"hauslet/internal/auth/repository/schema"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// ============================================================================
// User Management
// ============================================================================

// GetUser retrieves a user by ID
func (s *AuthServiceImpl) GetUser(ctx context.Context, userID string) (*domain.User, error) {
	schemaUser, err := s.repository.GetUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if schemaUser == nil {
		return nil, errors.New("user not found")
	}

	return domain.MapUserFromSchema(schemaUser), nil
}

// GetUserByEmail retrieves a user by email
func (s *AuthServiceImpl) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	schemaUser, err := s.repository.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	if schemaUser == nil {
		return nil, errors.New("user not found")
	}

	return domain.MapUserFromSchema(schemaUser), nil
}

// UpdateUser updates user information 
// SAFEGUARD: Cannot change role of root user
func (s *AuthServiceImpl) UpdateUser(ctx context.Context, user *domain.User) error {
	if user == nil {
		return errors.New("user cannot be nil")
	}

	// Check if trying to modify a root user's role
	existingUser, err := s.repository.GetUserByID(ctx, user.ID.String())
	if err != nil {
		return fmt.Errorf("failed to get existing user: %w", err)
	}
	if existingUser == nil {
		return errors.New("user not found")
	}

	// SAFEGUARD: Cannot change role of root user through UpdateUser
	if existingUser.Role == schema.RoleRoot && user.Role != domain.RoleRoot {
		return errors.New("cannot demote root user through UpdateUser - this operation is not allowed")
	}

	schemaUser := domain.MapUserToSchema(user)
	if err := s.repository.UpdateUser(ctx, schemaUser); err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

// DeactivateUser deactivates a user account
// SAFEGUARD: Cannot deactivate root user
func (s *AuthServiceImpl) DeactivateUser(ctx context.Context, userID string) error {
	user, err := s.GetUser(ctx, userID)
	if err != nil {
		return err
	}

	// SAFEGUARD: Cannot deactivate root user
	if user.Role == domain.RoleRoot {
		return errors.New("cannot deactivate root user")
	}

	user.IsActive = false
	if err := s.UpdateUser(ctx, user); err != nil {
		return fmt.Errorf("failed to deactivate user: %w", err)
	}

	// Also revoke all sessions
	if err := s.RevokeAllUserSessions(ctx, userID); err != nil {
		return fmt.Errorf("failed to revoke user sessions: %w", err)
	}

	return nil
}

// ListUsers retrieves a paginated list of users
func (s *AuthServiceImpl) ListUsers(ctx context.Context, limit, offset int) ([]domain.User, error) {
	schemaUsers, err := s.repository.ListUsers(ctx, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	return domain.MapUsersFromSchema(schemaUsers), nil
}

// ============================================================================
// Identity Management
// ============================================================================

// ListUserIdentities retrieves all identities for a user
func (s *AuthServiceImpl) ListUserIdentities(ctx context.Context, userID string) ([]domain.UserIdentity, error) {
	schemaIdentities, err := s.repository.ListUserIdentitiesByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list user identities: %w", err)
	}

	identities := make([]domain.UserIdentity, len(schemaIdentities))
	for i, schemaIdentity := range schemaIdentities {
		if mapped := domain.MapUserIdentityFromSchema(&schemaIdentity); mapped != nil {
			identities[i] = *mapped
		}
	}

	return identities, nil
}

// UnlinkIdentity removes an OAuth identity from a user
func (s *AuthServiceImpl) UnlinkIdentity(ctx context.Context, identityID string) error {
	// Get the identity to find which user it belongs to
	identity, err := s.repository.GetUserIdentityByID(ctx, identityID)
	if err != nil {
		return fmt.Errorf("failed to get identity: %w", err)
	}
	if identity == nil {
		return fmt.Errorf("identity not found")
	}

	// Get all identities for this user
	userIdentities, err := s.repository.ListUserIdentitiesByUserID(ctx, identity.UserID.String())
	if err != nil {
		return fmt.Errorf("failed to get user identities: %w", err)
	}

	// Prevent unlinking the last identity (user must have at least one way to login)
	if len(userIdentities) <= 1 {
		return fmt.Errorf("cannot unlink the last identity: user must have at least one login method")
	}

	// Delete the identity
	if err := s.repository.DeleteUserIdentity(ctx, identityID); err != nil {
		return fmt.Errorf("failed to unlink identity: %w", err)
	}

	return nil
}

// ============================================================================
// Session Management
// ============================================================================

// GetUserSessions retrieves all active sessions for a user
func (s *AuthServiceImpl) GetUserSessions(ctx context.Context, userID string) ([]domain.Session, error) {
	sessions, err := s.repository.ListActiveSessions(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user sessions: %w", err)
	}

	return sessions, nil
}

// RevokeSession revokes a specific session
func (s *AuthServiceImpl) RevokeSession(ctx context.Context, sessionID string) error {
	if err := s.repository.DeleteSession(ctx, sessionID); err != nil {
		return fmt.Errorf("failed to revoke session: %w", err)
	}

	return nil
}

// RevokeAllUserSessions revokes all sessions for a user (logout everywhere)
func (s *AuthServiceImpl) RevokeAllUserSessions(ctx context.Context, userID string) error {
	if err := s.repository.DeleteSessionsByUserID(ctx, userID); err != nil {
		return fmt.Errorf("failed to revoke all user sessions: %w", err)
	}

	return nil
}

// ExtendSession extends the expiration time of a session
func (s *AuthServiceImpl) ExtendSession(ctx context.Context, sessionID string, duration time.Duration) error {
	newExpiry := time.Now().Add(duration)

	if err := s.repository.UpdateSessionExpiry(ctx, sessionID, newExpiry); err != nil {
		return fmt.Errorf("failed to extend session: %w", err)
	}

	return nil
}

// ============================================================================
// Password Authentication
// ============================================================================

// CreatePasswordUser creates a new user with email/password authentication
func (s *AuthServiceImpl) CreatePasswordUser(ctx context.Context, email, password, name string) (*domain.User, error) {
	// Validate inputs
	if email == "" {
		return nil, domain.ErrEmailRequired
	}
	if password == "" {
		return nil, domain.ErrPasswordRequired
	}
	if name == "" {
		return nil, domain.ErrNameRequired
	}

	// Check if user already exists
	existingUser, _ := s.repository.GetUserByEmail(ctx, email)
	if existingUser != nil {
		return nil, domain.ErrUserAlreadyExists
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
	user := &schema.User{
		ID:           uuid.New(),
		Name:         name,
		PrimaryEmail: email,
		Role:         schema.RoleUser,
		IsActive:     true,
	}

	// Create password identity
	hashedPasswordStr := string(hashedPassword)
	identity := &schema.UserIdentity{
		ID:            uuid.New(),
		UserID:        user.ID,
		Provider:      "password",
		ProviderID:    user.ID.String(), // For password provider, use user ID
		Email:         email,
		EmailVerified: false, // Requires email verification
		PasswordHash:  &hashedPasswordStr,
	}

	// Create user and identity atomically
	if err := s.repository.CreateUserWithIdentity(ctx, user, identity); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return domain.MapUserFromSchema(user), nil
}

// AuthenticatePassword authenticates a user with email/password
func (s *AuthServiceImpl) AuthenticatePassword(ctx context.Context, email, password string) (*domain.User, error) {
	// Get user by email
	user, err := s.repository.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	if user == nil {
		return nil, errors.New("invalid email or password")
	}

	// Check if user is active
	if !user.IsActive {
		return nil, errors.New("user account is deactivated")
	}

	// Get password identity
	identities, err := s.repository.ListUserIdentitiesByUserID(ctx, user.ID.String())
	if err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	// Find password identity
	var passwordIdentity *schema.UserIdentity
	for _, identity := range identities {
		if identity.Provider == "password" {
			passwordIdentity = &identity
			break
		}
	}

	if passwordIdentity == nil || passwordIdentity.PasswordHash == nil {
		return nil, errors.New("invalid email or password")
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(*passwordIdentity.PasswordHash), []byte(password)); err != nil {
		return nil, errors.New("invalid email or password")
	}

	// Update last login
	_ = s.repository.UpdateUserLastLogin(ctx, user.ID.String())

	return domain.MapUserFromSchema(user), nil
}

// ChangePassword changes a user's password
func (s *AuthServiceImpl) ChangePassword(ctx context.Context, userID, oldPassword, newPassword string) error {
	// Get user
	user, err := s.repository.GetUserByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	if user == nil {
		return errors.New("user not found")
	}

	// Get password identity
	identities, err := s.repository.ListUserIdentitiesByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user identities: %w", err)
	}

	var passwordIdentity *schema.UserIdentity
	for _, identity := range identities {
		if identity.Provider == "password" {
			passwordIdentity = &identity
			break
		}
	}

	if passwordIdentity == nil || passwordIdentity.PasswordHash == nil {
		return errors.New("user does not have password authentication enabled")
	}

	// Verify old password
	if err := bcrypt.CompareHashAndPassword([]byte(*passwordIdentity.PasswordHash), []byte(oldPassword)); err != nil {
		return errors.New("invalid old password")
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Update password
	hashedPasswordStr := string(hashedPassword)
	passwordIdentity.PasswordHash = &hashedPasswordStr

	if err := s.repository.UpdateUserIdentity(ctx, passwordIdentity); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	return nil
}
