package http

import (
	"context"
	"hauslet/internal/platform/payment"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestIsPaymentEvent tests the payment event checker
func TestIsPaymentEvent(t *testing.T) {
	tests := []struct {
		eventType string
		expected  bool
	}{
		{"charge.success", true},
		{"charge.failed", true},
		{"refund.processed", true},
		{"refund.failed", true},
		{"transfer.success", false},
		{"transfer.failed", false},
		{"subscription.create", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.eventType, func(t *testing.T) {
			result := isPaymentEvent(tt.eventType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestParseExpiry tests expiry date parsing
func TestParseExpiry_Success(t *testing.T) {
	month, year, err := parseExpiry("12", "2025")

	assert.NoError(t, err)
	assert.NotNil(t, month)
	assert.NotNil(t, year)
	assert.Equal(t, 12, *month)
	assert.Equal(t, 2025, *year)
}

// TestParseExpiry_InvalidMonth tests invalid month scenarios
func TestParseExpiry_InvalidMonth(t *testing.T) {
	tests := []struct {
		name  string
		month string
		year  string
	}{
		{"empty month", "", "2025"},
		{"invalid month format", "abc", "2025"},
		{"month too low", "0", "2025"},
		{"month too high", "13", "2025"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := parseExpiry(tt.month, tt.year)
			assert.Error(t, err)
		})
	}
}

// TestParseExpiry_InvalidYear tests invalid year scenarios
func TestParseExpiry_InvalidYear(t *testing.T) {
	tests := []struct {
		name  string
		month string
		year  string
	}{
		{"empty year", "12", ""},
		{"invalid year format", "12", "abcd"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := parseExpiry(tt.month, tt.year)
			assert.Error(t, err)
		})
	}
}

// TestProcessEvent_NilEvent tests processing nil event
func TestProcessEvent_NilEvent(t *testing.T) {
	handler := &WebhookHandler{}

	err := handler.ProcessEvent(context.TODO(), nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "event is nil")
}

// TestProcessEvent_UnhandledEvent tests unhandled event type
func TestProcessEvent_UnhandledEvent(t *testing.T) {
	handler := &WebhookHandler{}

	event := &payment.UnifiedEvent{
		Type:      "unknown.event",
		Reference: "PAY-TEST-123",
	}

	err := handler.ProcessEvent(context.TODO(), event)

	// Should not error on unhandled events
	assert.NoError(t, err)
}
