package domain

import (
	"hauslet/internal/platform/payment"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreatePaymentInput_Validate(t *testing.T) {
	validUserID := uuid.New()
	validBookingID := uuid.New()

	tests := []struct {
		name        string
		input       CreatePaymentInput
		expectError bool
		expectedErr error
	}{
		{
			name: "valid input",
			input: CreatePaymentInput{
				Amount:       10000,
				Currency:     payment.NGN,
				Market:       MarketNigeria,
				PayerID:      validUserID,
				PayerEmail:   "user@example.com",
				PayerName:    "Test User",
				ResourceType: ResourceTypeBooking,
			},
			expectError: false,
		},
		{
			name: "invalid amount - zero",
			input: CreatePaymentInput{
				Amount:     0,
				Currency:   payment.NGN,
				Market:     MarketNigeria,
				PayerID:    validUserID,
				PayerEmail: "user@example.com",
				PayerName:  "Test User",
			},
			expectError: true,
			expectedErr: ErrInvalidPaymentAmount,
		},
		{
			name: "invalid amount - negative",
			input: CreatePaymentInput{
				Amount:     -5000,
				Currency:   payment.NGN,
				Market:     MarketNigeria,
				PayerID:    validUserID,
				PayerEmail: "user@example.com",
				PayerName:  "Test User",
			},
			expectError: true,
			expectedErr: ErrInvalidPaymentAmount,
		},
		{
			name: "invalid currency for market",
			input: CreatePaymentInput{
				Amount:     10000,
				Currency:   payment.USD,
				Market:     MarketGhana,
				PayerID:    validUserID,
				PayerEmail: "user@example.com",
				PayerName:  "Test User",
			},
			expectError: true,
			expectedErr: ErrInvalidPaymentCurrency,
		},
		{
			name: "missing payer ID",
			input: CreatePaymentInput{
				Amount:     10000,
				Currency:   payment.NGN,
				Market:     MarketNigeria,
				PayerID:    uuid.Nil,
				PayerEmail: "user@example.com",
				PayerName:  "Test User",
			},
			expectError: true,
			expectedErr: ErrMissingRequiredField,
		},
		{
			name: "missing payer email",
			input: CreatePaymentInput{
				Amount:     10000,
				Currency:   payment.NGN,
				Market:     MarketNigeria,
				PayerID:    validUserID,
				PayerEmail: "",
				PayerName:  "Test User",
			},
			expectError: true,
			expectedErr: ErrMissingRequiredField,
		},
		{
			name: "invalid resource type",
			input: CreatePaymentInput{
				Amount:       10000,
				Currency:     payment.NGN,
				Market:       MarketNigeria,
				PayerID:      validUserID,
				PayerEmail:   "user@example.com",
				PayerName:    "Test User",
				ResourceType: "invalid_type",
			},
			expectError: true,
			expectedErr: ErrInvalidResourceType,
		},
		{
			name: "valid with booking ID",
			input: CreatePaymentInput{
				Amount:       10000,
				Currency:     payment.NGN,
				Market:       MarketNigeria,
				PayerID:      validUserID,
				PayerEmail:   "user@example.com",
				PayerName:    "Test User",
				BookingID:    &validBookingID,
				ResourceType: ResourceTypeBooking,
				ResourceID:   &validBookingID,
			},
			expectError: false,
		},
		{
			name: "default resource type to general when empty",
			input: CreatePaymentInput{
				Amount:       10000,
				Currency:     payment.NGN,
				Market:       MarketNigeria,
				PayerID:      validUserID,
				PayerEmail:   "user@example.com",
				PayerName:    "Test User",
				ResourceType: "",
			},
			expectError: false,
		},
		{
			name: "valid USD in Nigeria",
			input: CreatePaymentInput{
				Amount:     10000,
				Currency:   payment.USD,
				Market:     MarketNigeria,
				PayerID:    validUserID,
				PayerEmail: "user@example.com",
				PayerName:  "Test User",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.input.Validate()
			if tt.expectError {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.expectedErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestRefundPaymentInput_Validate(t *testing.T) {
	validPaymentID := uuid.New()
	partialAmount := int64(5000)

	tests := []struct {
		name        string
		input       RefundPaymentInput
		expectError bool
		expectedErr error
	}{
		{
			name: "valid full refund",
			input: RefundPaymentInput{
				PaymentID:  validPaymentID,
				Amount:     nil,
				Reason:     "Customer request",
				RefundedBy: uuid.New(),
			},
			expectError: false,
		},
		{
			name: "valid partial refund",
			input: RefundPaymentInput{
				PaymentID:  validPaymentID,
				Amount:     &partialAmount,
				Reason:     "Partial cancellation",
				RefundedBy: uuid.New(),
			},
			expectError: false,
		},
		{
			name: "missing payment ID",
			input: RefundPaymentInput{
				PaymentID:  uuid.Nil,
				Reason:     "Some reason",
				RefundedBy: uuid.New(),
			},
			expectError: true,
			expectedErr: ErrMissingRequiredField,
		},
		{
			name: "invalid partial refund amount - zero",
			input: RefundPaymentInput{
				PaymentID:  validPaymentID,
				Amount:     func() *int64 { a := int64(0); return &a }(),
				Reason:     "Some reason",
				RefundedBy: uuid.New(),
			},
			expectError: true,
			expectedErr: ErrInvalidPaymentAmount,
		},
		{
			name: "invalid partial refund amount - negative",
			input: RefundPaymentInput{
				PaymentID:  validPaymentID,
				Amount:     func() *int64 { a := int64(-1000); return &a }(),
				Reason:     "Some reason",
				RefundedBy: uuid.New(),
			},
			expectError: true,
			expectedErr: ErrInvalidPaymentAmount,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.input.Validate()
			if tt.expectError {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.expectedErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestCreatePaymentMethodInput_Validate(t *testing.T) {
	validUserID := uuid.New()

	tests := []struct {
		name        string
		input       CreatePaymentMethodInput
		expectError bool
		expectedErr error
	}{
		{
			name: "valid input",
			input: CreatePaymentMethodInput{
				UserID:            validUserID,
				AuthorizationCode: "AUTH_xyz123",
				Currency:          payment.NGN,
				Provider:          "paystack",
				SetAsDefault:      false,
			},
			expectError: false,
		},
		{
			name: "valid with card metadata",
			input: CreatePaymentMethodInput{
				UserID:            validUserID,
				AuthorizationCode: "AUTH_xyz123",
				Currency:          payment.NGN,
				Provider:          "paystack",
				Last4Digits:       func() *string { s := "1234"; return &s }(),
				Brand:             func() *string { s := "Visa"; return &s }(),
				SetAsDefault:      true,
			},
			expectError: false,
		},
		{
			name: "missing user ID",
			input: CreatePaymentMethodInput{
				UserID:            uuid.Nil,
				AuthorizationCode: "AUTH_xyz123",
				Currency:          payment.NGN,
				Provider:          "paystack",
			},
			expectError: true,
			expectedErr: ErrMissingRequiredField,
		},
		{
			name: "missing authorization code",
			input: CreatePaymentMethodInput{
				UserID:            validUserID,
				AuthorizationCode: "",
				Currency:          payment.NGN,
				Provider:          "paystack",
			},
			expectError: true,
			expectedErr: ErrMissingRequiredField,
		},
		{
			name: "invalid currency",
			input: CreatePaymentMethodInput{
				UserID:            validUserID,
				AuthorizationCode: "AUTH_xyz123",
				Currency:          "",
				Provider:          "paystack",
			},
			expectError: true,
			expectedErr: ErrInvalidPaymentCurrency,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.input.Validate()
			if tt.expectError {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.expectedErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestCreatePayoutDetailInput_Validate(t *testing.T) {
	validUserID := uuid.New()
	validBusinessID := uuid.New()

	tests := []struct {
		name        string
		input       CreatePayoutDetailInput
		expectError bool
		expectedErr error
	}{
		{
			name: "valid user payout detail",
			input: CreatePayoutDetailInput{
				UserID:        &validUserID,
				BankCode:      "058",
				AccountNumber: "0123456789",
				AccountName:   "John Doe",
				Currency:      payment.NGN,
				Market:        MarketNigeria,
				SetAsDefault:  false,
			},
			expectError: false,
		},
		{
			name: "valid business payout detail",
			input: CreatePayoutDetailInput{
				BusinessID:    &validBusinessID,
				BankCode:      "044",
				AccountNumber: "9876543210",
				AccountName:   "Business Ltd",
				Currency:      payment.NGN,
				Market:        MarketNigeria,
				SetAsDefault:  true,
			},
			expectError: false,
		},
		{
			name: "missing owner (no user or business ID)",
			input: CreatePayoutDetailInput{
				BankCode:      "058",
				AccountNumber: "0123456789",
				AccountName:   "John Doe",
				Currency:      payment.NGN,
				Market:        MarketNigeria,
			},
			expectError: true,
			expectedErr: ErrMissingRequiredField,
		},
		{
			name: "missing bank code",
			input: CreatePayoutDetailInput{
				UserID:        &validUserID,
				BankCode:      "",
				AccountNumber: "0123456789",
				AccountName:   "John Doe",
				Currency:      payment.NGN,
				Market:        MarketNigeria,
			},
			expectError: true,
			expectedErr: ErrMissingRequiredField,
		},
		{
			name: "missing account number",
			input: CreatePayoutDetailInput{
				UserID:       &validUserID,
				BankCode:     "058",
				AccountName:  "John Doe",
				Currency:     payment.NGN,
				Market:       MarketNigeria,
			},
			expectError: true,
			expectedErr: ErrMissingRequiredField,
		},
		{
			name: "missing account name",
			input: CreatePayoutDetailInput{
				UserID:        &validUserID,
				BankCode:      "058",
				AccountNumber: "0123456789",
				Currency:      payment.NGN,
				Market:        MarketNigeria,
			},
			expectError: true,
			expectedErr: ErrMissingRequiredField,
		},
		{
			name: "invalid currency",
			input: CreatePayoutDetailInput{
				UserID:        &validUserID,
				BankCode:      "058",
				AccountNumber: "0123456789",
				AccountName:   "John Doe",
				Currency:      "",
				Market:        MarketNigeria,
			},
			expectError: true,
			expectedErr: ErrInvalidPaymentCurrency,
		},
		{
			name: "unsupported currency for market",
			input: CreatePayoutDetailInput{
				UserID:        &validUserID,
				BankCode:      "058",
				AccountNumber: "0123456789",
				AccountName:   "John Doe",
				Currency:      payment.USD,
				Market:        MarketGhana,
			},
			expectError: true,
			expectedErr: ErrInvalidPaymentCurrency,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.input.Validate()
			if tt.expectError {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.expectedErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestProcessPayoutInput_Validate(t *testing.T) {
	validPayoutDetailID := uuid.New()
	validBookingID := uuid.New()

	tests := []struct {
		name        string
		input       ProcessPayoutInput
		expectError bool
		expectedErr error
	}{
		{
			name: "valid payout",
			input: ProcessPayoutInput{
				Amount:         50000,
				Currency:       payment.NGN,
				PayoutDetailID: validPayoutDetailID,
				BookingID:      &validBookingID,
				Description:    "Host payout",
			},
			expectError: false,
		},
		{
			name: "invalid amount - zero",
			input: ProcessPayoutInput{
				Amount:         0,
				Currency:       payment.NGN,
				PayoutDetailID: validPayoutDetailID,
			},
			expectError: true,
			expectedErr: ErrInvalidPaymentAmount,
		},
		{
			name: "invalid amount - negative",
			input: ProcessPayoutInput{
				Amount:         -5000,
				Currency:       payment.NGN,
				PayoutDetailID: validPayoutDetailID,
			},
			expectError: true,
			expectedErr: ErrInvalidPaymentAmount,
		},
		{
			name: "invalid currency",
			input: ProcessPayoutInput{
				Amount:         50000,
				Currency:       "",
				PayoutDetailID: validPayoutDetailID,
			},
			expectError: true,
			expectedErr: ErrInvalidPaymentCurrency,
		},
		{
			name: "missing payout detail ID",
			input: ProcessPayoutInput{
				Amount:         50000,
				Currency:       payment.NGN,
				PayoutDetailID: uuid.Nil,
			},
			expectError: true,
			expectedErr: ErrMissingRequiredField,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.input.Validate()
			if tt.expectError {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.expectedErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestCreatePaymentInput_ResourceTypeAutoCorrection(t *testing.T) {
	validUserID := uuid.New()

	input := CreatePaymentInput{
		Amount:       10000,
		Currency:     payment.NGN,
		Market:       MarketNigeria,
		PayerID:      validUserID,
		PayerEmail:   "user@example.com",
		PayerName:    "Test User",
		ResourceType: "", // Empty resource type
	}

	err := input.Validate()
	require.NoError(t, err)
	assert.Equal(t, ResourceTypeGeneral, input.ResourceType)
}

func TestCreatePaymentInput_NilResourceID(t *testing.T) {
	validUserID := uuid.New()
	nilUUID := uuid.Nil

	input := CreatePaymentInput{
		Amount:       10000,
		Currency:     payment.NGN,
		Market:       MarketNigeria,
		PayerID:      validUserID,
		PayerEmail:   "user@example.com",
		PayerName:    "Test User",
		ResourceType: ResourceTypeGeneral,
		ResourceID:   &nilUUID, // Nil UUID should be set to nil pointer
	}

	err := input.Validate()
	require.NoError(t, err)
	assert.Nil(t, input.ResourceID)
}
