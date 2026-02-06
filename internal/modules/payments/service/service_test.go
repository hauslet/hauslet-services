package service

import (
	"context"
	"hauslet/internal/modules/payments/domain"
	"hauslet/internal/modules/payments/mocks"
	"hauslet/internal/modules/payments/repository/schema"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestService creates a service instance for testing
func createTestService(repo *mocks.MockRepository) *PaymentServiceImpl {

	return &PaymentServiceImpl{
		repo:            repo,
		paymentClient:   nil, // Not needed for these tests
		notificationSvc: nil, // Not needed for these tests
		log:             nil,
	}
}

// TestPaymentService_GetPayment tests getting a payment by ID
func TestPaymentService_GetPayment(t *testing.T) {
	mockRepo := new(mocks.MockRepository)
	svc := createTestService(mockRepo)
	ctx := context.Background()

	paymentID := uuid.New()
	expectedPayment := &schema.Payment{
		ID:         paymentID,
		Reference:  "PAY-TEST-123",
		Amount:     10000,
		Currency:   "NGN",
		Status:     schema.PaymentStatusSucceeded,
		PayerID:    uuid.New(),
		PayerEmail: "test@example.com",
		PayerName:  "Test User",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	mockRepo.On("GetPaymentByID", ctx, paymentID).Return(expectedPayment, nil)

	result, err := svc.GetPayment(ctx, paymentID)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, paymentID, result.ID)
	assert.Equal(t, "PAY-TEST-123", result.Reference)
	assert.Equal(t, int64(10000), result.Amount)
	mockRepo.AssertExpectations(t)
}

// TestPaymentService_GetPayment_NotFound tests payment not found scenario
func TestPaymentService_GetPayment_NotFound(t *testing.T) {
	mockRepo := new(mocks.MockRepository)
	svc := createTestService(mockRepo)
	ctx := context.Background()

	paymentID := uuid.New()
	mockRepo.On("GetPaymentByID", ctx, paymentID).Return(nil, nil)

	result, err := svc.GetPayment(ctx, paymentID)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, domain.ErrPaymentNotFound)
	mockRepo.AssertExpectations(t)
}

// TestPaymentService_GetPaymentByReference tests getting payment by reference
func TestPaymentService_GetPaymentByReference(t *testing.T) {
	mockRepo := new(mocks.MockRepository)
	svc := createTestService(mockRepo)
	ctx := context.Background()

	reference := "PAY-TEST-123"
	expectedPayment := &schema.Payment{
		ID:         uuid.New(),
		Reference:  reference,
		Amount:     10000,
		Currency:   "NGN",
		Status:     schema.PaymentStatusSucceeded,
		PayerID:    uuid.New(),
		PayerEmail: "test@example.com",
		PayerName:  "Test User",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	mockRepo.On("GetPaymentByReference", ctx, reference).Return(expectedPayment, nil)

	result, err := svc.GetPaymentByReference(ctx, reference)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, reference, result.Reference)
	mockRepo.AssertExpectations(t)
}

// TestPaymentService_ListPaymentsByPayer tests listing payments for a payer
func TestPaymentService_ListPaymentsByPayer(t *testing.T) {
	mockRepo := new(mocks.MockRepository)
	svc := createTestService(mockRepo)
	ctx := context.Background()

	payerID := uuid.New()
	expectedPayments := []*schema.Payment{
		{
			ID:         uuid.New(),
			Reference:  "PAY-1",
			Amount:     10000,
			Currency:   "NGN",
			Status:     schema.PaymentStatusSucceeded,
			PayerID:    payerID,
			PayerEmail: "test@example.com",
			PayerName:  "Test User",
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		},
		{
			ID:         uuid.New(),
			Reference:  "PAY-2",
			Amount:     20000,
			Currency:   "NGN",
			Status:     schema.PaymentStatusPending,
			PayerID:    payerID,
			PayerEmail: "test@example.com",
			PayerName:  "Test User",
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		},
	}

	mockRepo.On("ListPaymentsByPayer", ctx, payerID, (*schema.ResourceType)(nil), 10, 0).Return(expectedPayments, nil)

	result, err := svc.ListPaymentsByPayer(ctx, payerID, nil, 10, 0)

	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, int64(10000), result[0].Amount)
	assert.Equal(t, int64(20000), result[1].Amount)
	mockRepo.AssertExpectations(t)
}

