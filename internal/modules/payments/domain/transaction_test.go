package domain

import (
	"hauslet/internal/platform/payment"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestTransaction_IsSucceeded(t *testing.T) {
	tests := []struct {
		name     string
		status   TransactionStatus
		expected bool
	}{
		{"pending", TransactionStatusPending, false},
		{"succeeded", TransactionStatusSucceeded, true},
		{"failed", TransactionStatusFailed, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx := &Transaction{Status: tt.status}
			assert.Equal(t, tt.expected, tx.IsSucceeded())
		})
	}
}

func TestTransaction_IsFailed(t *testing.T) {
	tests := []struct {
		name     string
		status   TransactionStatus
		expected bool
	}{
		{"pending", TransactionStatusPending, false},
		{"succeeded", TransactionStatusSucceeded, false},
		{"failed", TransactionStatusFailed, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx := &Transaction{Status: tt.status}
			assert.Equal(t, tt.expected, tx.IsFailed())
		})
	}
}

func TestTransaction_IsPending(t *testing.T) {
	tests := []struct {
		name     string
		status   TransactionStatus
		expected bool
	}{
		{"pending", TransactionStatusPending, true},
		{"succeeded", TransactionStatusSucceeded, false},
		{"failed", TransactionStatusFailed, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx := &Transaction{Status: tt.status}
			assert.Equal(t, tt.expected, tx.IsPending())
		})
	}
}

func TestTransaction_SetError(t *testing.T) {
	tx := &Transaction{
		ID:        uuid.New(),
		Status:    TransactionStatusPending,
		Amount:    10000,
		Currency:  payment.NGN,
		CreatedAt: time.Now(),
	}

	errorCode := "insufficient_funds"
	errorMessage := "Insufficient funds in account"

	tx.SetError(errorCode, errorMessage)

	assert.Equal(t, TransactionStatusFailed, tx.Status)
	assert.NotNil(t, tx.ErrorCode)
	assert.NotNil(t, tx.ErrorMessage)
	assert.Equal(t, errorCode, *tx.ErrorCode)
	assert.Equal(t, errorMessage, *tx.ErrorMessage)
}

func TestTransaction_MarkSucceeded(t *testing.T) {
	tx := &Transaction{
		ID:        uuid.New(),
		Status:    TransactionStatusPending,
		Amount:    10000,
		Currency:  payment.NGN,
		CreatedAt: time.Now(),
	}

	assert.True(t, tx.IsPending())
	assert.Nil(t, tx.ProcessedAt)

	beforeMark := time.Now()
	tx.MarkSucceeded()
	afterMark := time.Now()

	assert.True(t, tx.IsSucceeded())
	assert.NotNil(t, tx.ProcessedAt)
	assert.True(t, tx.ProcessedAt.After(beforeMark) || tx.ProcessedAt.Equal(beforeMark))
	assert.True(t, tx.ProcessedAt.Before(afterMark) || tx.ProcessedAt.Equal(afterMark))
}

