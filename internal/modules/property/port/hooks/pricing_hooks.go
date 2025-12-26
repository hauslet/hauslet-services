package hooks

import (
	"context"
	"fmt"
	pricingservice "hauslet/internal/modules/pricing/service"
	"hauslet/internal/modules/property/domain"
	listingService "hauslet/internal/modules/property/service"

	"github.com/google/uuid"
)

// PricingHooksAdapter implements pricing service ListingHooks.
type PricingHooksAdapter struct {
	*listingHookAdapter
}

// NewPricingHooksAdapter wires property data into the pricing service contract.
func NewPricingHooksAdapter(svc listingService.Service) *PricingHooksAdapter {
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
	return &pricingservice.ListingPricing{
		ListingID:      listing.ID,
		Currency:       currencyOrDefault(listing.Currency),
		BaseRate:       detail.NightlyRate,
		CleaningFee:    detail.CleaningFee,
		ServiceFee:     detail.ServiceFee,
		CautionFee:     detail.CautionFee,
		ExtraGuestFee:  detail.ExtraGuestFee,
		BaseGuestCount: detail.BaseGuestCount,
	}, nil
}

func currencyOrDefault(code domain.CurrencyCode) string {
	if code == "" {
		return defaultListingCurrency
	}
	return string(code)
}
