package service

import (
	"context"
	"fmt"
	"strings"

	"hauslet/internal/modules/profile/domain"

	"github.com/google/uuid"
)

var supplyUserTypeSet = map[domain.UserType]struct{}{
	domain.Host:     {},
	domain.Agent:    {},
	domain.LandLord: {},
	domain.CoHost:   {},
}

// SelectSupplyRoles assigns the selected supply-side roles while preserving non-supply roles.
func (s *ProfileServiceImpl) SelectSupplyRoles(ctx context.Context, userID string, userTypes []domain.UserType) (*domain.Profile, error) {
	if userID == "" {
		return nil, domain.ErrInvalidUserID
	}
	if len(userTypes) == 0 {
		return nil, domain.ErrSupplyRoleRequired
	}

	profile, err := s.ensureProfile(ctx, userID)
	if err != nil {
		return nil, err
	}

	normalized, err := normalizeSupplyUserTypes(userTypes)
	if err != nil {
		return nil, err
	}

	merged := make([]domain.UserType, 0, len(profile.UserTypes)+len(normalized))
	for _, existing := range profile.UserTypes {
		if !isSupplyUserType(existing) {
			merged = append(merged, existing)
		}
	}
	merged = append(merged, normalized...)
	if len(merged) == 0 {
		merged = []domain.UserType{domain.Guest}
	}

	profile.UserTypes = merged

	schemaProfile, err := domain.MapProfileToSchema(profile)
	if err != nil {
		return nil, err
	}

	if err := s.repo.UpdateProfile(ctx, schemaProfile); err != nil {
		s.log.Error("failed to update supply roles", "user_id", userID, "error", err)
		return nil, err
	}

	// Auto-create free subscription if adapter is available
	if s.subscriptionAdapter != nil {
		userUUID, err := uuid.Parse(userID)
		if err == nil {
			if err := s.subscriptionAdapter.GetOrCreateFreeSubscription(ctx, userUUID); err != nil {
				// Log error but don't fail - subscription creation is best-effort
				s.log.Warn("failed to auto-create free subscription",
					"user_id", userID,
					"error", err,
				)
			} else {
				s.log.Info("auto-created free subscription for supply role",
					"user_id", userID,
				)
			}
		} else {
			s.log.Error("invalid user ID format for subscription creation",
				"user_id", userID,
				"error", err,
			)
		}
	}

	s.log.Info("updated supply roles", "user_id", userID, "roles", normalized)
	return domain.MapProfileFromSchema(schemaProfile), nil
}

func normalizeSupplyUserTypes(userTypes []domain.UserType) ([]domain.UserType, error) {
	normalized := make([]domain.UserType, 0, len(userTypes))
	seen := make(map[domain.UserType]struct{}, len(userTypes))

	for _, t := range userTypes {
		role := domain.UserType(strings.ToLower(strings.TrimSpace(string(t))))
		if !isSupplyUserType(role) {
			return nil, fmt.Errorf("%w: %s", domain.ErrInvalidUserType, t)
		}
		if _, ok := seen[role]; ok {
			continue
		}
		seen[role] = struct{}{}
		normalized = append(normalized, role)
	}

	if len(normalized) == 0 {
		return nil, domain.ErrSupplyRoleRequired
	}

	return normalized, nil
}

func isSupplyUserType(userType domain.UserType) bool {
	_, ok := supplyUserTypeSet[userType]
	return ok
}
