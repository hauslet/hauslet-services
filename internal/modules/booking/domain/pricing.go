package domain

import (
	pricingdomain "hauslet/internal/modules/pricing/domain"
)

// NewPriceBreakdownSnapshotFromPricing converts pricing domain object into booking snapshot.
func NewPriceBreakdownSnapshotFromPricing(src *pricingdomain.PriceBreakdown) *PriceBreakdownSnapshot {
	if src == nil {
		return nil
	}

	snapshot := &PriceBreakdownSnapshot{
		BaseTotal:     src.BaseTotal,
		CleaningFee:   src.CleaningFee,
		ServiceFee:    src.ServiceFee,
		CautionFee:    src.CautionFee,
		ExtraGuestFee: src.ExtraGuestFee,
		Subtotal:      src.Subtotal,
		VATPercent:    src.VATPercent,
		VATAmount:     src.VATAmount,
		Total:         src.Total,
		Currency:      src.Currency,
	}

	if len(src.Discounts) > 0 {
		snapshot.Discounts = make([]DiscountSnapshot, len(src.Discounts))
		for i, discount := range src.Discounts {
			snapshot.Discounts[i] = DiscountSnapshot{
				Name:   discount.Name,
				Amount: discount.Amount,
				Type:   discount.Type,
			}
		}
	}

	if len(src.DailyRates) > 0 {
		snapshot.NightlyRates = make([]DailyRate, len(src.DailyRates))
		for i, rate := range src.DailyRates {
			snapshot.NightlyRates[i] = DailyRate{
				Date:      rate.Date.Format("2006-01-02"),
				BaseRate:  rate.BaseRate,
				FinalRate: rate.FinalRate,
			}
		}
	}

	if len(src.Fees) > 0 {
		snapshot.Fees = make([]FeeSnapshot, len(src.Fees))
		for i, fee := range src.Fees {
			snapshot.Fees[i] = FeeSnapshot{
				Name:         fee.Name,
				Frequency:    fee.Frequency,
				Category:     fee.Category,
				Amount:       fee.Amount,
				Total:        fee.Total,
				IsOptional:   fee.IsOptional,
				IsRefundable: fee.IsRefundable,
			}
		}
	}

	if src.PlatformFees != nil {
		snapshot.PlatformFees = &PlatformFeeBreakdown{
			GuestFeePercent:         src.PlatformFees.GuestFeePercent,
			GuestFeeAmount:          src.PlatformFees.GuestFeeAmount,
			HostCommissionPercent:   src.PlatformFees.HostCommissionPercent,
			HostCommissionAmount:    src.PlatformFees.HostCommissionAmount,
			PayoutProcessingPercent: src.PlatformFees.PayoutProcessingPercent,
			PayoutProcessingAmount:  src.PlatformFees.PayoutProcessingAmount,
			MinimumGuestFeeApplied:  src.PlatformFees.MinimumGuestFeeApplied,
			HostNetAmount:           src.PlatformFees.HostNetAmount,
		}
	}

	return snapshot
}
