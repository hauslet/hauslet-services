package service

import (
	"context"

	"hauslet/internal/modules/auth/authorization"
	"hauslet/internal/modules/property/domain"
	"hauslet/internal/platform/authz"

	"github.com/google/uuid"
)

func (s *ServiceImpl) authorizeSupplyAction(ctx context.Context, action authorization.SupplyAction, opts *authorization.SupplyOptions) error {
	if s.supplyGate == nil {
		return nil
	}

	actor := authz.FromContext(ctx)
	if actor == nil || actor.UserID == "" {
		return domain.ErrUnauthorized
	}

	userID, err := uuid.Parse(actor.UserID)
	if err != nil {
		return domain.ErrUnauthorized
	}

	return s.supplyGate.Authorize(ctx, userID, action, opts)
}
