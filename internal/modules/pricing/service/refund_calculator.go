package service

import (
	"context"
	"fmt"
	"hauslet/internal/modules/pricing/domain"
	"time"
)

// CalculateRefund calculates the refund amount based on cancellation policy
func (s *PricingServiceImpl) CalculateRefund(
	ctx context.Context,
	input domain.RefundCalculationInput,
) (*domain.RefundBreakdown, error) {
	if s.log != nil {
		s.log.Info(" calculating refund booking=%s policy=%s cancelled_by=%s",
			input.BookingID, input.RefundPolicy, input.CancelledBy)
	}

	// Validate input
	if err := s.validateRefundInput(input); err != nil {
		return nil, err
	}

	now := time.Now()
	calculatedAt := now

	// Calculate time differences
	hoursAfterBooking := input.CancellationTime.Sub(input.BookingCreatedAt).Hours()
	hoursUntilCheckIn := input.CheckInTime.Sub(input.CancellationTime).Hours()

	// Initialize breakdown
	breakdown := &domain.RefundBreakdown{
		BookingID:         input.BookingID,
		OriginalAmount:    input.TotalPaid,
		Currency:          input.Currency,
		HoursUntilCheckIn: hoursUntilCheckIn,
		HoursAfterBooking: hoursAfterBooking,
		CancelledBy:       string(input.CancelledBy),
		BookedAt:          input.BookingCreatedAt,
		CheckInAt:         input.CheckInTime,
		CancelledAt:       input.CancellationTime,
		CalculatedAt:      calculatedAt,
	}

	// Check for invalid scenarios
	if hoursUntilCheckIn < 0 {
		// Already checked in or past check-in time
		return s.buildNoRefundBreakdown(breakdown, "no_refund_post_checkin",
			"No refund available - cancellation after check-in time")
	}

	// STEP 1: Check if within grace period (24h from booking creation)
	gracePeriodHours := float64(s.platformConfig.Refunds.GracePeriodHours)
	if hoursAfterBooking <= gracePeriodHours {
		return s.calculateGracePeriodRefund(breakdown)
	}

	// STEP 2: Host-initiated cancellation (always full refund to guest)
	if input.CancelledBy == domain.CancelledByHost {
		return s.calculateHostCancellationRefund(breakdown)
	}

	// STEP 3: Admin-initiated cancellation (full refund, no fees)
	if input.CancelledBy == domain.CancelledByAdmin {
		return s.calculateAdminCancellationRefund(breakdown)
	}

	// STEP 4: Guest-initiated cancellation - apply policy-based calculation
	return s.calculatePolicyBasedRefund(breakdown, input.RefundPolicy)
}

// validateRefundInput validates the refund calculation input
func (s *PricingServiceImpl) validateRefundInput(input domain.RefundCalculationInput) error {
	if input.BookingID == [16]byte{} {
		return domain.ErrInvalidDates
	}
	if input.TotalPaid <= 0 {
		return domain.ErrNoPaymentFound
	}
	if input.Currency == "" {
		return fmt.Errorf("currency is required")
	}
	if input.BookingCreatedAt.IsZero() || input.CheckInTime.IsZero() || input.CancellationTime.IsZero() {
		return domain.ErrInvalidDates
	}
	if input.CancellationTime.Before(input.BookingCreatedAt) {
		return domain.ErrInvalidDates
	}
	if input.RefundPolicy == "" {
		return domain.ErrInvalidPolicy
	}
	return nil
}

// calculateGracePeriodRefund handles grace period (24h) full refunds
func (s *PricingServiceImpl) calculateGracePeriodRefund(breakdown *domain.RefundBreakdown) (*domain.RefundBreakdown, error) {
	// Full refund during grace period
	breakdown.RefundPercentage = 100.0
	breakdown.BaseRefund = breakdown.OriginalAmount
	breakdown.AppliedPolicy = "grace_period"
	breakdown.IsGracePeriod = true

	// Calculate processing fee
	processingFee := s.calculateProcessingFee(breakdown.OriginalAmount)
	breakdown.ProcessingFee = processingFee
	breakdown.ProcessingFeePayer = s.platformConfig.Refunds.ProcessingFeePayer

	// Apply processing fee based on who pays
	feeDeduction := s.applyProcessingFee(processingFee, breakdown.ProcessingFeePayer)
	breakdown.NetRefund = breakdown.BaseRefund - feeDeduction

	// Build explanation
	gracePeriodHours := s.platformConfig.Refunds.GracePeriodHours
	breakdown.Summary = fmt.Sprintf("100%% refund - %dh grace period", gracePeriodHours)
	breakdown.Reason = fmt.Sprintf(
		"Full refund applied. Booking cancelled within %d-hour grace period (%.1f hours after booking).",
		gracePeriodHours,
		breakdown.HoursAfterBooking,
	)
	breakdown.PolicyRules = fmt.Sprintf(
		"Grace Period Policy: 100%% refund within %d hours of booking (processing fee: %.1f%%)",
		gracePeriodHours,
		s.platformConfig.Refunds.ProcessingFeePercent,
	)

	if s.log != nil {
		s.log.Info(" refund calculation: grace_period refund=%.2f (%.1f%%)",
			breakdown.NetRefund, breakdown.RefundPercentage)
	}

	return breakdown, nil
}

