package service

import (
	"context"
	"errors"
	"fmt"

	"hauslet/internal/modules/auth/domain"
	"hauslet/internal/modules/auth/repository/schema"
)

const reasonUserSelfDeactivated = "user_self_deactivated"

// User management

func (s *AuthServiceImpl) GetUser(ctx context.Context, userID string) (*domain.User, error) {
	schemaUser, err := s.repository.GetUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if schemaUser == nil {
		return nil, errors.New("user not found")
	}
	user := domain.MapUserFromSchema(schemaUser)
	s.enrichUserAvatar(ctx, user)
	return user, nil
}

func (s *AuthServiceImpl) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	schemaUser, err := s.repository.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}
	if schemaUser == nil {
		return nil, errors.New("user not found")
	}
	user := domain.MapUserFromSchema(schemaUser)
	s.enrichUserAvatar(ctx, user)
	return user, nil
}

// SAFEGUARD: Cannot change role of root user
func (s *AuthServiceImpl) UpdateUser(ctx context.Context, user *domain.User) error {
	if user == nil {
		return errors.New("user cannot be nil")
	}

	existingUser, err := s.repository.GetUserByID(ctx, user.ID.String())
	if err != nil {
		return fmt.Errorf("failed to get existing user: %w", err)
	}
	if existingUser == nil {
		return errors.New("user not found")
	}

	if existingUser.Role == schema.RoleRoot && user.Role != domain.RoleRoot {
		return errors.New("cannot demote root user through UpdateUser - this operation is not allowed")
	}

	schemaUser := domain.MapUserToSchema(user)
	if err := s.repository.UpdateUser(ctx, schemaUser); err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

// SAFEGUARD: Cannot deactivate root user
func (s *AuthServiceImpl) DeactivateUser(ctx context.Context, userID string) error {
	user, err := s.GetUser(ctx, userID)
	if err != nil {
		return err
	}
	if user.Role == domain.RoleRoot || user.Role == domain.RoleAdmin {
		return errors.New("cannot deactivate root user")
	}

	if err := s.repository.DeactivateUser(ctx, userID, userID, reasonUserSelfDeactivated, true); err != nil {
		return fmt.Errorf("failed to deactivate user: %w", err)
	}

	if err := s.RevokeAllUserSessions(ctx, userID); err != nil {
		return fmt.Errorf("failed to revoke user sessions: %w", err)
	}

	return nil
}

func (s *AuthServiceImpl) ListUsers(ctx context.Context, limit, offset int) ([]domain.User, error) {
	schemaUsers, err := s.repository.ListUsers(ctx, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	return domain.MapUsersFromSchema(schemaUsers), nil
}

// enrichUserAvatar attaches avatar URL from the profile service when available.
func (s *AuthServiceImpl) enrichUserAvatar(ctx context.Context, user *domain.User) {
	if user == nil || s.profileHooks == nil {
		return
	}

	avatar, err := s.profileHooks.GetProfileAvatarURL(ctx, user.ID.String())
	if err != nil {
		s.log.Logf("WARN Auth: failed to fetch avatar for user %s: %v", user.ID, err)
		return
	}
	if avatar != nil {
		user.AvatarURL = avatar
	}
}
