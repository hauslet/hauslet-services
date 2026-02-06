package authorization

import (
	"context"
	"fmt"
	"slices"
	"strings"

	profiledomain "hauslet/internal/modules/profile/domain"
	profileservice "hauslet/internal/modules/profile/service"
	promotionservice "hauslet/internal/modules/promotions/service"
	"hauslet/internal/platform/authz"

	"github.com/google/uuid"
)

// SupplyAction represents a supply-side action that requires gating.
type SupplyAction string

const (
	SupplyActionCreateListing  SupplyAction = "create_listing"
	SupplyActionCreateBusiness SupplyAction = "create_business"
	SupplyActionPublishListing SupplyAction = "publish_listing"
	SupplyActionCreateViewing  SupplyAction = "create_viewing"
	SupplyActionManageBookings SupplyAction = "manage_bookings"
)

// SupplyOptions carries action-specific attributes for supply checks.
type SupplyOptions struct {
	ListingType string
}

// SupplyGate enforces supply-side eligibility rules.
type SupplyGate interface {
	Authorize(ctx context.Context, userID uuid.UUID, action SupplyAction, opts *SupplyOptions) error
}

// SupplyGateImpl implements SupplyGate.
type SupplyGateImpl struct {
	profiles      profileservice.ProfileService
	subscriptions promotionservice.SubscriptionService
	bypassRoleSet map[string]struct{}
	supplyTypeSet map[profiledomain.UserType]struct{}
}

// NewSupplyGate creates a supply-side gate with explicit bypass roles.
func NewSupplyGate(
	profileSvc profileservice.ProfileService,
	subscriptionSvc promotionservice.SubscriptionService,
	bypassRoles []string,
) *SupplyGateImpl {
	roleSet := make(map[string]struct{}, len(bypassRoles))
	for _, role := range bypassRoles {
		normalized := strings.ToLower(strings.TrimSpace(role))
		if normalized == "" {
			continue
		}
		roleSet[normalized] = struct{}{}
	}

	supplyTypes := map[profiledomain.UserType]struct{}{
		profiledomain.Host:     {},
		profiledomain.Agent:    {},
		profiledomain.LandLord: {},
		profiledomain.CoHost:   {},
	}

	return &SupplyGateImpl{
		profiles:      profileSvc,
		subscriptions: subscriptionSvc,
		bypassRoleSet: roleSet,
		supplyTypeSet: supplyTypes,
	}
}

// Authorize enforces the supply-side access policy for the given user and action.
func (g *SupplyGateImpl) Authorize(ctx context.Context, userID uuid.UUID, action SupplyAction, opts *SupplyOptions) error {
	if userID == uuid.Nil {
		return ErrSupplyUnauthorized
	}

	if g.isBypassRole(ctx) {
		return nil
	}

	if g.profiles == nil || g.subscriptions == nil {
		return ErrSupplyUnauthorized
	}

	profile, err := g.profiles.GetProfileByUserID(ctx, userID.String())
	if err != nil {
		return fmt.Errorf("fetch profile: %w", err)
	}
	if profile == nil {
		return ErrSupplyProfileRequired
	}

	if !g.hasSupplyType(profile.UserTypes) {
		return ErrSupplyUserTypeRequired
	}

	if !profile.IDVerified {
		return ErrSupplyVerificationRequired
	}

	if g.isShortletHostBypass(action, opts, profile.UserTypes) {
		return nil
	}

	subscription, err := g.subscriptions.GetUserSubscription(ctx, userID)
	if err != nil {
		return fmt.Errorf("fetch subscription: %w", err)
	}
	if subscription == nil {
		return ErrSupplySubscriptionRequired
	}

	return nil
}

func (g *SupplyGateImpl) hasSupplyType(types []profiledomain.UserType) bool {
	for _, t := range types {
		if _, ok := g.supplyTypeSet[t]; ok {
			return true
		}
	}
	return false
}

func (g *SupplyGateImpl) hasSpecificType(types []profiledomain.UserType, target profiledomain.UserType) bool {
	return slices.Contains(types, target)
}

func (g *SupplyGateImpl) isShortletHostBypass(action SupplyAction, opts *SupplyOptions, types []profiledomain.UserType) bool {
	if opts == nil {
		return false
	}
	switch action {
	case SupplyActionCreateListing, SupplyActionPublishListing:
		// allow
	default:
		return false
	}
	listingType := strings.ToLower(strings.TrimSpace(opts.ListingType))
	if listingType != "shortlet" {
		return false
	}
	return g.hasSpecificType(types, profiledomain.Host)
}

func (g *SupplyGateImpl) isBypassRole(ctx context.Context) bool {
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
