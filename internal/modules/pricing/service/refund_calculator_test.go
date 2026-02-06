package service

import (
	"context"
	"testing"
	"time"

	"hauslet/config"
	"hauslet/internal/modules/pricing/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestCalculateRefund(t *testing.T) {
	// Setup simplified config for testing
	cfg := config.PlatformYAMLConfig{
		Fees: config.PlatformFeesConfig{
			HostCommissionPercent: 10.0,
		},
		Refunds: config.PlatformRefundConfig{
			GracePeriodHours:     48,
			ProcessingFeePercent: 3.0,
			ProcessingFeePayer:   "guest",
			PolicyTiers: map[string]config.RefundPolicyTier{
				"moderate": {
					Name:                      "moderate",
					CutoffHoursBeforeCheckIn:  120, // 5 days
					RefundPercentBeforeCutoff: 100.0,
					RefundPercentAfterCutoff:  50.0,
				},
				"strict": {
					Name:                      "strict",
					CutoffHoursBeforeCheckIn:  336, // 14 days
					RefundPercentBeforeCutoff: 50.0,
					RefundPercentAfterCutoff:  0.0,
				},
			},
		},
	}

	svc := &PricingServiceImpl{
		platformConfig: cfg,
	}
	ctx := context.Background()

	now := time.Now()
	bookingID := uuid.New()
	tenDays := 240 * time.Hour
	// Checkin 10 days from now
	checkInTime := now.Add(tenDays)

	tests := []struct {
		name     string
		input    domain.RefundCalculationInput
		expected func(*testing.T, *domain.RefundBreakdown, error)
	}{
		{
			name: "Grace Period Cancellation (Full Refund)",
			input: domain.RefundCalculationInput{
				BookingID:        bookingID,
				TotalPaid:        1000.0,
				Currency:         "NGN",
				ServiceFee:       50.0,
				CancelledBy:      domain.CancelledByGuest,
				RefundPolicy:     "moderate",
				CheckInTime:      checkInTime,
				BookingCreatedAt: now.Add(-1 * time.Hour), // Booked 1 hour ago (within 48h grace)
				CancellationTime: now,
			},
			expected: func(t *testing.T, r *domain.RefundBreakdown, err error) {
				assert.NoError(t, err)
				assert.Equal(t, 1000.0, r.NetRefund)
				assert.Equal(t, 100.0, r.RefundPercentage)
				assert.True(t, r.IsGracePeriod)
				assert.True(t, r.ServiceFeeRefundable)
				// Full refund, so no retention
				assert.Equal(t, 0.0, r.NonRefundedAmount)
				assert.Equal(t, 0.0, r.PlatformRetained)
			},
		},
		{
			name: "Moderate Policy - Before Cutoff (100% Refund minus fees)",
			input: domain.RefundCalculationInput{
				BookingID:        bookingID,
				TotalPaid:        1000.0, // Base 950 + Service 50
				Currency:         "NGN",
				ServiceFee:       50.0,
				CancelledBy:      domain.CancelledByGuest,
				RefundPolicy:     "moderate",
				CheckInTime:      checkInTime,              // 10 days before
				BookingCreatedAt: now.Add(-72 * time.Hour), // Booked 3 days ago (outside grace)
				CancellationTime: now,
			},
			expected: func(t *testing.T, r *domain.RefundBreakdown, err error) {
				assert.NoError(t, err)
				// Base Refund: 100% of 950 = 950
				// Service Fee: 50 (non-refundable)
				// Processing Fee: 3% of 1000 = 30 (Guest pays)
				// Net Refund: 950 - 30 = 920.
				assert.Equal(t, 950.0, r.BaseRefund)
				assert.Equal(t, 920.0, r.NetRefund)
				assert.Equal(t, 30.0, r.ProcessingFee)
				assert.Equal(t, 50.0, r.ServiceFee) // retained
				// Non-Refunded Breakdown:
				// Platform Retained = ServiceFee (50) + ProcessingFee (30) = 80
				// Host Retained = 0

				assert.InDelta(t, 80.0, r.NonRefundedAmount, 0.01)
				assert.InDelta(t, 80.0, r.PlatformRetained, 0.01)
				assert.InDelta(t, 0.0, r.HostRetainedAmount, 0.01)
			},
		},
		{
			name: "Moderate Policy - After Cutoff (50% Refund)",
			input: domain.RefundCalculationInput{
				BookingID:        bookingID,
				TotalPaid:        1000.0,
				Currency:         "NGN",
				ServiceFee:       50.0,
				CancelledBy:      domain.CancelledByGuest,
				RefundPolicy:     "moderate",
				CheckInTime:      now.Add(24 * time.Hour), // 1 day before check-in (cutoff is 5 days)
				BookingCreatedAt: now.Add(-100 * time.Hour),
				CancellationTime: now,
			},
			expected: func(t *testing.T, r *domain.RefundBreakdown, err error) {
				assert.NoError(t, err)
				// Base Amount: 950
				// Refund%: 50% -> 475 base refund
				// Processing Fee: 3% of 1000 = 30
				// Net Refund: 475 - 30 = 445
				assert.Equal(t, 475.0, r.BaseRefund)
				assert.InDelta(t, 445.0, r.NetRefund, 0.01)

				// Non-Refunded: 1000 - 445 = 555
				// Breakdown:
				// Service Fee (retained): 50
				// Processing Fee (retained by platform/recovered): 30
				// Host Gross = 555 - 50 - 30 = 475
				// Commission = 10% of 475 = 47.5
				// Host Retained = 475 - 47.5 = 427.5
				// Platform Retained = 50 + 30 + 47.5 = 127.5

				assert.InDelta(t, 127.5, r.PlatformRetained, 0.01)
				assert.InDelta(t, 427.5, r.HostRetainedAmount, 0.01)
			},
		},
		{
			name: "Host Cancellation (Full Refund, Host Penalized)",
			input: domain.RefundCalculationInput{
				BookingID:        bookingID,
				TotalPaid:        1000.0,
				Currency:         "NGN",
				ServiceFee:       50.0,
				CancelledBy:      domain.CancelledByHost,
				RefundPolicy:     "moderate",
				CheckInTime:      checkInTime,
				BookingCreatedAt: now.Add(-100 * time.Hour),
				CancellationTime: now,
			},
			expected: func(t *testing.T, r *domain.RefundBreakdown, err error) {
				assert.NoError(t, err)
				assert.Equal(t, 1000.0, r.NetRefund)
				assert.True(t, r.ServiceFeeRefundable)
				assert.Equal(t, 0.0, r.NonRefundedAmount)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := svc.CalculateRefund(ctx, tt.input)
			tt.expected(t, result, err)
		})
	}
}
