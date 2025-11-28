package service

import (
	"context"
	"errors"
	"fmt"

	"hauslet/internal/auth/domain"
	"hauslet/internal/auth/repository/schema"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// ============================================================================
// Root User Management
// ============================================================================

// CreateRootUser creates a new root user with password authentication
// This should only be called manually (e.g., via CLI or setup script)
func (s *AuthServiceImpl) CreateRootUser(ctx context.Context, email, password, name string) (*domain.User, error) {
	// Validate inputs
	if email == "" || password == "" || name == "" {
		return nil, errors.New("email, password, and name are required")
	}

	// Check if user already exists
	existingUser, _ := s.repository.GetUserByEmail(ctx, email)
	if existingUser != nil {
		return nil, fmt.Errorf("user with email %s already exists", email)
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user with ROOT role
	user := &schema.User{
		ID:           uuid.New(),
		Name:         name,
		PrimaryEmail: email,
		Role:         schema.RoleRoot,
		IsActive:     true,
	}

	// Create password identity
	hashedPasswordStr := string(hashedPassword)
	identity := &schema.UserIdentity{
		ID:            uuid.New(),
		UserID:        user.ID,
		Provider:      "password",
		ProviderID:    user.ID.String(),
		Email:         email,
		EmailVerified: true, // Root user is auto-verified
		PasswordHash:  &hashedPasswordStr,
	}

	// Create user and identity atomically
	if err := s.repository.CreateUserWithIdentity(ctx, user, identity); err != nil {
		return nil, fmt.Errorf("failed to create root user: %w", err)
	}

	return domain.MapUserFromSchema(user), nil
}

// EnsureRootUserExists creates a root user if one doesn't already exist
// This is idempotent and safe to call on every startup
func (s *AuthServiceImpl) EnsureRootUserExists(ctx context.Context, email, password, name string) (*domain.User, error) {
	// Check if ANY root user exists
	rootCount, err := s.repository.CountUsersByRole(ctx, schema.RoleRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to check for root users: %w", err)
	}

	if rootCount > 0 {
		// Root user(s) already exist, fetch by email
		existingRoot, _ := s.repository.GetUserByEmail(ctx, email)
		if existingRoot != nil {
			return domain.MapUserFromSchema(existingRoot), nil
		}
		// Different root exists, return nil (not an error)
		return nil, nil
	}

	// No root user exists, create one
	return s.CreateRootUser(ctx, email, password, name)
}

// GetRootCount returns the number of root users
func (s *AuthServiceImpl) GetRootCount(ctx context.Context) (int64, error) {
	count, err := s.repository.CountUsersByRole(ctx, schema.RoleRoot)
	if err != nil {
		return 0, fmt.Errorf("failed to count root users: %w", err)
	}
	return count, nil
}

// ListRootUsers returns all root users
func (s *AuthServiceImpl) ListRootUsers(ctx context.Context) ([]domain.User, error) {
	schemaUsers, err := s.repository.GetUsersByRole(ctx, schema.RoleRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to list root users: %w", err)
	}

	return domain.MapUsersFromSchema(schemaUsers), nil
}

// ListAdminUsers returns all admin users (not including root)
func (s *AuthServiceImpl) ListAdminUsers(ctx context.Context) ([]domain.User, error) {
	schemaUsers, err := s.repository.GetUsersByRole(ctx, schema.RoleAdmin)
	if err != nil {
		return nil, fmt.Errorf("failed to list admin users: %w", err)
	}

	return domain.MapUsersFromSchema(schemaUsers), nil
}

// ============================================================================
// Role Promotion/Demotion
// ============================================================================

// PromoteToAdmin promotes a user to admin role
// Only root users can promote to admin
func (s *AuthServiceImpl) PromoteToAdmin(ctx context.Context, actorID, targetUserID string) error {
	// Verify actor is root
	actor, err := s.repository.GetUserByID(ctx, actorID)
	if err != nil {
		return fmt.Errorf("failed to get actor: %w", err)
	}
	if actor == nil {
		return errors.New("actor not found")
	}
	if actor.Role != schema.RoleRoot {
		return errors.New("only root users can promote to admin")
	}

	// Get target user
	target, err := s.repository.GetUserByID(ctx, targetUserID)
	if err != nil {
		return fmt.Errorf("failed to get target user: %w", err)
	}
	if target == nil {
		return errors.New("target user not found")
	}

	// Check if already admin or root
	if target.Role == schema.RoleAdmin {
		return errors.New("user is already an admin")
	}
	if target.Role == schema.RoleRoot {
		return errors.New("cannot change role of root user")
	}

	// Promote to admin
	if err := s.repository.UpdateUserRole(ctx, targetUserID, schema.RoleAdmin); err != nil {
		return fmt.Errorf("failed to promote user to admin: %w", err)
	}

	return nil
}

// DemoteFromAdmin demotes an admin to regular user
// Only root users can demote admins
func (s *AuthServiceImpl) DemoteFromAdmin(ctx context.Context, actorID, targetUserID string) error {
	// Verify actor is root
	actor, err := s.repository.GetUserByID(ctx, actorID)
	if err != nil {
		return fmt.Errorf("failed to get actor: %w", err)
	}
	if actor == nil {
		return errors.New("actor not found")
	}
	if actor.Role != schema.RoleRoot {
		return errors.New("only root users can demote admins")
	}

	// Get target user
	target, err := s.repository.GetUserByID(ctx, targetUserID)
	if err != nil {
		return fmt.Errorf("failed to get target user: %w", err)
	}
	if target == nil {
		return errors.New("target user not found")
	}

	// Check if user is root
	if target.Role == schema.RoleRoot {
		return errors.New("cannot demote root user")
	}

	// Check if user is admin
	if target.Role != schema.RoleAdmin {
		return fmt.Errorf("user is not an admin (current role: %s)", target.Role)
	}

	// Demote to user
	if err := s.repository.UpdateUserRole(ctx, targetUserID, schema.RoleUser); err != nil {
		return fmt.Errorf("failed to demote admin: %w", err)
	}

	return nil
}

// PromoteToRoot promotes a user to root role
// Only existing root users can promote to root (for future multi-root support)
func (s *AuthServiceImpl) PromoteToRoot(ctx context.Context, actorID, targetUserID string) error {
	// Verify actor is root
	actor, err := s.repository.GetUserByID(ctx, actorID)
	if err != nil {
		return fmt.Errorf("failed to get actor: %w", err)
	}
	if actor == nil {
		return errors.New("actor not found")
	}
	if actor.Role != schema.RoleRoot {
		return errors.New("only root users can promote to root")
	}

	// Get target user
	target, err := s.repository.GetUserByID(ctx, targetUserID)
	if err != nil {
		return fmt.Errorf("failed to get target user: %w", err)
	}
	if target == nil {
		return errors.New("target user not found")
	}

	// Check if already root
	if target.Role == schema.RoleRoot {
		return errors.New("user is already a root user")
	}

	// Promote to root
	if err := s.repository.UpdateUserRole(ctx, targetUserID, schema.RoleRoot); err != nil {
		return fmt.Errorf("failed to promote user to root: %w", err)
	}

	return nil
}

// ============================================================================
// Flexible Role Management
// ============================================================================

// ChangeUserRole changes a user's role with proper permission checks
// This is the primary method for role changes with granular access control
func (s *AuthServiceImpl) ChangeUserRole(ctx context.Context, actorID, targetUserID string, newRole domain.UserRole) error {
	// 1. Get actor (person making the change)
	actor, err := s.repository.GetUserByID(ctx, actorID)
	if err != nil {
		return fmt.Errorf("failed to get actor: %w", err)
	}
	if actor == nil {
		return errors.New("actor not found")
	}

	// 2. Get target user
	target, err := s.repository.GetUserByID(ctx, targetUserID)
	if err != nil {
		return fmt.Errorf("failed to get target user: %w", err)
	}
	if target == nil {
		return errors.New("target user not found")
	}

	// 3. SAFEGUARD: Cannot change own role (prevent self-promotion/demotion)
	if actorID == targetUserID {
		return errors.New("cannot change your own role")
	}

	// 4. SAFEGUARD: Cannot change root user's role
	if target.Role == schema.RoleRoot {
		return errors.New("cannot change role of root user")
	}

	// 5. Check if role is already set
	if target.Role == schema.UserRole(newRole) {
		return fmt.Errorf("user already has role '%s'", newRole)
	}

	// 6. Permission check based on target role
	switch newRole {
	case domain.RoleRoot:
		// Only root can promote to root
		if actor.Role != schema.RoleRoot {
			return errors.New("only root users can promote to root")
		}

	case domain.RoleAdmin:
		// Only root can promote to admin
		if actor.Role != schema.RoleRoot {
			return errors.New("only root users can promote to admin")
		}

	case domain.RoleModerator, domain.RoleSupport, domain.RoleStaff:
		// Admin or root can assign these mid-level roles
		if actor.Role != schema.RoleRoot && actor.Role != schema.RoleAdmin {
			return errors.New("admin or root access required to assign this role")
		}

	case domain.RoleUser:
		// Admin or root can demote to user
		if actor.Role != schema.RoleRoot && actor.Role != schema.RoleAdmin {
			return errors.New("admin or root access required to change role")
		}

	default:
		return fmt.Errorf("invalid role: %s", newRole)
	}

	// 7. Additional permission check: Admin cannot demote another admin
	if actor.Role == schema.RoleAdmin && target.Role == schema.RoleAdmin {
		return errors.New("admins cannot change other admins' roles - root access required")
	}

	// 8. Apply role change
	if err := s.repository.UpdateUserRole(ctx, targetUserID, schema.UserRole(newRole)); err != nil {
		return fmt.Errorf("failed to change user role: %w", err)
	}

	return nil
}

// GetUsersByRole retrieves all users with a specific role
func (s *AuthServiceImpl) GetUsersByRole(ctx context.Context, role domain.UserRole) ([]domain.User, error) {
	schemaUsers, err := s.repository.GetUsersByRole(ctx, schema.UserRole(role))
	if err != nil {
		return nil, fmt.Errorf("failed to get users by role: %w", err)
	}

	return domain.MapUsersFromSchema(schemaUsers), nil
}