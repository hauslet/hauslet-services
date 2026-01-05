package domain

import (
	"hauslet/internal/platform/payment"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPayment_CanRefund(t *testing.T) {
	tests := []struct {
		name     string
		payment  *Payment
		expected bool
	}{
		{
			name: "succeeded payment can be refunded",
			payment: &Payment{
				Status:         PaymentStatusSucceeded,
				Amount:         10000,
				RefundedAmount: 0,
			},
			expected: true,
		},
		{
			name: "pending payment cannot be refunded",
			payment: &Payment{
				Status:         PaymentStatusPending,
				Amount:         10000,
				RefundedAmount: 0,
			},
			expected: false,
		},
		{
			name: "failed payment cannot be refunded",
			payment: &Payment{
				Status:         PaymentStatusFailed,
				Amount:         10000,
				RefundedAmount: 0,
			},
			expected: false,
		},
		{
			name: "fully refunded payment cannot be refunded again",
			payment: &Payment{
				Status:         PaymentStatusSucceeded,
				Amount:         10000,
				RefundedAmount: 10000,
			},
			expected: false,
		},
		{
			name: "partially refunded payment can be refunded",
			payment: &Payment{
				Status:         PaymentStatusSucceeded,
				Amount:         10000,
				RefundedAmount: 5000,
			},
			expected: true,
		},
		{
			name: "over-refunded payment cannot be refunded (data integrity issue)",
			payment: &Payment{
				Status:         PaymentStatusSucceeded,
				Amount:         10000,
				RefundedAmount: 15000,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.payment.CanRefund()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPayment_RemainingRefundableAmount(t *testing.T) {
	tests := []struct {
		name     string
		payment  *Payment
		expected int64
	}{
		{
			name: "full amount refundable",
			payment: &Payment{
				Status:         PaymentStatusSucceeded,
				Amount:         10000,
				RefundedAmount: 0,
			},
			expected: 10000,
		},
		{
			name: "partial amount refundable",
			payment: &Payment{
				Status:         PaymentStatusSucceeded,
				Amount:         10000,
				RefundedAmount: 3000,
			},
			expected: 7000,
		},
		{
			name: "no amount refundable when fully refunded",
			payment: &Payment{
				Status:         PaymentStatusSucceeded,
				Amount:         10000,
				RefundedAmount: 10000,
			},
			expected: 0,
		},
		{
			name: "no amount refundable when payment failed",
			payment: &Payment{
				Status:         PaymentStatusFailed,
				Amount:         10000,
				RefundedAmount: 0,
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.payment.RemainingRefundableAmount()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPayment_IsFullyRefunded(t *testing.T) {
	tests := []struct {
		name     string
		payment  *Payment
		expected bool
	}{
		{
			name: "not refunded",
			payment: &Payment{
				Amount:         10000,
				RefundedAmount: 0,
			},
			expected: false,
		},
		{
			name: "partially refunded",
			payment: &Payment{
				Amount:         10000,
				RefundedAmount: 5000,
			},
			expected: false,
		},
		{
			name: "fully refunded",
			payment: &Payment{
				Amount:         10000,
				RefundedAmount: 10000,
			},
			expected: true,
		},
		{
			name: "over refunded (edge case)",
			payment: &Payment{
				Amount:         10000,
				RefundedAmount: 15000,
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.payment.IsFullyRefunded()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPayment_IsPending(t *testing.T) {
	tests := []struct {
		name     string
		status   PaymentStatus
		expected bool
	}{
		{"pending", PaymentStatusPending, true},
		{"succeeded", PaymentStatusSucceeded, false},
		{"failed", PaymentStatusFailed, false},
		{"cancelled", PaymentStatusCancelled, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Payment{Status: tt.status}
			assert.Equal(t, tt.expected, p.IsPending())
		})
	}
}

func TestPayment_IsSucceeded(t *testing.T) {
	tests := []struct {
		name     string
		status   PaymentStatus
		expected bool
	}{
		{"pending", PaymentStatusPending, false},
		{"succeeded", PaymentStatusSucceeded, true},
		{"failed", PaymentStatusFailed, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Payment{Status: tt.status}
			assert.Equal(t, tt.expected, p.IsSucceeded())
		})
	}
}

func TestPayment_IsFailed(t *testing.T) {
	tests := []struct {
		name     string
		status   PaymentStatus
		expected bool
	}{
		{"pending", PaymentStatusPending, false},
		{"succeeded", PaymentStatusSucceeded, false},
		{"failed", PaymentStatusFailed, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Payment{Status: tt.status}
			assert.Equal(t, tt.expected, p.IsFailed())
		})
	}
}

func TestGenerateReference(t *testing.T) {
	id := uuid.New()
	reference := GenerateReference("PMT", id)

	require.NotEmpty(t, reference)
	assert.Contains(t, reference, "PAY-PMT-")
	assert.Contains(t, reference, id.String()[:8])
}

func TestPayment_RefundScenarios(t *testing.T) {
	now := time.Now()

	t.Run("successful payment with full refund", func(t *testing.T) {
		p := &Payment{
			ID:             uuid.New(),
			Amount:         50000,
			Status:         PaymentStatusSucceeded,
			RefundedAmount: 0,
			CreatedAt:      now,
		}

		assert.True(t, p.CanRefund())
		assert.Equal(t, int64(50000), p.RemainingRefundableAmount())
		assert.False(t, p.IsFullyRefunded())

		// Simulate full refund
		p.RefundedAmount = 50000
		refundTime := now.Add(time.Hour)
		p.RefundedAt = &refundTime

		assert.False(t, p.CanRefund())
		assert.Equal(t, int64(0), p.RemainingRefundableAmount())
		assert.True(t, p.IsFullyRefunded())
	})

	t.Run("successful payment with partial refund", func(t *testing.T) {
		p := &Payment{
			ID:             uuid.New(),
			Amount:         100000,
			Status:         PaymentStatusSucceeded,
			RefundedAmount: 0,
		}

		// First partial refund
		p.RefundedAmount = 30000
		assert.True(t, p.CanRefund())
		assert.Equal(t, int64(70000), p.RemainingRefundableAmount())
		assert.False(t, p.IsFullyRefunded())

		// Second partial refund
		p.RefundedAmount = 80000
		assert.True(t, p.CanRefund())
		assert.Equal(t, int64(20000), p.RemainingRefundableAmount())
		assert.False(t, p.IsFullyRefunded())

		// Final refund
		p.RefundedAmount = 100000
		assert.False(t, p.CanRefund())
		assert.Equal(t, int64(0), p.RemainingRefundableAmount())
		assert.True(t, p.IsFullyRefunded())
	})
}

func TestPayment_StatusTransitions(t *testing.T) {
	p := &Payment{
		ID:        uuid.New(),
		Amount:    10000,
		Currency:  payment.NGN,
		Status:    PaymentStatusPending,
		CreatedAt: time.Now(),
	}

	// Initial state - pending
	assert.True(t, p.IsPending())
	assert.False(t, p.IsSucceeded())
	assert.False(t, p.IsFailed())
	assert.False(t, p.CanRefund())

	// Transition to succeeded
	p.Status = PaymentStatusSucceeded
	assert.False(t, p.IsPending())
	assert.True(t, p.IsSucceeded())
	assert.False(t, p.IsFailed())
	assert.True(t, p.CanRefund())

	// Transition to failed (edge case - normally wouldn't happen)
	p.Status = PaymentStatusFailed
	assert.False(t, p.IsPending())
	assert.False(t, p.IsSucceeded())
	assert.True(t, p.IsFailed())
	assert.False(t, p.CanRefund())
}

func TestPayment_CompletePaymentLifecycle(t *testing.T) {
	userID := uuid.New()
	bookingID := uuid.New()
	businessID := uuid.New()

	p := &Payment{
		ID:           uuid.New(),
		Reference:    "PAY-TEST-12345678",
		BookingID:    &bookingID,
		BusinessID:   &businessID,
		ResourceType: ResourceTypeBooking,
		ResourceID:   &bookingID,
		PayerID:      userID,
		PayerEmail:   "user@example.com",
		PayerName:    "Test User",
		Amount:       25000,
		Currency:     payment.NGN,
		Market:       MarketNigeria,
		Status:       PaymentStatusPending,
		Provider:     "paystack",
		Description:  "Booking payment",
		Metadata: map[string]string{
			"booking_id": bookingID.String(),
		},
		RequiresAction: true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	// Verify initial state
	assert.True(t, p.IsPending())
	assert.False(t, p.CanRefund())
	assert.NotNil(t, p.BookingID)
	assert.Equal(t, ResourceTypeBooking, p.ResourceType)

	// Simulate payment success
	p.Status = PaymentStatusSucceeded
	p.RequiresAction = false
	p.UpdatedAt = time.Now()

	assert.True(t, p.IsSucceeded())
	assert.True(t, p.CanRefund())
	assert.Equal(t, int64(25000), p.RemainingRefundableAmount())

	// Simulate partial refund
	p.RefundedAmount = 10000
	refundTime := time.Now()
	p.RefundedAt = &refundTime

	assert.True(t, p.CanRefund())
	assert.Equal(t, int64(15000), p.RemainingRefundableAmount())
	assert.False(t, p.IsFullyRefunded())
	assert.NotNil(t, p.RefundedAt)
}
