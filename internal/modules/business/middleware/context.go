package middleware

import (
	"context"

	"hauslet/internal/modules/business/domain"

	"github.com/google/uuid"
)

type contextKey string

const (
	businessContextKey    contextKey = "business"
	membershipContextKey  contextKey = "business_membership"
	permissionsContextKey contextKey = "business_permissions"
)

// BusinessContext holds business-related context information
type BusinessContext struct {
	BusinessID  uuid.UUID
	Business    *domain.Business
	Membership  *domain.BusinessMember
	Permissions *domain.MemberPermissions
	IsOwner     bool
	IsAdmin     bool
}

// WithBusiness adds a business to the context
func WithBusiness(ctx context.Context, business *domain.Business) context.Context {
	return context.WithValue(ctx, businessContextKey, business)
}

// BusinessFromContext retrieves the business from context
func BusinessFromContext(ctx context.Context) (*domain.Business, bool) {
	business, ok := ctx.Value(businessContextKey).(*domain.Business)
	return business, ok
}

// WithMembership adds a business membership to the context
func WithMembership(ctx context.Context, membership *domain.BusinessMember) context.Context {
	return context.WithValue(ctx, membershipContextKey, membership)
}

// MembershipFromContext retrieves the membership from context
func MembershipFromContext(ctx context.Context) (*domain.BusinessMember, bool) {
	membership, ok := ctx.Value(membershipContextKey).(*domain.BusinessMember)
	return membership, ok
}

// WithPermissions adds business permissions to the context
func WithPermissions(ctx context.Context, permissions *domain.MemberPermissions) context.Context {
	return context.WithValue(ctx, permissionsContextKey, permissions)
}

// PermissionsFromContext retrieves the permissions from context
func PermissionsFromContext(ctx context.Context) (*domain.MemberPermissions, bool) {
	permissions, ok := ctx.Value(permissionsContextKey).(*domain.MemberPermissions)
	return permissions, ok
}

// WithBusinessContext adds a complete business context
func WithBusinessContext(ctx context.Context, bc *BusinessContext) context.Context {
	ctx = WithBusiness(ctx, bc.Business)
	if bc.Membership != nil {
		ctx = WithMembership(ctx, bc.Membership)
	}
	if bc.Permissions != nil {
		ctx = WithPermissions(ctx, bc.Permissions)
	}
	return ctx
}

// GetBusinessContext retrieves the complete business context
func GetBusinessContext(ctx context.Context) (*BusinessContext, bool) {
	business, hasBusiness := BusinessFromContext(ctx)
	if !hasBusiness {
		return nil, false
	}

	bc := &BusinessContext{
		Business:   business,
		BusinessID: business.ID,
	}

	if membership, ok := MembershipFromContext(ctx); ok {
		bc.Membership = membership
		bc.IsOwner = membership.IsOwner()
		bc.IsAdmin = membership.IsAdmin()
	}

	if permissions, ok := PermissionsFromContext(ctx); ok {
		bc.Permissions = permissions
	}

	return bc, true
}
