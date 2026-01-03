package domain

import (
	"hauslet/internal/modules/booking/repository/schema"
)

// MapBookingFromSchema converts schema entity to domain entity.
func MapBookingFromSchema(s *schema.Booking) *Booking {
	if s == nil {
		return nil
	}

	d := &Booking{
		ID:                s.ID,
		BookingReference:  s.BookingReference,
		ListingID:         s.ListingID,
		CalendarEventID:   s.CalendarEventID,
		CleaningEventID:   s.CleaningEventID,
		GuestID:           s.GuestID,
		GuestName:         s.GuestName,
		GuestEmail:        s.GuestEmail,
		GuestPhone:        s.GuestPhone,
		GuestCount:        s.GuestCount,
		Status:            BookingStatus(s.Status),
		BookingType:       BookingType(s.BookingType),
		CheckIn:           s.CheckIn,
		CheckOut:          s.CheckOut,
		CheckInTime:       s.CheckInTime,
		CheckOutTime:      s.CheckOutTime,
		HoldExpiresAt:     s.HoldExpiresAt,
		PaymentDueAt:      s.PaymentDueAt,
		ActiveAt:          s.ActiveAt,
		CompletedAt:       s.CompletedAt,
		ArchivedAt:        s.ArchivedAt,
		PaymentReference:  s.PaymentReference,
		LastPaymentID:     s.LastPaymentID,
		RefundAmount:      s.RefundAmount,
		RefundInitiatedAt: s.RefundInitiatedAt,
		RefundProcessedAt: s.RefundProcessedAt,
		RefundReason:      s.RefundReason,
		RefundReference:   s.RefundReference,
		CancelledBy:       s.CancelledBy,
		SpecialRequests:   s.SpecialRequests,
		TotalPrice:        s.TotalPrice,
		Currency:          s.Currency,
		ConfirmedAt:       s.ConfirmedAt,
		CancelledAt:       s.CancelledAt,
		CreatedAt:         s.CreatedAt,
		UpdatedAt:         s.UpdatedAt,
	}

	if s.PriceBreakdown != nil {
		d.PriceBreakdown = mapPriceBreakdownFromSchema(s.PriceBreakdown)
	}

	return d
}

// MapBookingFromDomain converts a domain entity into schema representation.
func MapBookingFromDomain(d *Booking) *schema.Booking {
	if d == nil {
		return nil
	}

	s := &schema.Booking{
		ID:                d.ID,
		BookingReference:  d.BookingReference,
		ListingID:         d.ListingID,
		CalendarEventID:   d.CalendarEventID,
		CleaningEventID:   d.CleaningEventID,
		GuestID:           d.GuestID,
		GuestName:         d.GuestName,
		GuestEmail:        d.GuestEmail,
		GuestPhone:        d.GuestPhone,
		GuestCount:        d.GuestCount,
		Status:            schema.BookingStatus(d.Status),
		BookingType:       schema.BookingType(d.BookingType),
		CheckIn:           d.CheckIn,
		CheckOut:          d.CheckOut,
		CheckInTime:       d.CheckInTime,
		CheckOutTime:      d.CheckOutTime,
		HoldExpiresAt:     d.HoldExpiresAt,
		PaymentDueAt:      d.PaymentDueAt,
		ActiveAt:          d.ActiveAt,
		CompletedAt:       d.CompletedAt,
		ArchivedAt:        d.ArchivedAt,
		PaymentReference:  d.PaymentReference,
		LastPaymentID:     d.LastPaymentID,
		RefundAmount:      d.RefundAmount,
		RefundInitiatedAt: d.RefundInitiatedAt,
		RefundProcessedAt: d.RefundProcessedAt,
		RefundReason:      d.RefundReason,
		RefundReference:   d.RefundReference,
		CancelledBy:       d.CancelledBy,
		SpecialRequests:   d.SpecialRequests,
		TotalPrice:        d.TotalPrice,
		Currency:          d.Currency,
		ConfirmedAt:       d.ConfirmedAt,
		CancelledAt:       d.CancelledAt,
		CreatedAt:         d.CreatedAt,
		UpdatedAt:         d.UpdatedAt,
	}

	if d.PriceBreakdown != nil {
		s.PriceBreakdown = mapPriceBreakdownToSchema(d.PriceBreakdown)
	}

	return s
}

