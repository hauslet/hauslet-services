package mocks

import (
	"context"
	"hauslet/internal/modules/payments/repository/schema"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

// MockRepository is a mock implementation of the Repository interface
type MockRepository struct {
	mock.Mock
}

// Payment repository methods

func (m *MockRepository) CreatePayment(ctx context.Context, payment *schema.Payment) error {
	args := m.Called(ctx, payment)
	return args.Error(0)
}

func (m *MockRepository) GetPaymentByID(ctx context.Context, id uuid.UUID) (*schema.Payment, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*schema.Payment), args.Error(1)
}

func (m *MockRepository) GetPaymentByReference(ctx context.Context, reference string) (*schema.Payment, error) {
	args := m.Called(ctx, reference)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*schema.Payment), args.Error(1)
}

func (m *MockRepository) UpdatePayment(ctx context.Context, payment *schema.Payment) error {
	args := m.Called(ctx, payment)
	return args.Error(0)
}

func (m *MockRepository) DeletePayment(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRepository) ListPaymentsByPayer(ctx context.Context, payerID uuid.UUID, resourceType *schema.ResourceType, limit, offset int) ([]*schema.Payment, error) {
	args := m.Called(ctx, payerID, resourceType, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*schema.Payment), args.Error(1)
}

func (m *MockRepository) ListPaymentsByBookingID(ctx context.Context, bookingID uuid.UUID) ([]*schema.Payment, error) {
	args := m.Called(ctx, bookingID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*schema.Payment), args.Error(1)
}

func (m *MockRepository) ListPaymentsByBusinessID(ctx context.Context, businessID uuid.UUID, limit, offset int) ([]*schema.Payment, error) {
	args := m.Called(ctx, businessID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*schema.Payment), args.Error(1)
}

func (m *MockRepository) ListPaymentsByStatus(ctx context.Context, status schema.PaymentStatus, limit, offset int) ([]*schema.Payment, error) {
	args := m.Called(ctx, status, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*schema.Payment), args.Error(1)
}

func (m *MockRepository) WithinTransaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	args := m.Called(ctx, fn)
	return args.Error(0)
}

func (m *MockRepository) GetPaymentStats(ctx context.Context, payerID uuid.UUID) (*schema.PaymentStats, error) {
	args := m.Called(ctx, payerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*schema.PaymentStats), args.Error(1)
}

// Transaction repository methods

func (m *MockRepository) CreateTransaction(ctx context.Context, tx *schema.Transaction) error {
	args := m.Called(ctx, tx)
	return args.Error(0)
}

func (m *MockRepository) CreateTransactionTx(ctx context.Context, db *gorm.DB, tx *schema.Transaction) error {
	args := m.Called(ctx, db, tx)
	return args.Error(0)
}

func (m *MockRepository) GetTransactionByID(ctx context.Context, id uuid.UUID) (*schema.Transaction, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*schema.Transaction), args.Error(1)
}

func (m *MockRepository) GetTransactionByReference(ctx context.Context, reference string) (*schema.Transaction, error) {
	args := m.Called(ctx, reference)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*schema.Transaction), args.Error(1)
}

func (m *MockRepository) GetTransactionByProviderTxID(ctx context.Context, providerTxID string) (*schema.Transaction, error) {
	args := m.Called(ctx, providerTxID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*schema.Transaction), args.Error(1)
}

func (m *MockRepository) UpdateTransaction(ctx context.Context, tx *schema.Transaction) error {
	args := m.Called(ctx, tx)
	return args.Error(0)
}

func (m *MockRepository) ListTransactionsByPaymentID(ctx context.Context, paymentID uuid.UUID) ([]*schema.Transaction, error) {
	args := m.Called(ctx, paymentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*schema.Transaction), args.Error(1)
}

func (m *MockRepository) ListTransactionsByBookingID(ctx context.Context, bookingID uuid.UUID) ([]*schema.Transaction, error) {
	args := m.Called(ctx, bookingID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*schema.Transaction), args.Error(1)
}

func (m *MockRepository) ListTransactionsByType(ctx context.Context, txType schema.TransactionType, limit, offset int) ([]*schema.Transaction, error) {
	args := m.Called(ctx, txType, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*schema.Transaction), args.Error(1)
}

// PaymentMethod repository methods

func (m *MockRepository) CreatePaymentMethod(ctx context.Context, pm *schema.PaymentMethod) error {
	args := m.Called(ctx, pm)
	return args.Error(0)
}

func (m *MockRepository) GetPaymentMethodByID(ctx context.Context, id uuid.UUID) (*schema.PaymentMethod, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*schema.PaymentMethod), args.Error(1)
}

func (m *MockRepository) GetPaymentMethodByAuthCode(ctx context.Context, authCode string) (*schema.PaymentMethod, error) {
	args := m.Called(ctx, authCode)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*schema.PaymentMethod), args.Error(1)
}

func (m *MockRepository) UpdatePaymentMethod(ctx context.Context, pm *schema.PaymentMethod) error {
	args := m.Called(ctx, pm)
	return args.Error(0)
}

func (m *MockRepository) DeletePaymentMethod(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRepository) ListPaymentMethodsByUserID(ctx context.Context, userID uuid.UUID) ([]*schema.PaymentMethod, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*schema.PaymentMethod), args.Error(1)
}

func (m *MockRepository) GetDefaultPaymentMethod(ctx context.Context, userID uuid.UUID) (*schema.PaymentMethod, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*schema.PaymentMethod), args.Error(1)
}

func (m *MockRepository) SetDefaultPaymentMethod(ctx context.Context, userID, methodID uuid.UUID) error {
	args := m.Called(ctx, userID, methodID)
	return args.Error(0)
}

func (m *MockRepository) DeactivatePaymentMethod(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// PayoutDetail repository methods

func (m *MockRepository) CreatePayoutDetail(ctx context.Context, pd *schema.PayoutDetail) error {
	args := m.Called(ctx, pd)
	return args.Error(0)
}

func (m *MockRepository) GetPayoutDetailByID(ctx context.Context, id uuid.UUID) (*schema.PayoutDetail, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*schema.PayoutDetail), args.Error(1)
}

func (m *MockRepository) GetPayoutDetailByRecipientCode(ctx context.Context, recipientCode string) (*schema.PayoutDetail, error) {
	args := m.Called(ctx, recipientCode)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*schema.PayoutDetail), args.Error(1)
}

func (m *MockRepository) UpdatePayoutDetail(ctx context.Context, pd *schema.PayoutDetail) error {
	args := m.Called(ctx, pd)
	return args.Error(0)
}

func (m *MockRepository) DeletePayoutDetail(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRepository) ListPayoutDetailsByUserID(ctx context.Context, userID uuid.UUID) ([]*schema.PayoutDetail, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*schema.PayoutDetail), args.Error(1)
}

func (m *MockRepository) ListPayoutDetailsByBusinessID(ctx context.Context, businessID uuid.UUID) ([]*schema.PayoutDetail, error) {
	args := m.Called(ctx, businessID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*schema.PayoutDetail), args.Error(1)
}

func (m *MockRepository) GetDefaultPayoutDetail(ctx context.Context, userID *uuid.UUID, businessID *uuid.UUID) (*schema.PayoutDetail, error) {
	args := m.Called(ctx, userID, businessID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*schema.PayoutDetail), args.Error(1)
}

func (m *MockRepository) SetDefaultPayoutDetail(ctx context.Context, id uuid.UUID, userID *uuid.UUID, businessID *uuid.UUID) error {
	args := m.Called(ctx, id, userID, businessID)
	return args.Error(0)
}
