package domain

import (
	"hauslet/internal/platform/payment"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestPayoutDetail_CanReceivePayouts(t *testing.T) {
	tests := []struct {
		name          string
		isActive      bool
		isVerified    bool
		recipientCode string
		expected      bool
	}{
		{
			name:          "active, verified, with recipient code",
			isActive:      true,
			isVerified:    true,
			recipientCode: "RCP_12345",
			expected:      true,
		},
		{
			name:          "inactive",
			isActive:      false,
			isVerified:    true,
			recipientCode: "RCP_12345",
			expected:      false,
		},
		{
			name:          "not verified",
			isActive:      true,
			isVerified:    false,
			recipientCode: "RCP_12345",
			expected:      false,
		},
		{
			name:          "no recipient code",
			isActive:      true,
			isVerified:    true,
			recipientCode: "",
			expected:      false,
		},
		{
			name:          "all conditions failed",
			isActive:      false,
			isVerified:    false,
			recipientCode: "",
			expected:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pd := &PayoutDetail{
				IsActive:      tt.isActive,
				IsVerified:    tt.isVerified,
				RecipientCode: tt.recipientCode,
			}
			result := pd.CanReceivePayouts()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPayoutDetail_GetDisplayName(t *testing.T) {
	tests := []struct {
		name          string
		bankName      string
		accountNumber string
		expected      string
	}{
		{
			name:          "standard account",
			bankName:      "Test Bank",
			accountNumber: "0123456789",
			expected:      "Test Bank - ****6789",
		},
		{
			name:          "access bank",
			bankName:      "Access Bank",
			accountNumber: "9876543210",
			expected:      "Access Bank - ****3210",
		},
		{
			name:          "short account number",
			bankName:      "Some Bank",
			accountNumber: "123",
			expected:      "Some Bank - 123",
		},
		{
			name:          "exactly 4 digits",
			bankName:      "Mini Bank",
			accountNumber: "1234",
			expected:      "Mini Bank - 1234",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pd := &PayoutDetail{
				BankName:      tt.bankName,
				AccountNumber: tt.accountNumber,
			}
			result := pd.GetDisplayName()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMaskAccountNumber(t *testing.T) {
	tests := []struct {
		name          string
		accountNumber string
		expected      string
	}{
		{
			name:          "standard 10 digit account",
			accountNumber: "0123456789",
			expected:      "****6789",
		},
		{
			name:          "short account number",
			accountNumber: "123",
			expected:      "123",
		},
		{
			name:          "exactly 4 digits",
			accountNumber: "1234",
			expected:      "1234",
		},
		{
			name:          "long account number",
			accountNumber: "12345678901234",
			expected:      "****1234",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MaskAccountNumber(tt.accountNumber)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPayoutDetail_BelongsToUser(t *testing.T) {
	userID := uuid.New()
	otherUserID := uuid.New()

	tests := []struct {
		name     string
		userID   *uuid.UUID
		checkID  uuid.UUID
		expected bool
	}{
		{
			name:     "belongs to user",
			userID:   &userID,
			checkID:  userID,
			expected: true,
		},
		{
			name:     "belongs to different user",
			userID:   &userID,
			checkID:  otherUserID,
			expected: false,
		},
		{
			name:     "no user ID set",
			userID:   nil,
			checkID:  userID,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pd := &PayoutDetail{
				UserID: tt.userID,
			}
			result := pd.BelongsToUser(tt.checkID)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPayoutDetail_BelongsToBusiness(t *testing.T) {
	businessID := uuid.New()
	otherBusinessID := uuid.New()

	tests := []struct {
		name       string
		businessID *uuid.UUID
		checkID    uuid.UUID
		expected   bool
	}{
		{
			name:       "belongs to business",
			businessID: &businessID,
			checkID:    businessID,
			expected:   true,
		},
		{
			name:       "belongs to different business",
			businessID: &businessID,
			checkID:    otherBusinessID,
			expected:   false,
		},
		{
			name:       "no business ID set",
			businessID: nil,
			checkID:    businessID,
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pd := &PayoutDetail{
				BusinessID: tt.businessID,
			}
			result := pd.BelongsToBusiness(tt.checkID)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPayoutDetail_UserPayoutDetailLifecycle(t *testing.T) {
	userID := uuid.New()
	verifiedTime := time.Now()

	pd := &PayoutDetail{
		ID:            uuid.New(),
		UserID:        &userID,
		BusinessID:    nil,
		BankCode:      "058",
		BankName:      "GTBank",
		AccountNumber: "0123456789",
		AccountName:   "John Doe",
		Currency:      payment.NGN,
		Market:        MarketNigeria,
		Provider:      "paystack",
		RecipientCode: "RCP_xyz123",
		IsVerified:    true,
		VerifiedAt:    &verifiedTime,
		IsDefault:     true,
		IsActive:      true,
		Metadata:      map[string]string{"type": "personal"},
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// Test initial state
	assert.True(t, pd.BelongsToUser(userID))
	assert.False(t, pd.BelongsToBusiness(uuid.New()))
	assert.True(t, pd.IsVerified)
	assert.True(t, pd.IsActive)
	assert.True(t, pd.IsDefault)
	assert.True(t, pd.CanReceivePayouts())
	assert.Equal(t, "GTBank - ****6789", pd.GetDisplayName())
	assert.NotNil(t, pd.VerifiedAt)

	// Deactivate
	pd.IsActive = false
	assert.False(t, pd.CanReceivePayouts())

	// Reactivate
	pd.IsActive = true
	assert.True(t, pd.CanReceivePayouts())
}

func TestPayoutDetail_BusinessPayoutDetailLifecycle(t *testing.T) {
	businessID := uuid.New()
	verifiedTime := time.Now()

	pd := &PayoutDetail{
		ID:            uuid.New(),
		UserID:        nil,
		BusinessID:    &businessID,
		BankCode:      "044",
		BankName:      "Access Bank",
		AccountNumber: "9876543210",
		AccountName:   "Business Name Ltd",
		Currency:      payment.NGN,
		Market:        MarketNigeria,
		Provider:      "paystack",
		RecipientCode: "RCP_business123",
		IsVerified:    true,
		VerifiedAt:    &verifiedTime,
		IsDefault:     true,
		IsActive:      true,
		Metadata:      map[string]string{"type": "business"},
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// Test initial state
	assert.False(t, pd.BelongsToUser(uuid.New()))
	assert.True(t, pd.BelongsToBusiness(businessID))
	assert.True(t, pd.IsVerified)
	assert.True(t, pd.CanReceivePayouts())
	assert.Equal(t, "Access Bank - ****3210", pd.GetDisplayName())
}

func TestPayoutDetail_UnverifiedAccount(t *testing.T) {
	userID := uuid.New()

	pd := &PayoutDetail{
		ID:            uuid.New(),
		UserID:        &userID,
		BankCode:      "058",
		BankName:      "GTBank",
		AccountNumber: "0123456789",
		AccountName:   "Jane Doe",
		Currency:      payment.NGN,
		Market:        MarketNigeria,
		Provider:      "paystack",
		RecipientCode: "RCP_temp123",
		IsVerified:    false,
		VerifiedAt:    nil,
		IsDefault:     false,
		IsActive:      true,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// Unverified account cannot receive payouts
	assert.False(t, pd.CanReceivePayouts())
	assert.Nil(t, pd.VerifiedAt)

	// Verify account
	verifiedTime := time.Now()
	pd.IsVerified = true
	pd.VerifiedAt = &verifiedTime

	assert.True(t, pd.CanReceivePayouts())
	assert.NotNil(t, pd.VerifiedAt)
}

func TestPayoutDetail_MultipleCurrencies(t *testing.T) {
	userID := uuid.New()

	currencies := []struct {
		currency payment.Currency
		market   Market
	}{
		{payment.NGN, MarketNigeria},
		{payment.GHS, MarketGhana},
		{payment.USD, MarketOther},
		{"KES", MarketKenya},
		{"ZAR", MarketSouthAfrica},
	}

	for _, tc := range currencies {
		t.Run(string(tc.currency), func(t *testing.T) {
			pd := &PayoutDetail{
				ID:            uuid.New(),
				UserID:        &userID,
				BankCode:      "123",
				BankName:      "Test Bank",
				AccountNumber: "0123456789",
				AccountName:   "Test User",
				Currency:      tc.currency,
				Market:        tc.market,
				Provider:      "paystack",
				RecipientCode: "RCP_123",
				IsVerified:    true,
				IsActive:      true,
				CreatedAt:     time.Now(),
			}

			assert.Equal(t, tc.currency, pd.Currency)
			assert.Equal(t, tc.market, pd.Market)
			assert.True(t, pd.CanReceivePayouts())
		})
	}
}

func TestPayoutDetail_DefaultBehavior(t *testing.T) {
	userID := uuid.New()

	// Create two payout details
	pd1 := &PayoutDetail{
		ID:            uuid.New(),
		UserID:        &userID,
		BankCode:      "058",
		BankName:      "GTBank",
		AccountNumber: "1111111111",
		AccountName:   "User Name",
		RecipientCode: "RCP_1",
		IsVerified:    true,
		IsActive:      true,
		IsDefault:     true,
		CreatedAt:     time.Now(),
	}

	pd2 := &PayoutDetail{
		ID:            uuid.New(),
		UserID:        &userID,
		BankCode:      "044",
		BankName:      "Access Bank",
		AccountNumber: "2222222222",
		AccountName:   "User Name",
		RecipientCode: "RCP_2",
		IsVerified:    true,
		IsActive:      true,
		IsDefault:     false,
		CreatedAt:     time.Now(),
	}

	assert.True(t, pd1.IsDefault)
	assert.False(t, pd2.IsDefault)

	// Change default
	pd1.IsDefault = false
	pd2.IsDefault = true

	assert.False(t, pd1.IsDefault)
	assert.True(t, pd2.IsDefault)
}

func TestPayoutDetail_SoftDelete(t *testing.T) {
	userID := uuid.New()

	pd := &PayoutDetail{
		ID:            uuid.New(),
		UserID:        &userID,
		BankCode:      "058",
		BankName:      "GTBank",
		AccountNumber: "1234567890",
		AccountName:   "User Name",
		RecipientCode: "RCP_123",
		IsVerified:    true,
		IsActive:      true,
		IsDefault:     true,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	assert.Nil(t, pd.DeletedAt)
	assert.True(t, pd.IsActive)

	// Soft delete
	now := time.Now()
	pd.DeletedAt = &now
	pd.IsActive = false

	assert.NotNil(t, pd.DeletedAt)
	assert.False(t, pd.IsActive)
	assert.False(t, pd.CanReceivePayouts())
}