func mapPriceBreakdownFromSchema(s *schema.PriceBreakdownSnapshot) *PriceBreakdownSnapshot {
	if s == nil {
		return nil
	}

	d := &PriceBreakdownSnapshot{
		BaseTotal:     s.BaseTotal,
		CleaningFee:   s.CleaningFee,
		ServiceFee:    s.ServiceFee,
		CautionFee:    s.CautionFee,
		ExtraGuestFee: s.ExtraGuestFee,
		Subtotal:      s.Subtotal,
		Total:         s.Total,
		Currency:      s.Currency,
	}

	if len(s.Discounts) > 0 {
		d.Discounts = make([]DiscountSnapshot, len(s.Discounts))
		for i, discount := range s.Discounts {
			d.Discounts[i] = DiscountSnapshot{
				Name:   discount.Name,
				Amount: discount.Amount,
				Type:   discount.Type,
			}
		}
	}

	if len(s.NightlyRates) > 0 {
		d.NightlyRates = make([]DailyRate, len(s.NightlyRates))
		for i, rate := range s.NightlyRates {
			d.NightlyRates[i] = DailyRate{
				Date:      rate.Date,
				BaseRate:  rate.BaseRate,
				FinalRate: rate.FinalRate,
			}
		}
	}

	if s.PlatformFees != nil {
		d.PlatformFees = &PlatformFeeBreakdown{
			GuestFeePercent:         s.PlatformFees.GuestFeePercent,
			GuestFeeAmount:          s.PlatformFees.GuestFeeAmount,
			HostCommissionPercent:   s.PlatformFees.HostCommissionPercent,
			HostCommissionAmount:    s.PlatformFees.HostCommissionAmount,
			PayoutProcessingPercent: s.PlatformFees.PayoutProcessingPercent,
			PayoutProcessingAmount:  s.PlatformFees.PayoutProcessingAmount,
			MinimumGuestFeeApplied:  s.PlatformFees.MinimumGuestFeeApplied,
			HostNetAmount:           s.PlatformFees.HostNetAmount,
		}
	}

	return d
}

func mapPriceBreakdownToSchema(d *PriceBreakdownSnapshot) *schema.PriceBreakdownSnapshot {
	if d == nil {
		return nil
	}

	s := &schema.PriceBreakdownSnapshot{
		BaseTotal:     d.BaseTotal,
		CleaningFee:   d.CleaningFee,
		ServiceFee:    d.ServiceFee,
		CautionFee:    d.CautionFee,
		ExtraGuestFee: d.ExtraGuestFee,
		Subtotal:      d.Subtotal,
		Total:         d.Total,
		Currency:      d.Currency,
	}

	if len(d.Discounts) > 0 {
		s.Discounts = make([]schema.DiscountSnapshot, len(d.Discounts))
		for i, discount := range d.Discounts {
			s.Discounts[i] = schema.DiscountSnapshot{
				Name:   discount.Name,
				Amount: discount.Amount,
				Type:   discount.Type,
			}
		}
	}

	if len(d.NightlyRates) > 0 {
		s.NightlyRates = make([]schema.DailyRate, len(d.NightlyRates))
		for i, rate := range d.NightlyRates {
			s.NightlyRates[i] = schema.DailyRate{
				Date:      rate.Date,
				BaseRate:  rate.BaseRate,
				FinalRate: rate.FinalRate,
			}
		}
	}

	if d.PlatformFees != nil {
		s.PlatformFees = &schema.PlatformFeeBreakdown{
			GuestFeePercent:         d.PlatformFees.GuestFeePercent,
			GuestFeeAmount:          d.PlatformFees.GuestFeeAmount,
			HostCommissionPercent:   d.PlatformFees.HostCommissionPercent,
			HostCommissionAmount:    d.PlatformFees.HostCommissionAmount,
			PayoutProcessingPercent: d.PlatformFees.PayoutProcessingPercent,
			PayoutProcessingAmount:  d.PlatformFees.PayoutProcessingAmount,
			MinimumGuestFeeApplied:  d.PlatformFees.MinimumGuestFeeApplied,
			HostNetAmount:           d.PlatformFees.HostNetAmount,
		}
	}

	return s
}