// TestPaymentService_ListPaymentsByBooking tests listing payments for a booking
func TestPaymentService_ListPaymentsByBooking(t *testing.T) {
	mockRepo := new(mocks.MockRepository)
	svc := createTestService(mockRepo)
	ctx := context.Background()

	bookingID := uuid.New()
	expectedPayments := []*schema.Payment{
		{
			ID:         uuid.New(),
			Reference:  "PAY-1",
			Amount:     10000,
			Currency:   "NGN",
			Status:     schema.PaymentStatusSucceeded,
			BookingID:  &bookingID,
			PayerID:    uuid.New(),
			PayerEmail: "test@example.com",
			PayerName:  "Test User",
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		},
	}

	mockRepo.On("ListPaymentsByBookingID", ctx, bookingID).Return(expectedPayments, nil)

	result, err := svc.ListPaymentsByBooking(ctx, bookingID)

	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.NotNil(t, result[0].BookingID)
	mockRepo.AssertExpectations(t)
}

// TestPaymentService_GetTransaction tests getting a transaction by ID
func TestPaymentService_GetTransaction(t *testing.T) {
	mockRepo := new(mocks.MockRepository)
	svc := createTestService(mockRepo)
	ctx := context.Background()

	txID := uuid.New()
	expectedTx := &schema.Transaction{
		ID:        txID,
		Type:      schema.TransactionTypePayment,
		Reference: "TXN-TEST-123",
		Amount:    10000,
		Currency:  "NGN",
		Status:    schema.TransactionStatusSucceeded,
		Provider:  "paystack",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	mockRepo.On("GetTransactionByID", ctx, txID).Return(expectedTx, nil)

	result, err := svc.GetTransaction(ctx, txID)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, txID, result.ID)
	assert.Equal(t, int64(10000), result.Amount)
	mockRepo.AssertExpectations(t)
}

// TestPaymentService_GetTransaction_NotFound tests transaction not found
func TestPaymentService_GetTransaction_NotFound(t *testing.T) {
	mockRepo := new(mocks.MockRepository)
	svc := createTestService(mockRepo)
	ctx := context.Background()

	txID := uuid.New()
	mockRepo.On("GetTransactionByID", ctx, txID).Return(nil, nil)

	result, err := svc.GetTransaction(ctx, txID)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, domain.ErrTransactionNotFound)
	mockRepo.AssertExpectations(t)
}

