package service

import (
	"context"
	"hauslet/internal/modules/booking/domain"
	"hauslet/internal/platform/xchange"
	"hauslet/internal/transport/middleware/localization"
	"math"
	"strings"
)

func (s *BookingServiceImpl) LocalizeBooking(ctx context.Context, booking *domain.Booking) {
	if booking == nil || s.fx == nil {
		return
	}

	target, ok := localization.PreferredCurrency(ctx)
	if !ok {
		return
	}
	target = xchange.NormalizeCurrency(target)

	source := booking.Currency
	if source == "" {
		source = xchange.NormalizeCurrency("")
	}
	if strings.EqualFold(source, target) {
		return
	}

	rate, err := s.fx.GetExchangeRate(source, target)
	if err != nil {
		if s.log != nil {
			s.log.Warn("failed to convert booking", "booking_id", booking.ID, "from", source, "to", target, "error", err)
		}
		return
	}

	convert := func(value float64) float64 {
		return math.Round(value*rate*100) / 100
	}

	booking.TotalPrice = convert(booking.TotalPrice)
	s.localizePriceBreakdownSnapshot(booking.PriceBreakdown, convert, target)
	booking.Currency = target
}

func (s *BookingServiceImpl) LocalizeQuote(ctx context.Context, quote *domain.BookingQuote) {
	if quote == nil || s.fx == nil {
		return
	}

	target, ok := localization.PreferredCurrency(ctx)
	if !ok {
		return
	}
	target = xchange.NormalizeCurrency(target)

	source := quote.Currency
	if source == "" {
		source = xchange.NormalizeCurrency("")
	}
	if strings.EqualFold(source, target) {
		return
	}

	rate, err := s.fx.GetExchangeRate(source, target)
	if err != nil {
		if s.log != nil {
			s.log.Warn("failed to convert booking quote", "listing_id", quote.ListingID, "from", source, "to", target, "error", err)
		}
		return
	}

	convert := func(value float64) float64 {
		return math.Round(value*rate*100) / 100
	}

	quote.TotalPrice = convert(quote.TotalPrice)
	s.localizePriceBreakdownSnapshot(quote.PriceBreakdown, convert, target)
	quote.Currency = target
}

func (s *BookingServiceImpl) localizePriceBreakdownSnapshot(breakdown *domain.PriceBreakdownSnapshot, convert func(float64) float64, currency string) {
	if breakdown == nil {
		return
	}

	breakdown.BaseTotal = convert(breakdown.BaseTotal)
	breakdown.ExtraGuestFee = convert(breakdown.ExtraGuestFee)
	breakdown.Subtotal = convert(breakdown.Subtotal)
	breakdown.Total = convert(breakdown.Total)
	breakdown.Currency = currency

	if breakdown.CleaningFee != nil {
		val := convert(*breakdown.CleaningFee)
		breakdown.CleaningFee = &val
	}
	if breakdown.ServiceFee != nil {
		val := convert(*breakdown.ServiceFee)
		breakdown.ServiceFee = &val
	}
	if breakdown.CautionFee != nil {
		val := convert(*breakdown.CautionFee)
		breakdown.CautionFee = &val
	}

	for i := range breakdown.Discounts {
		breakdown.Discounts[i].Amount = convert(breakdown.Discounts[i].Amount)
	}

	for i := range breakdown.NightlyRates {
		breakdown.NightlyRates[i].BaseRate = convert(breakdown.NightlyRates[i].BaseRate)
		breakdown.NightlyRates[i].FinalRate = convert(breakdown.NightlyRates[i].FinalRate)
	}

	if breakdown.PlatformFees != nil {
		breakdown.PlatformFees.GuestFeeAmount = convert(breakdown.PlatformFees.GuestFeeAmount)
		breakdown.PlatformFees.HostCommissionAmount = convert(breakdown.PlatformFees.HostCommissionAmount)
		breakdown.PlatformFees.PayoutProcessingAmount = convert(breakdown.PlatformFees.PayoutProcessingAmount)
		breakdown.PlatformFees.HostNetAmount = convert(breakdown.PlatformFees.HostNetAmount)
	}
}
