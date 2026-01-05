package mocks

import (
	"context"
	"hauslet/internal/platform/payment"

	"github.com/stretchr/testify/mock"
)

// MockPaymentClient is a mock implementation of payment client interfaces
type MockPaymentClient struct {
	mock.Mock
}

// Initialize mocks payment initialization
func (m *MockPaymentClient) Initialize(ctx context.Context, req payment.PaymentRequest) (*payment.PaymentResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*payment.PaymentResponse), args.Error(1)
}

// AuthorizePayment mocks payment authorization
func (m *MockPaymentClient) AuthorizePayment(ctx context.Context, req payment.AuthorizationRequest) (*payment.AuthorizationResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*payment.AuthorizationResponse), args.Error(1)
}

// ChargeAuthorization mocks charging a saved payment method
func (m *MockPaymentClient) ChargeAuthorization(ctx context.Context, req payment.PaymentRequest) (*payment.PaymentResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*payment.PaymentResponse), args.Error(1)
}

// Verify mocks payment verification
func (m *MockPaymentClient) Verify(ctx context.Context, currency payment.Currency, reference string) (*payment.PaymentResponse, error) {
	args := m.Called(ctx, currency, reference)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*payment.PaymentResponse), args.Error(1)
}

// Refund mocks payment refund
func (m *MockPaymentClient) Refund(ctx context.Context, currency payment.Currency, originalTxID string, amount int64, reason string) (*payment.RefundResponse, error) {
	args := m.Called(ctx, currency, originalTxID, amount, reason)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*payment.RefundResponse), args.Error(1)
}

// ValidateAccount mocks bank account validation
func (m *MockPaymentClient) ValidateAccount(ctx context.Context, currency payment.Currency, bankCode, accountNumber string) (string, error) {
	args := m.Called(ctx, currency, bankCode, accountNumber)
	return args.String(0), args.Error(1)
}

// CreateRecipient mocks recipient creation
func (m *MockPaymentClient) CreateRecipient(ctx context.Context, currency payment.Currency, bankCode, accountNumber, accountName string) (string, error) {
	args := m.Called(ctx, currency, bankCode, accountNumber, accountName)
	return args.String(0), args.Error(1)
}

// Transfer mocks money transfer/payout
func (m *MockPaymentClient) Transfer(ctx context.Context, req payment.PayoutRequest) (*payment.PayoutResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*payment.PayoutResponse), args.Error(1)
}

// VerifyTransfer mocks transfer verification
func (m *MockPaymentClient) VerifyTransfer(ctx context.Context, currency payment.Currency, reference string) (*payment.PayoutResponse, error) {
	args := m.Called(ctx, currency, reference)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*payment.PayoutResponse), args.Error(1)
}

// MockNotificationService is a mock for notification service
type MockNotificationService struct {
	mock.Mock
}

// SendPaymentReceipt mocks sending payment receipt
func (m *MockNotificationService) SendPaymentReceipt(payment interface{}) {
	m.Called(payment)
}

// SendPaymentFailed mocks sending payment failure notification
func (m *MockNotificationService) SendPaymentFailed(payment interface{}) {
	m.Called(payment)
}

// SendRefundNotification mocks sending refund notification
func (m *MockNotificationService) SendRefundNotification(payment interface{}) {
	m.Called(payment)
}

// SendPayoutNotification mocks sending payout notification
func (m *MockNotificationService) SendPayoutNotification(transaction interface{}) {
	m.Called(transaction)
}
