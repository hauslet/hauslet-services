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

// Password authentication helpers

func (s *AuthServiceImpl) LinkPasswordIdentity(ctx context.Context, userID, email, password string) (*domain.UserIdentity, error) {
	if email == "" {
		return nil, domain.ErrEmailRequired
	}
	if password == "" {
		return nil, domain.ErrPasswordRequired
	}

	user, err := s.repository.GetUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, domain.ErrUserNotFound
	}
	if !user.IsActive {
		return nil, domain.ErrUserDeactivated
	}
	if user.PrimaryEmail != email {
		return nil, fmt.Errorf("email must match user's primary email")
	}

	existingIdentity, err := s.repository.GetUserIdentityByProvider(ctx, "password", user.ID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to check existing password identity: %w", err)
	}
	if existingIdentity != nil {
		return nil, errors.New("password identity already exists for this user")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	hashedPasswordStr := string(hashedPassword)
	identity := &schema.UserIdentity{
		ID:            uuid.New(),
		UserID:        user.ID,
		Provider:      "password",
		ProviderID:    user.ID.String(),
		Email:         email,
		EmailVerified: false,
		PasswordHash:  &hashedPasswordStr,
	}

	if err := s.repository.CreateUserIdentity(ctx, identity); err != nil {
		return nil, fmt.Errorf("failed to link password identity: %w", err)
	}

	return domain.MapUserIdentityFromSchema(identity), nil
}

func (s *AuthServiceImpl) CreatePasswordUser(ctx context.Context, email, password, name string, birthDate *time.Time) (*domain.User, error) {
	if email == "" {
		return nil, domain.ErrEmailRequired
	}
	if password == "" {
		return nil, domain.ErrPasswordRequired
	}
	if name == "" {
		return nil, domain.ErrNameRequired
	}

	existingUser, _ := s.repository.GetUserByEmail(ctx, email)
	if existingUser != nil {
		return nil, domain.ErrUserAlreadyExists
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user := &schema.User{
		ID:           uuid.New(),
		Name:         name,
		PrimaryEmail: email,
		Role:         schema.RoleUser,
		IsActive:     true,
	}

	hashedPasswordStr := string(hashedPassword)
	identity := &schema.UserIdentity{
		ID:            uuid.New(),
		UserID:        user.ID,
		Provider:      "password",
		ProviderID:    user.ID.String(),
		Email:         email,
		EmailVerified: false,
		PasswordHash:  &hashedPasswordStr,
	}

	if err := s.repository.CreateUserWithIdentity(ctx, user, identity); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	if s.profileHooks != nil {
		if err := s.profileHooks.CreateDefaultProfile(ctx, user.ID.String(), name, birthDate); err != nil {
			s.log.Logf("WARN failed to create default profile for user %s: %v", user.ID, err)
		}
	}

	return domain.MapUserFromSchema(user), nil
}

func (s *AuthServiceImpl) AuthenticatePassword(ctx context.Context, email, password string) (*domain.User, error) {
	user, err := s.repository.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}
	if user == nil {
		return nil, errors.New("invalid email or password")
	}
	if !user.IsActive {
		return nil, errors.New("user account is deactivated")
	}

	identities, err := s.repository.ListUserIdentitiesByUserID(ctx, user.ID.String())
	if err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

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

	if err := bcrypt.CompareHashAndPassword([]byte(*passwordIdentity.PasswordHash), []byte(password)); err != nil {
		return nil, errors.New("invalid email or password")
	}

	if !passwordIdentity.EmailVerified {
		return nil, errors.New("please verify your email before logging in")
	}

	_ = s.repository.UpdateUserLastLogin(ctx, user.ID.String())

	return domain.MapUserFromSchema(user), nil
}

func (s *AuthServiceImpl) ChangePassword(ctx context.Context, userID, oldPassword, newPassword string) error {
	user, err := s.repository.GetUserByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return errors.New("user not found")
	}

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

	if err := bcrypt.CompareHashAndPassword([]byte(*passwordIdentity.PasswordHash), []byte(oldPassword)); err != nil {
		return errors.New("invalid old password")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	hashedPasswordStr := string(hashedPassword)
	passwordIdentity.PasswordHash = &hashedPasswordStr

	if err := s.repository.UpdateUserIdentity(ctx, passwordIdentity); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	return nil
}
