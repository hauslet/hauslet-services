package graphql

import (
	"context"
	"fmt"
	"hauslet/internal/transport/graph/viewer"

	"github.com/google/uuid"
)

// Authorization errors
var (
	ErrUnauthorized      = fmt.Errorf("unauthorized: authentication required")
	ErrAdminRequired     = fmt.Errorf("forbidden: admin or root access required")
	ErrOwnershipRequired = fmt.Errorf("forbidden: resource ownership or admin access required")
)

// isAdminRole checks if role is admin or root
func isAdminRole(role string) bool {
	return role == "admin" || role == "root"
}

// getUserIDFromContext extracts user ID from context
func getUserIDFromContext(ctx context.Context) (uuid.UUID, error) {
	v := viewer.FromContext(ctx)
	if v == nil || v.UserID == "" {
		return uuid.Nil, ErrUnauthorized
	}

	userID, err := uuid.Parse(v.UserID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid user ID in context")
	}

	return userID, nil
}

// requireOwnershipOrAdmin ensures user owns the resource OR is an admin
func requireOwnershipOrAdmin(ctx context.Context, resourceOwnerID uuid.UUID) error {
	v := viewer.FromContext(ctx)
	if v == nil || v.UserID == "" {
		return ErrUnauthorized
	}

	// Check if admin/root
	if isAdminRole(v.Role) {
		return nil
	}

	// Check if owner
	userID, err := uuid.Parse(v.UserID)
	if err != nil {
		return fmt.Errorf("invalid user ID")
	}

	if userID == resourceOwnerID {
		return nil
	}

	return ErrOwnershipRequired
}
