package hooks

import (
	"context"
	"fmt"
	"hauslet/internal/modules/booking/service"
	"hauslet/internal/modules/property/repository"
	"hauslet/internal/modules/property/repository/schema"
	"slices"

	"github.com/google/uuid"
)

// PropertyHooksAdapter adapts property repository for booking's ListingHooks interface.
type PropertyHooksAdapter struct {
	propertyRepo repository.Repository
}

// NewPropertyHooksAdapter creates a new property hooks adapter for booking.
func NewPropertyHooksAdapter(propertyRepo repository.Repository) service.ListingHooks {
	return &PropertyHooksAdapter{
		propertyRepo: propertyRepo,
	}
}

// GetListingOwner retrieves the owner ID of a listing.
func (a *PropertyHooksAdapter) GetListingOwner(ctx context.Context, listingID uuid.UUID) (uuid.UUID, error) {
	listing, err := a.propertyRepo.GetListingByID(ctx, listingID, false)
	if err != nil {
		return uuid.Nil, err
	}
	if listing == nil {
		return uuid.Nil, fmt.Errorf("listing not found")
	}
	return listing.OwnerID, nil
}

// GetListingConstraints retrieves booking constraints for a listing.
func (a *PropertyHooksAdapter) GetListingConstraints(ctx context.Context, listingID uuid.UUID) (*service.ListingConstraints, error) {
	listing, err := a.propertyRepo.GetListingByID(ctx, listingID, false)
	if err != nil {
		return nil, err
	}
	if listing == nil {
		return nil, fmt.Errorf("listing not found")
	}

	// Extract shortlet details
	constraints := &service.ListingConstraints{
		ListingID: listing.ID,
		Timezone:  "Africa/Lagos", // Default timezone
	}

	if listing.ShortletDetails != nil {
		constraints.MinNights = listing.ShortletDetails.MinNights
		constraints.MaxNights = listing.ShortletDetails.MaxNights
		constraints.MaxGuests = listing.ShortletDetails.MaxGuests
		constraints.CheckInTime = listing.ShortletDetails.CheckInTime
		constraints.CheckOutTime = listing.ShortletDetails.CheckOutTime
		constraints.AutoAcceptBookings = listing.ShortletDetails.AutoAcceptBookings

		// Extract refund policy from rule groups
		constraints.RefundPolicy = a.extractRefundPolicy(listing.ShortletDetails.Rules)
	}

	// Set currency
	if listing.Currency != "" {
		constraints.Currency = string(listing.Currency) // Convert CurrencyCode to string
	} else {
		constraints.Currency = "NGN" // Default
	}

	return constraints, nil
}

// extractRefundPolicy extracts the refund policy type from rule groups.
// It looks for the cancellation_policy category and the refund_policy rule item.
// Returns the policy type (flexible, moderate, strict, long_term) or "moderate" as default.
func (a *PropertyHooksAdapter) extractRefundPolicy(ruleGroups []schema.RuleGroup) string {
	// Default policy
	const defaultPolicy = "moderate"

	// Find cancellation_policy category
	for _, group := range ruleGroups {
		if group.Category == "cancellation_policy" {
			// Find refund_policy rule item
			for _, rule := range group.Rules {
				if rule.Name == "refund_policy" {
					// Extract type from description map
					if rule.Description != nil {
						if policyType, ok := rule.Description["type"].(string); ok && policyType != "" {
							// Validate against known policy tiers
							if a.isValidPolicyTier(policyType) {
								return policyType
							}
						}
					}
				}
			}
		}
	}

	return defaultPolicy
}

// isValidPolicyTier checks if the policy tier is valid.
// Valid tiers match the policy_tiers defined in config/defaults/platform.yaml
func (a *PropertyHooksAdapter) isValidPolicyTier(tier string) bool {
	validTiers := []string{"flexible", "moderate", "strict", "long_term"}
	return slices.Contains(validTiers, tier)
}
