package hooks

import (
	"context"
	"fmt"
	bookingdomain "hauslet/internal/modules/booking/domain"
	"hauslet/internal/modules/booking/service"
	"hauslet/internal/modules/property/repository"
	"hauslet/internal/modules/property/repository/schema"
	"slices"
	"time"

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
	if err := listingBookingEligibilityError(listing); err != nil {
		return nil, err
	}

	// Extract shortlet details
	constraints := &service.ListingConstraints{
		ListingID: listing.ID,
		Timezone:  "Africa/Lagos", // Default timezone
	}

	if listing.ShortletDetails != nil {
		detail := listing.ShortletDetails

		// Basic constraints
		constraints.MinNights = detail.StayLimits.MinNights
		constraints.MaxNights = detail.StayLimits.MaxNights
		constraints.MaxGuests = detail.MaxGuests
		constraints.CheckInTime = detail.CheckInTime
		constraints.CheckOutTime = detail.CheckOutTime
		// Update to use BookingSettings.ApprovalMethod instead of AutoAcceptBookings
		constraints.AutoAcceptBookings = detail.BookingSettings.ApprovalMethod == "instant"

		// Extract refund policy from rule groups
		constraints.RefundPolicy = a.extractRefundPolicy(detail.Rules)

		// Convert custom fees
		if len(detail.Fees) > 0 {
			constraints.Fees = make([]service.CustomFee, len(detail.Fees))
			for i, fee := range detail.Fees {
				constraints.Fees[i] = service.CustomFee{
					Name:         fee.Name,
					Amount:       fee.Amount,
					Frequency:    string(fee.Frequency),
					Category:     string(fee.Category),
					IsRefundable: fee.IsRefundable,
					IsOptional:   fee.IsOptional,
				}
			}
		}

		// Convert discounts
		if len(detail.Discounts) > 0 {
			constraints.Discounts = make([]service.Discount, len(detail.Discounts))
			for i, discount := range detail.Discounts {
				constraints.Discounts[i] = service.Discount{
					Name:       discount.Name,
					Type:       string(discount.Type),
					Percentage: discount.Percentage,
					MinNights:  discount.MinNights,
					Active:     discount.Active,
				}
			}
		}

		// Convert booking settings
		constraints.BookingSettings = &service.BookingSettings{
			ApprovalMethod:       string(detail.BookingSettings.ApprovalMethod),
			VerifiedID:           detail.BookingSettings.GuestRequirements.VerifiedID,
			PositiveReviewsOnly:  detail.BookingSettings.GuestRequirements.PositiveReviewsOnly,
			ProfilePhotoRequired: detail.BookingSettings.GuestRequirements.ProfilePhotoRequired,
			PreBookingMessage:    detail.BookingSettings.PreBookingMessage,
		}

		// Convert advance booking settings
		constraints.AdvanceBooking = &service.AdvanceBooking{
			MonthsAhead:    detail.AdvanceBooking.MonthsAhead,
			MinNoticeHours: detail.AdvanceBooking.MinNoticeHours,
		}
	}

	// Set currency
	if listing.Currency != "" {
		constraints.Currency = string(listing.Currency) // Convert CurrencyCode to string
	} else {
		constraints.Currency = "NGN" // Default
	}

	return constraints, nil
}

func listingBookingEligibilityError(listing *schema.Listing) error {
	if listing == nil {
		return fmt.Errorf("listing not found")
	}

	now := time.Now()
	if listing.Status == schema.StatusSuspended {
		return bookingdomain.ErrListingSuspended
	}
	if listing.SuspendedUntil != nil && listing.SuspendedUntil.After(now) {
		return bookingdomain.ErrListingSuspended
	}

	if !listing.Published || listing.Status != schema.StatusActive {
		return bookingdomain.ErrListingUnavailable
	}

	if listing.ListingType != schema.ListingShortLet || listing.ShortletDetails == nil {
		return bookingdomain.ErrListingUnavailable
	}

	return nil
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
