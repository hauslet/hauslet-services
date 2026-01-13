package hooks

import (
	"context"
	"fmt"

	pricingservice "hauslet/internal/modules/pricing/service"
	"hauslet/internal/modules/property/domain"
	propertyService "hauslet/internal/modules/property/service"

	"github.com/google/uuid"
)

// PricingHooksAdapter implements pricing service ListingHooks.
type PricingHooksAdapter struct {
	*listingHookAdapter
}

// NewPricingHooksAdapter wires property data into the pricing service contract.
func NewPricingHooksAdapter(svc propertyService.PropertyService) *PricingHooksAdapter {
	return &PricingHooksAdapter{
		listingHookAdapter: newListingHookAdapter(svc),
	}
}

func (a *PricingHooksAdapter) GetListingPricing(ctx context.Context, listingID uuid.UUID) (*pricingservice.ListingPricing, error) {
	listing, err := a.getListing(ctx, listingID)
	if err != nil {
		return nil, err
	}

	if listing.ShortletDetails == nil {
		return nil, fmt.Errorf("listing %s does not have shortlet pricing details", listingID)
	}

	detail := listing.ShortletDetails

	// Convert domain CustomFees to pricing service CustomFees
	fees := make([]pricingservice.CustomFee, len(detail.Fees))
	for i, fee := range detail.Fees {
		fees[i] = pricingservice.CustomFee{
			Name:         fee.Name,
			Amount:       fee.Amount,
			Frequency:    string(fee.Frequency),
			Category:     string(fee.Category),
			IsRefundable: fee.IsRefundable,
			IsOptional:   fee.IsOptional,
		}
	}

	discount := make([]pricingservice.Discount, len(detail.Discounts))
	for i, d := range detail.Discounts {
		discount[i] = pricingservice.Discount{
			Name:       d.Name,
			Type:       pricingservice.DiscountType(d.Type),
			Percentage: d.Percentage,
			MinNights:  d.MinNights,
			Active:     d.Active,
		}
	}

	return &pricingservice.ListingPricing{
		ListingID:      listing.ID,
		Currency:       currencyOrDefault(listing.Currency),
		BaseRate:       detail.NightlyRate,
		Fees:           fees,
		Discounts:      discount,
		BaseGuestCount: detail.BaseGuestCount,
	}, nil
}

func currencyOrDefault(code domain.CurrencyCode) string {
	if code == "" {
		return defaultListingCurrency
	}
	return string(code)
}
