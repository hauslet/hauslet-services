package service

import (
	"hauslet/config"
	"math"
)

// Helper methods for platform fee calculations

func (s *PricingServiceImpl) isFeesEnabled() bool {
	return s.platformConfig.Fees.HostCommissionPercent > 0 ||
		s.platformConfig.Fees.GuestServicePercent > 0 ||
		s.platformConfig.Fees.PayoutProcessingPercent > 0 ||
		s.platformConfig.Fees.MinimumServiceFeeMinor > 0
}

func (s *PricingServiceImpl) minorUnitDivisor() float64 {
	if s.platformConfig.Currency.MinorUnit > 0 {
		return float64(s.platformConfig.Currency.MinorUnit)
	}
	return 100.0
}

func (s *PricingServiceImpl) minGuestFeeAmount() float64 {
	if s.platformConfig.Fees.MinimumServiceFeeMinor <= 0 {
		return 0
	}
	return float64(s.platformConfig.Fees.MinimumServiceFeeMinor) / s.minorUnitDivisor()
}

func (s *PricingServiceImpl) calculateGuestServiceFee(base float64) (amount float64, minApplied bool) {
	if base <= 0 || s.platformConfig.Fees.GuestServicePercent <= 0 {
		return 0, false
	}

	amount = base * (s.platformConfig.Fees.GuestServicePercent / 100)

	minFee := s.minGuestFeeAmount()
	if minFee > 0 && amount < minFee {
		return minFee, true
	}

	if s.platformConfig.Fees.MaximumServiceFeePercent > 0 {
		maxFee := base * (s.platformConfig.Fees.MaximumServiceFeePercent / 100)
		if maxFee > 0 {
			amount = math.Min(amount, maxFee)
		}
	}

	return amount, false
}

func (s *PricingServiceImpl) calculateHostCommission(base float64) float64 {
	if base <= 0 || s.platformConfig.Fees.HostCommissionPercent <= 0 {
		return 0
	}
	return base * (s.platformConfig.Fees.HostCommissionPercent / 100)
}

func (s *PricingServiceImpl) calculatePayoutProcessing(amount float64) float64 {
	if amount <= 0 || s.platformConfig.Fees.PayoutProcessingPercent <= 0 {
		return 0
	}
	return amount * (s.platformConfig.Fees.PayoutProcessingPercent / 100)
}

// Legacy helper for backward compatibility with config types
func platformFeesFromConfig(cfg config.PlatformYAMLConfig) struct {
	HostCommissionPercent    float64
	GuestServicePercent      float64
	PayoutProcessingPercent  float64
	MinimumServiceFeeMinor   int64
	MaximumServiceFeePercent float64
	CurrencyMinorUnit        int64
} {
	return struct {
		HostCommissionPercent    float64
		GuestServicePercent      float64
		PayoutProcessingPercent  float64
		MinimumServiceFeeMinor   int64
		MaximumServiceFeePercent float64
		CurrencyMinorUnit        int64
	}{
		HostCommissionPercent:    cfg.Fees.HostCommissionPercent,
		GuestServicePercent:      cfg.Fees.GuestServicePercent,
		PayoutProcessingPercent:  cfg.Fees.PayoutProcessingPercent,
		MinimumServiceFeeMinor:   cfg.Fees.MinimumServiceFeeMinor,
		MaximumServiceFeePercent: cfg.Fees.MaximumServiceFeePercent,
		CurrencyMinorUnit:        cfg.Currency.MinorUnit,
	}
}
