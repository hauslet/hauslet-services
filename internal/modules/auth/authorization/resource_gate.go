package authorization

import (
	"context"
	"fmt"
	"strings"

	profiledomain "hauslet/internal/modules/profile/domain"
	profileservice "hauslet/internal/modules/profile/service"
	"hauslet/internal/platform/authz"

	"github.com/google/uuid"
)

// ResourceAction represents a resource access action that requires gating.
type ResourceAction string

const (
	ResourceActionAccess ResourceAction = "access_resource"
)

// VerificationLevel defines the minimum verification level for a resource.
type VerificationLevel string

const (
	VerificationLevelBasic    VerificationLevel = "basic"
	VerificationLevelIdentity VerificationLevel = "identity"
	VerificationLevelTrusted  VerificationLevel = "trusted"
)

// ResourceOptions carries action-specific attributes for resource checks.
type ResourceOptions struct {
	RequireEmail           bool
	RequirePhone           bool
	RequirePhoneVerified   bool
	RequireIDVerified      bool
	MinVerificationLevel   VerificationLevel
}

// ResourceGate enforces access rules for shared resources.
type ResourceGate interface {
	Authorize(ctx context.Context, userID uuid.UUID, action ResourceAction, opts *ResourceOptions) error
}

// ResourceGateImpl implements ResourceGate.
type ResourceGateImpl struct {
	profiles      profileservice.ProfileService
	bypassRoleSet map[string]struct{}
}

// NewResourceGate creates a resource gate with explicit bypass roles.
func NewResourceGate(profileSvc profileservice.ProfileService, bypassRoles []string) *ResourceGateImpl {
	roleSet := make(map[string]struct{}, len(bypassRoles))
	for _, role := range bypassRoles {
		normalized := strings.ToLower(strings.TrimSpace(role))
		if normalized == "" {
			continue
		}
		roleSet[normalized] = struct{}{}
	}

	return &ResourceGateImpl{
		profiles:      profileSvc,
		bypassRoleSet: roleSet,
	}
}

// Authorize enforces access policy for the given user and resource options.
func (g *ResourceGateImpl) Authorize(ctx context.Context, userID uuid.UUID, _ ResourceAction, opts *ResourceOptions) error {
	if userID == uuid.Nil {
		return ErrResourceUnauthorized
	}

	if g.isBypassRole(ctx) {
		return nil
	}

	if g.profiles == nil {
		return ErrResourceUnauthorized
	}

	profile, err := g.profiles.GetProfileByUserID(ctx, userID.String())
	if err != nil {
		return fmt.Errorf("fetch profile: %w", err)
	}
	if profile == nil {
		return ErrResourceProfileRequired
	}

	if opts == nil {
		return nil
	}

	if opts.RequireEmail {
		if profile.Email == nil || strings.TrimSpace(*profile.Email) == "" {
			return ErrResourceEmailRequired
		}
	}

	if opts.RequirePhone && len(profile.PhoneNumbers) == 0 {
		return ErrResourcePhoneRequired
	}

	if opts.RequirePhoneVerified && !profile.PhoneVerified {
		return ErrResourcePhoneVerificationRequired
	}

	if opts.RequireIDVerified && !profile.IDVerified {
		return ErrResourceIDVerificationRequired
	}

	if opts.MinVerificationLevel != "" {
		requiredRank := verificationRank(string(opts.MinVerificationLevel))
		if requiredRank < 0 {
			return ErrResourceVerificationLevelRequired
		}
		current := effectiveVerificationLevel(profile)
		if verificationRank(current) < requiredRank {
			return ErrResourceVerificationLevelRequired
		}
	}

	return nil
}

func effectiveVerificationLevel(profile *profiledomain.Profile) string {
	if profile.VerificationLevel != "" {
		return profile.VerificationLevel
	}
	switch {
	case profile.IDVerified && profile.PhoneVerified:
		return string(VerificationLevelTrusted)
	case profile.IDVerified:
		return string(VerificationLevelIdentity)
	case profile.PhoneVerified:
		return string(VerificationLevelBasic)
	default:
		return ""
	}
}

func verificationRank(level string) int {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "":
		return 0
	case string(VerificationLevelBasic):
		return 1
	case string(VerificationLevelIdentity):
		return 2
	case string(VerificationLevelTrusted):
		return 3
	default:
		return -1
	}
}   

func (g *ResourceGateImpl) isBypassRole(ctx context.Context) bool {
	actor := authz.FromContext(ctx)
	if actor == nil {
		return false
	}
	role := strings.ToLower(strings.TrimSpace(actor.Role))
	if role == "" {
		return false
	}
	_, ok := g.bypassRoleSet[role]
	return ok
}