func TestTransaction_PaymentScenario(t *testing.T) {
	paymentID := uuid.New()
	bookingID := uuid.New()
	businessID := uuid.New()
	providerTxID := "pstk_12345678"

	tx := &Transaction{
		ID:           uuid.New(),
		PaymentID:    &paymentID,
		BookingID:    &bookingID,
		BusinessID:   &businessID,
		Type:         TransactionTypePayment,
		Reference:    "TXN-PAY-12345678",
		Amount:       50000,
		Currency:     payment.NGN,
		Status:       TransactionStatusPending,
		Provider:     "paystack",
		ProviderTxID: &providerTxID,
		Description:  "Payment for booking",
		Metadata: map[string]string{
			"booking_id": bookingID.String(),
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Initial state
	assert.True(t, tx.IsPending())
	assert.False(t, tx.IsSucceeded())
	assert.False(t, tx.IsFailed())
	assert.Nil(t, tx.ProcessedAt)

	// Mark as succeeded
	tx.MarkSucceeded()

	assert.False(t, tx.IsPending())
	assert.True(t, tx.IsSucceeded())
	assert.False(t, tx.IsFailed())
	assert.NotNil(t, tx.ProcessedAt)
	assert.Equal(t, TransactionTypePayment, tx.Type)
}

func TestTransaction_RefundScenario(t *testing.T) {
	paymentID := uuid.New()
	bookingID := uuid.New()
	providerTxID := "pstk_refund_12345678"

	tx := &Transaction{
		ID:           uuid.New(),
		PaymentID:    &paymentID,
		BookingID:    &bookingID,
		Type:         TransactionTypeRefund,
		Reference:    "TXN-REF-12345678",
		Amount:       25000,
		Currency:     payment.NGN,
		Status:       TransactionStatusPending,
		Provider:     "paystack",
		ProviderTxID: &providerTxID,
		Description:  "Refund for cancelled booking",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Mark as succeeded
	tx.MarkSucceeded()

	assert.True(t, tx.IsSucceeded())
	assert.Equal(t, TransactionTypeRefund, tx.Type)
	assert.NotNil(t, tx.ProcessedAt)
}

func TestTransaction_PayoutScenario(t *testing.T) {
	businessID := uuid.New()
	bookingID := uuid.New()

	tx := &Transaction{
		ID:          uuid.New(),
		BookingID:   &bookingID,
		BusinessID:  &businessID,
		Type:        TransactionTypePayout,
		Reference:   "TXN-PYT-12345678",
		Amount:      100000,
		Currency:    payment.NGN,
		Status:      TransactionStatusPending,
		Provider:    "paystack",
		Description: "Payout to host",
		Metadata: map[string]string{
			"payout_type": "host_earnings",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Mark as succeeded
	tx.MarkSucceeded()

	assert.True(t, tx.IsSucceeded())
	assert.Equal(t, TransactionTypePayout, tx.Type)
	assert.NotNil(t, tx.ProcessedAt)
}

func TestTransaction_FailedScenario(t *testing.T) {
	tx := &Transaction{
		ID:        uuid.New(),
		Type:      TransactionTypePayment,
		Reference: "TXN-FAIL-12345678",
		Amount:    10000,
		Currency:  payment.NGN,
		Status:    TransactionStatusPending,
		Provider:  "paystack",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Simulate failure
	tx.SetError("card_declined", "Card was declined by issuer")

	assert.True(t, tx.IsFailed())
	assert.False(t, tx.IsSucceeded())
	assert.NotNil(t, tx.ErrorCode)
	assert.NotNil(t, tx.ErrorMessage)
	assert.Equal(t, "card_declined", *tx.ErrorCode)
	assert.Equal(t, "Card was declined by issuer", *tx.ErrorMessage)
}

func TestTransaction_StatusTransitions(t *testing.T) {
	tx := &Transaction{
		ID:        uuid.New(),
		Type:      TransactionTypePayment,
		Amount:    10000,
		Currency:  payment.USD,
		Status:    TransactionStatusPending,
		CreatedAt: time.Now(),
	}

	// Pending -> Succeeded
	assert.True(t, tx.IsPending())
	tx.MarkSucceeded()
	assert.True(t, tx.IsSucceeded())
	assert.False(t, tx.IsPending())

	// Create new transaction for pending -> failed test
	tx2 := &Transaction{
		ID:        uuid.New(),
		Type:      TransactionTypePayment,
		Amount:    10000,
		Currency:  payment.USD,
		Status:    TransactionStatusPending,
		CreatedAt: time.Now(),
	}

	assert.True(t, tx2.IsPending())
	tx2.SetError("timeout", "Request timeout")
	assert.True(t, tx2.IsFailed())
	assert.False(t, tx2.IsPending())
}

func TestTransaction_MultipleCurrencies(t *testing.T) {
	currencies := []payment.Currency{
		payment.NGN,
		payment.USD,
		payment.GHS,
		"KES",
		"ZAR",
	}

	for _, curr := range currencies {
		t.Run(string(curr), func(t *testing.T) {
			tx := &Transaction{
				ID:        uuid.New(),
				Type:      TransactionTypePayment,
				Amount:    10000,
				Currency:  curr,
				Status:    TransactionStatusPending,
				CreatedAt: time.Now(),
			}

			assert.Equal(t, curr, tx.Currency)
			assert.True(t, tx.IsPending())
		})
	}
}