// calculateHostCancellationRefund handles host-initiated cancellations
func (s *PricingServiceImpl) calculateHostCancellationRefund(breakdown *domain.RefundBreakdown) (*domain.RefundBreakdown, error) {
	// Host cancellation: Guest gets 100% back, host absorbs all fees
	breakdown.RefundPercentage = 100.0
	breakdown.BaseRefund = breakdown.OriginalAmount
	breakdown.AppliedPolicy = "host_cancellation"
	breakdown.IsGracePeriod = false

	// Host pays all fees
	breakdown.ProcessingFee = 0
	breakdown.ProcessingFeePayer = "host"
	breakdown.NetRefund = breakdown.OriginalAmount // Guest gets everything back

	breakdown.Summary = "100% refund - host cancelled"
	breakdown.Reason = fmt.Sprintf(
		"Full refund applied. Host cancelled the booking %.1f hours before check-in. Guest receives full amount with no fees deducted.",
		breakdown.HoursUntilCheckIn,
	)
	breakdown.PolicyRules = "Host Cancellation Policy: 100% refund to guest, host absorbs all fees and may face penalties"

	if s.log != nil {
		s.log.Info(" refund calculation: host_cancellation refund=%.2f (%.1f%%)",
			breakdown.NetRefund, breakdown.RefundPercentage)
	}

	return breakdown, nil
}

// calculateAdminCancellationRefund handles admin-initiated cancellations
func (s *PricingServiceImpl) calculateAdminCancellationRefund(breakdown *domain.RefundBreakdown) (*domain.RefundBreakdown, error) {
	// Admin cancellation: Full refund, no fees
	breakdown.RefundPercentage = 100.0
	breakdown.BaseRefund = breakdown.OriginalAmount
	breakdown.AppliedPolicy = "admin_override"
	breakdown.IsGracePeriod = false

	breakdown.ProcessingFee = 0
	breakdown.ProcessingFeePayer = "platform"
	breakdown.NetRefund = breakdown.OriginalAmount

	breakdown.Summary = "100% refund - admin override"
	breakdown.Reason = "Full refund applied by platform administrator. All fees waived."
	breakdown.PolicyRules = "Admin Override: 100% refund, platform absorbs all fees"

	if s.log != nil {
		s.log.Info(" refund calculation: admin_cancellation refund=%.2f (%.1f%%)",
			breakdown.NetRefund, breakdown.RefundPercentage)
	}

	return breakdown, nil
}

