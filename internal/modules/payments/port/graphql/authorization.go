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
	ErrSupportRequired   = fmt.Errorf("forbidden: support, admin, or root access required")
	ErrOwnershipRequired = fmt.Errorf("forbidden: resource ownership or admin access required")
)

// isAdminRole checks if role is admin or root
func isAdminRole(role string) bool {
	return role == "admin" || role == "root"
}

// isSupportRole checks if role is support, admin, or root
func isSupportRole(role string) bool {
	return role == "support" || role == "admin" || role == "root"
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

// requireAdmin ensures user has admin or root role
func requireAdmin(ctx context.Context) error {
	v := viewer.FromContext(ctx)
	if v == nil || v.UserID == "" {
		return ErrUnauthorized
	}

	if !isAdminRole(v.Role) {
		return ErrAdminRequired
	}

	return nil
}

// requireSupport ensures user has support, admin, or root role
func requireSupport(ctx context.Context) error {
	v := viewer.FromContext(ctx)
	if v == nil || v.UserID == "" {
		return ErrUnauthorized
	}

	if !isSupportRole(v.Role) {
		return ErrSupportRequired
	}

	return nil
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
