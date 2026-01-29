package service

import (
	"context"
	"math"
	"strings"

	"hauslet/internal/modules/property/domain"
	"hauslet/internal/platform/xchange"
	localization "hauslet/internal/transport/middleware/localization"
)

// LocalizeListing converts listing prices to the viewer's preferred currency when possible.
func (s *ServiceImpl) LocalizeListing(ctx context.Context, listing *domain.Listing) {
	if listing == nil || s.fx == nil {
		return
	}

	target, ok := localization.PreferredCurrency(ctx)
	if !ok {
		return
	}
	target = xchange.NormalizeCurrency(target)

	source := string(listing.Currency)
	if source == "" {
		source = xchange.NormalizeCurrency("")
	}
	if strings.EqualFold(source, target) {
		return
	}

	rate, err := s.fx.GetExchangeRate(source, target)
	if err != nil {
		if s.log != nil {
			s.log.Warn("failed to convert listing currency", "listing_id", listing.ID, "source_currency", source, "target_currency", target, "error", err)
		}
		return
	}

	convert := func(value float64) float64 {
		return math.Round(value*rate*100) / 100
	}

	if detail := listing.ShortletDetails; detail != nil {
		detail.NightlyRate = convert(detail.NightlyRate)
		for i := range detail.Fees {
			detail.Fees[i].Amount = convert(detail.Fees[i].Amount)
		}
	}

	if detail := listing.RentalDetails; detail != nil {
		detail.RentalPrice = convert(detail.RentalPrice)
		for i := range detail.Fees {
			detail.Fees[i].Amount = convert(detail.Fees[i].Amount)
		}
	}

	if detail := listing.SaleDetails; detail != nil {
		detail.SalePrice = convert(detail.SalePrice)
		for i := range detail.Fees {
			detail.Fees[i].Amount = convert(detail.Fees[i].Amount)
		}
	}

	listing.Currency = domain.CurrencyCode(target)
}