// calculatePolicyBasedRefund applies policy-specific refund rules
func (s *PricingServiceImpl) calculatePolicyBasedRefund(
	breakdown *domain.RefundBreakdown,
	policyName string,
) (*domain.RefundBreakdown, error) {
	// Get policy tier from config
	policyTier, exists := s.platformConfig.Refunds.PolicyTiers[policyName]
	if !exists {
		if s.log != nil {
			s.log.Warn("policy not found: %s, using moderate", policyName)
		}
		// Fallback to moderate policy
		policyTier = s.platformConfig.Refunds.PolicyTiers["moderate"]
		if policyTier.Name == "" {
			return nil, domain.ErrPolicyNotFound
		}
		policyName = "moderate"
	}

	breakdown.AppliedPolicy = policyName
	breakdown.IsGracePeriod = false
	breakdown.PolicyRules = fmt.Sprintf(
		"%s Policy: %s (cutoff: %dh before check-in, before: %.0f%%, after: %.0f%%)",
		policyTier.Name,
		policyTier.Description,
		policyTier.CutoffHoursBeforeCheckIn,
		policyTier.RefundPercentBeforeCutoff,
		policyTier.RefundPercentAfterCutoff,
	)

	// Determine refund percentage based on cutoff
	var refundPercent float64
	cutoffHours := float64(policyTier.CutoffHoursBeforeCheckIn)

	if breakdown.HoursUntilCheckIn >= cutoffHours {
		// Before cutoff - higher refund
		refundPercent = policyTier.RefundPercentBeforeCutoff
		breakdown.Reason = fmt.Sprintf(
			"Cancelled %.1f hours before check-in, which is before the %d-hour cutoff. %s policy applies %.0f%% refund.",
			breakdown.HoursUntilCheckIn,
			policyTier.CutoffHoursBeforeCheckIn,
			policyName,
			refundPercent,
		)
	} else {
		// After cutoff - lower/no refund
		refundPercent = policyTier.RefundPercentAfterCutoff
		breakdown.Reason = fmt.Sprintf(
			"Cancelled %.1f hours before check-in, which is after the %d-hour cutoff. %s policy applies %.0f%% refund.",
			breakdown.HoursUntilCheckIn,
			policyTier.CutoffHoursBeforeCheckIn,
			policyName,
			refundPercent,
		)
	}

	breakdown.RefundPercentage = refundPercent
	breakdown.BaseRefund = breakdown.OriginalAmount * (refundPercent / 100.0)

	// Handle zero refund case
	if refundPercent <= 0 {
		breakdown.ProcessingFee = 0
		breakdown.ProcessingFeePayer = "n/a"
		breakdown.NetRefund = 0
		breakdown.Summary = fmt.Sprintf("No refund - %s policy", policyName)

		if s.log != nil {
			s.log.Info(" refund calculation: %s policy no_refund", policyName)
		}
		return breakdown, nil
	}

	// Calculate processing fee
	processingFee := s.calculateProcessingFee(breakdown.OriginalAmount)
	breakdown.ProcessingFee = processingFee
	breakdown.ProcessingFeePayer = s.platformConfig.Refunds.ProcessingFeePayer

	// Apply processing fee
	feeDeduction := s.applyProcessingFee(processingFee, breakdown.ProcessingFeePayer)
	breakdown.NetRefund = breakdown.BaseRefund - feeDeduction

	// Ensure refund doesn't go negative
	if breakdown.NetRefund < 0 {
		breakdown.NetRefund = 0
	}

	// Build summary
	if refundPercent >= 100.0 {
		breakdown.Summary = fmt.Sprintf("100%% refund - %s policy", policyName)
	} else {
		breakdown.Summary = fmt.Sprintf("%.0f%% refund - %s policy", refundPercent, policyName)
	}

	if s.log != nil {
		s.log.Info(" refund calculation: %s policy refund=%.2f (%.1f%%) hours_until_checkin=%.1f cutoff=%d",
			policyName, breakdown.NetRefund, breakdown.RefundPercentage,
			breakdown.HoursUntilCheckIn, policyTier.CutoffHoursBeforeCheckIn)
	}

	return breakdown, nil
}

// buildNoRefundBreakdown creates a breakdown for no-refund scenarios
func (s *PricingServiceImpl) buildNoRefundBreakdown(
	breakdown *domain.RefundBreakdown,
	policy, reason string,
) (*domain.RefundBreakdown, error) {
	breakdown.RefundPercentage = 0
	breakdown.BaseRefund = 0
	breakdown.ProcessingFee = 0
	breakdown.ProcessingFeePayer = "n/a"
	breakdown.NetRefund = 0
	breakdown.AppliedPolicy = policy
	breakdown.IsGracePeriod = false
	breakdown.Summary = "No refund"
	breakdown.Reason = reason
	breakdown.PolicyRules = "No refund policy applies"

	if s.log != nil {
		s.log.Info(" refund calculation: %s no_refund", policy)
	}

	return breakdown, nil
}

// calculateProcessingFee calculates the non-refundable processing fee
func (s *PricingServiceImpl) calculateProcessingFee(amount float64) float64 {
	return amount * (s.platformConfig.Refunds.ProcessingFeePercent / 100.0)
}

// applyProcessingFee determines how much fee to deduct based on payer
func (s *PricingServiceImpl) applyProcessingFee(fee float64, payer string) float64 {
	switch payer {
	case "guest":
		return fee // Guest pays full fee
	case "host":
		return 0 // Host pays, guest gets full refund
	case "shared":
		return fee / 2 // Split 50/50
	case "platform", "n/a":
		return 0
	default:
		return fee // Default: guest pays
	}
}
