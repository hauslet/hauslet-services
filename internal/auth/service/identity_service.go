package service

import (
	"context"
	"fmt"

	"hauslet/internal/auth/domain"
)

// Identity management

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

func (s *AuthServiceImpl) UnlinkIdentity(ctx context.Context, identityID string) error {
	identity, err := s.repository.GetUserIdentityByID(ctx, identityID)
	if err != nil {
		return fmt.Errorf("failed to get identity: %w", err)
	}
	if identity == nil {
		return fmt.Errorf("identity not found")
	}

	userIdentities, err := s.repository.ListUserIdentitiesByUserID(ctx, identity.UserID.String())
	if err != nil {
		return fmt.Errorf("failed to get user identities: %w", err)
	}
	if len(userIdentities) <= 1 {
		return fmt.Errorf("cannot unlink the last identity: user must have at least one login method")
	}

	if err := s.repository.DeleteUserIdentity(ctx, identityID); err != nil {
		return fmt.Errorf("failed to unlink identity: %w", err)
	}
	return nil
}

func (s *AuthServiceImpl) UpdateIdentityVerified(ctx context.Context, email string) error {
	user, err := s.repository.GetUserByEmail(ctx, email)
	if err != nil {
		return fmt.Errorf("failed to get user by email: %w", err)
	}
	if user == nil {
		return fmt.Errorf("user not found")
	}

	identities, err := s.repository.ListUserIdentitiesByUserID(ctx, user.ID.String())
	if err != nil {
		return fmt.Errorf("failed to get user identities: %w", err)
	}

	for _, identity := range identities {
		if identity.Provider == "password" && identity.Email == email {
			identity.EmailVerified = true
			if err := s.repository.UpdateUserIdentity(ctx, &identity); err != nil {
				return fmt.Errorf("failed to update identity: %w", err)
			}
			s.log.Logf("INFO Email verified for identity %s (user: %s)", identity.ID, email)
			return nil
		}
	}

	return fmt.Errorf("password identity not found for email: %s", email)
}