// TestPaymentService_ListTransactionsByPayment tests listing transactions for a payment
func TestPaymentService_ListTransactionsByPayment(t *testing.T) {
	mockRepo := new(mocks.MockRepository)
	svc := createTestService(mockRepo)
	ctx := context.Background()

	paymentID := uuid.New()
	expectedTxs := []*schema.Transaction{
		{
			ID:        uuid.New(),
			PaymentID: &paymentID,
			Type:      schema.TransactionTypePayment,
			Reference: "TXN-1",
			Amount:    10000,
			Currency:  "NGN",
			Status:    schema.TransactionStatusSucceeded,
			Provider:  "paystack",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	mockRepo.On("ListTransactionsByPaymentID", ctx, paymentID).Return(expectedTxs, nil)

	result, err := svc.ListTransactionsByPayment(ctx, paymentID)

	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.NotNil(t, result[0].PaymentID)
	mockRepo.AssertExpectations(t)
}

// TestPaymentService_ListTransactionsByBooking tests listing transactions for a booking
func TestPaymentService_ListTransactionsByBooking(t *testing.T) {
	mockRepo := new(mocks.MockRepository)
	svc := createTestService(mockRepo)
	ctx := context.Background()

	bookingID := uuid.New()
	expectedTxs := []*schema.Transaction{
		{
			ID:        uuid.New(),
			BookingID: &bookingID,
			Type:      schema.TransactionTypePayout,
			Reference: "TXN-1",
			Amount:    50000,
			Currency:  "NGN",
			Status:    schema.TransactionStatusSucceeded,
			Provider:  "paystack",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	mockRepo.On("ListTransactionsByBookingID", ctx, bookingID).Return(expectedTxs, nil)

	result, err := svc.ListTransactionsByBooking(ctx, bookingID)

	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.NotNil(t, result[0].BookingID)
	mockRepo.AssertExpectations(t)
}

// TestPaymentService_GetPaymentMethod tests getting a payment method
func TestPaymentService_GetPaymentMethod(t *testing.T) {
	mockRepo := new(mocks.MockRepository)
	svc := createTestService(mockRepo)
	ctx := context.Background()

	userID := uuid.New()
	methodID := uuid.New()
	expectedMethod := &schema.PaymentMethod{
		ID:                methodID,
		UserID:            userID,
		Type:              schema.PaymentMethodCard,
		Provider:          "paystack",
		AuthorizationCode: "AUTH_123",
		Currency:          "NGN",
		IsActive:          true,
		IsDefault:         true,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	mockRepo.On("GetPaymentMethodByID", ctx, methodID).Return(expectedMethod, nil)

	result, err := svc.GetPaymentMethod(ctx, methodID, userID)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, methodID, result.ID)
	assert.Equal(t, userID, result.UserID)
	mockRepo.AssertExpectations(t)
}

// TestPaymentService_GetPaymentMethod_Unauthorized tests unauthorized access
func TestPaymentService_GetPaymentMethod_Unauthorized(t *testing.T) {
	mockRepo := new(mocks.MockRepository)
	svc := createTestService(mockRepo)
	ctx := context.Background()

	userID := uuid.New()
	differentUserID := uuid.New()
	methodID := uuid.New()

	expectedMethod := &schema.PaymentMethod{
		ID:                methodID,
		UserID:            differentUserID, // Different user!
		AuthorizationCode: "AUTH_123",
		IsActive:          true,
	}

	mockRepo.On("GetPaymentMethodByID", ctx, methodID).Return(expectedMethod, nil)

	result, err := svc.GetPaymentMethod(ctx, methodID, userID)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, domain.ErrUnauthorized)
	mockRepo.AssertExpectations(t)
}

// TestPaymentService_ListPaymentMethods tests listing payment methods
func TestPaymentService_ListPaymentMethods(t *testing.T) {
	mockRepo := new(mocks.MockRepository)
	svc := createTestService(mockRepo)
	ctx := context.Background()

	userID := uuid.New()
	expectedMethods := []*schema.PaymentMethod{
		{
			ID:                uuid.New(),
			UserID:            userID,
			AuthorizationCode: "AUTH_1",
			IsDefault:         true,
			IsActive:          true,
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		},
		{
			ID:                uuid.New(),
			UserID:            userID,
			AuthorizationCode: "AUTH_2",
			IsDefault:         false,
			IsActive:          true,
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		},
	}

	mockRepo.On("ListPaymentMethodsByUserID", ctx, userID).Return(expectedMethods, nil)

	result, err := svc.ListPaymentMethods(ctx, userID)

	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.True(t, result[0].IsDefault)
	assert.False(t, result[1].IsDefault)
	mockRepo.AssertExpectations(t)
}

// TestPaymentService_GetPayoutDetail tests getting a payout detail
func TestPayoutDetail_GetPayoutDetail(t *testing.T) {
	mockRepo := new(mocks.MockRepository)
	svc := createTestService(mockRepo)
	ctx := context.Background()

	detailID := uuid.New()
	userID := uuid.New()
	expectedDetail := &schema.PayoutDetail{
		ID:            detailID,
		UserID:        &userID,
		BankCode:      "058",
		BankName:      "GTBank",
		AccountNumber: "0123456789",
		AccountName:   "Test User",
		Currency:      "NGN",
		Market:        schema.MarketNigeria,
		Provider:      "paystack",
		RecipientCode: "RCP_123",
		IsVerified:    true,
		IsActive:      true,
		IsDefault:     true,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	mockRepo.On("GetPayoutDetailByID", ctx, detailID).Return(expectedDetail, nil)

	result, err := svc.GetPayoutDetail(ctx, detailID)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, detailID, result.ID)
	assert.Equal(t, "058", result.BankCode)
	mockRepo.AssertExpectations(t)
}

// TestPaymentService_GetPayoutDetail_NotFound tests payout detail not found
func TestPaymentService_GetPayoutDetail_NotFound(t *testing.T) {
	mockRepo := new(mocks.MockRepository)
	svc := createTestService(mockRepo)
	ctx := context.Background()

	detailID := uuid.New()
	mockRepo.On("GetPayoutDetailByID", ctx, detailID).Return(nil, nil)

	result, err := svc.GetPayoutDetail(ctx, detailID)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, domain.ErrPayoutDetailNotFound)
	mockRepo.AssertExpectations(t)
}
