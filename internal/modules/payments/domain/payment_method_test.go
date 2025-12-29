package domain

import (
	"hauslet/internal/platform/payment"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestPaymentMethod_IsExpired(t *testing.T) {
	now := time.Now()
	currentYear := now.Year()
	currentMonth := int(now.Month())

	tests := []struct {
		name        string
		expiryMonth *int
		expiryYear  *int
		expected    bool
	}{
		{
			name:        "no expiry data - not expired",
			expiryMonth: nil,
			expiryYear:  nil,
			expected:    false,
		},
		{
			name: "expired last year",
			expiryMonth: func() *int {
				m := currentMonth
				return &m
			}(),
			expiryYear: func() *int {
				y := currentYear - 1
				return &y
			}(),
			expected: true,
		},
		{
			name: "expires next year - not expired",
			expiryMonth: func() *int {
				m := currentMonth
				return &m
			}(),
			expiryYear: func() *int {
				y := currentYear + 1
				return &y
			}(),
			expected: false,
		},
		{
			name: "expired earlier this year",
			expiryMonth: func() *int {
				m := 1
				return &m
			}(),
			expiryYear: func() *int {
				y := currentYear
				return &y
			}(),
			expected: currentMonth > 1,
		},
		{
			name: "expires later this year - not expired",
			expiryMonth: func() *int {
				m := 12
				return &m
			}(),
			expiryYear: func() *int {
				y := currentYear
				return &y
			}(),
			expected: currentMonth > 12,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pm := &PaymentMethod{
				ExpiryMonth: tt.expiryMonth,
				ExpiryYear:  tt.expiryYear,
			}
			result := pm.IsExpired()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPaymentMethod_CanCharge(t *testing.T) {
	futureYear := time.Now().Year() + 2
	futureMonth := 12
	pastYear := time.Now().Year() - 1
	pastMonth := 1

	tests := []struct {
		name              string
		isActive          bool
		authorizationCode string
		expiryMonth       *int
		expiryYear        *int
		expected          bool
	}{
		{
			name:              "active card with valid expiry",
			isActive:          true,
			authorizationCode: "AUTH_12345",
			expiryMonth:       &futureMonth,
			expiryYear:        &futureYear,
			expected:          true,
		},
		{
			name:              "inactive card",
			isActive:          false,
			authorizationCode: "AUTH_12345",
			expiryMonth:       &futureMonth,
			expiryYear:        &futureYear,
			expected:          false,
		},
		{
			name:              "expired card",
			isActive:          true,
			authorizationCode: "AUTH_12345",
			expiryMonth:       &pastMonth,
			expiryYear:        &pastYear,
			expected:          false,
		},
		{
			name:        "no authorization code",
			isActive:    true,
			expiryMonth: &futureMonth,
			expiryYear:  &futureYear,
			expected:    false,
		},
		{
			name:              "active with no expiry data",
			isActive:          true,
			authorizationCode: "AUTH_12345",
			expiryMonth:       nil,
			expiryYear:        nil,
			expected:          true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pm := &PaymentMethod{
				IsActive:          tt.isActive,
				AuthorizationCode: tt.authorizationCode,
				ExpiryMonth:       tt.expiryMonth,
				ExpiryYear:        tt.expiryYear,
			}
			result := pm.CanCharge()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMaskCardNumber(t *testing.T) {
	tests := []struct {
		name     string
		last4    string
		expected string
	}{
		{
			name:     "standard card",
			last4:    "1234",
			expected: "**** **** **** 1234",
		},
		{
			name:     "another card",
			last4:    "5678",
			expected: "**** **** **** 5678",
		},
		{
			name:     "zeros",
			last4:    "0000",
			expected: "**** **** **** 0000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MaskCardNumber(tt.last4)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPaymentMethod_GetDisplayName(t *testing.T) {
	tests := []struct {
		name        string
		last4Digits *string
		brand       *string
		methodType  PaymentMethodType
		expected    string
	}{
		{
			name: "visa card with last 4 digits",
			last4Digits: func() *string {
				s := "1234"
				return &s
			}(),
			brand: func() *string {
				s := "Visa"
				return &s
			}(),
			methodType: PaymentMethodCard,
			expected:   "Visa ending in 1234",
		},
		{
			name: "mastercard with last 4 digits",
			last4Digits: func() *string {
				s := "5678"
				return &s
			}(),
			brand: func() *string {
				s := "Mastercard"
				return &s
			}(),
			methodType: PaymentMethodCard,
			expected:   "Mastercard ending in 5678",
		},
		{
			name: "card without brand",
			last4Digits: func() *string {
				s := "9999"
				return &s
			}(),
			brand:      nil,
			methodType: PaymentMethodCard,
			expected:   "Card ending in 9999",
		},
		{
			name:        "card without last 4 digits",
			last4Digits: nil,
			brand:       nil,
			methodType:  PaymentMethodCard,
			expected:    "card",
		},
		{
			name:        "bank account",
			last4Digits: nil,
			brand:       nil,
			methodType:  PaymentMethodBank,
			expected:    "bank",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pm := &PaymentMethod{
				Last4Digits: tt.last4Digits,
				Brand:       tt.brand,
				Type:        tt.methodType,
			}
			result := pm.GetDisplayName()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPaymentMethod_CompleteCardLifecycle(t *testing.T) {
	userID := uuid.New()
	last4 := "4321"
	brand := "Visa"
	cardType := "debit"
	bankName := "Test Bank"
	expiryMonth := 12
	expiryYear := time.Now().Year() + 3
	customerCode := "CUS_12345"

	pm := &PaymentMethod{
		ID:                uuid.New(),
		UserID:            userID,
		Type:              PaymentMethodCard,
		Provider:          "paystack",
		AuthorizationCode: "AUTH_xyz123",
		Currency:          payment.NGN,
		Last4Digits:       &last4,
		CardType:          &cardType,
		Brand:             &brand,
		ExpiryMonth:       &expiryMonth,
		ExpiryYear:        &expiryYear,
		BankName:          &bankName,
		CustomerCode:      &customerCode,
		IsDefault:         true,
		IsActive:          true,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	// Test initial state
	assert.True(t, pm.IsActive)
	assert.True(t, pm.IsDefault)
	assert.False(t, pm.IsExpired())
	assert.True(t, pm.CanCharge())
	assert.Equal(t, "Visa ending in 4321", pm.GetDisplayName())
	assert.Equal(t, "**** **** **** 4321", MaskCardNumber(*pm.Last4Digits))

	// Deactivate card
	pm.IsActive = false
	assert.False(t, pm.CanCharge())

	// Reactivate
	pm.IsActive = true
	assert.True(t, pm.CanCharge())

	// Simulate expiration
	pm.ExpiryYear = func() *int {
		y := time.Now().Year() - 1
		return &y
	}()
	assert.True(t, pm.IsExpired())
	assert.False(t, pm.CanCharge())
}

func TestPaymentMethod_MultipleCurrencies(t *testing.T) {
	userID := uuid.New()
	currencies := []payment.Currency{
		payment.NGN,
		payment.USD,
		payment.GHS,
		"KES",
		"ZAR",
	}

	for _, curr := range currencies {
		t.Run(string(curr), func(t *testing.T) {
			pm := &PaymentMethod{
				ID:                uuid.New(),
				UserID:            userID,
				Type:              PaymentMethodCard,
				Provider:          "paystack",
				AuthorizationCode: "AUTH_123",
				Currency:          curr,
				IsActive:          true,
				IsDefault:         false,
				CreatedAt:         time.Now(),
			}

			assert.Equal(t, curr, pm.Currency)
			assert.True(t, pm.CanCharge())
		})
	}
}

func TestPaymentMethod_DefaultBehavior(t *testing.T) {
	userID := uuid.New()

	// Create multiple payment methods
	pm1 := &PaymentMethod{
		ID:                uuid.New(),
		UserID:            userID,
		AuthorizationCode: "AUTH_1",
		IsActive:          true,
		IsDefault:         true,
		CreatedAt:         time.Now(),
	}

	pm2 := &PaymentMethod{
		ID:                uuid.New(),
		UserID:            userID,
		AuthorizationCode: "AUTH_2",
		IsActive:          true,
		IsDefault:         false,
		CreatedAt:         time.Now(),
	}

	assert.True(t, pm1.IsDefault)
	assert.False(t, pm2.IsDefault)

	// Set pm2 as default
	pm1.IsDefault = false
	pm2.IsDefault = true

	assert.False(t, pm1.IsDefault)
	assert.True(t, pm2.IsDefault)
}

func TestPaymentMethod_InactiveBehavior(t *testing.T) {
	pm := &PaymentMethod{
		ID:                uuid.New(),
		UserID:            uuid.New(),
		AuthorizationCode: "AUTH_123",
		IsActive:          true,
		IsDefault:         true,
		ExpiryMonth: func() *int {
			m := 12
			return &m
		}(),
		ExpiryYear: func() *int {
			y := time.Now().Year() + 2
			return &y
		}(),
		CreatedAt: time.Now(),
	}

	// Initially active and chargeable
	assert.True(t, pm.IsActive)
	assert.True(t, pm.CanCharge())

	// Deactivate
	pm.IsActive = false
	now := time.Now()
	pm.DeletedAt = &now

	assert.False(t, pm.IsActive)
	assert.False(t, pm.CanCharge())
	assert.NotNil(t, pm.DeletedAt)
}
