package repository

import (
	"context"
	"hauslet/internal/modules/payments/repository/schema"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PaymentRepository defines the interface for payment data access
type PaymentRepository interface {
	// Payment CRUD operations
	CreatePayment(ctx context.Context, payment *schema.Payment) error
	GetPaymentByID(ctx context.Context, id uuid.UUID) (*schema.Payment, error)
	GetPaymentByReference(ctx context.Context, reference string) (*schema.Payment, error)
	UpdatePayment(ctx context.Context, payment *schema.Payment) error
	DeletePayment(ctx context.Context, id uuid.UUID) error

	// Payment queries
	ListPaymentsByPayerID(ctx context.Context, payerID uuid.UUID, limit, offset int) ([]*schema.Payment, error)
	ListPaymentsByBookingID(ctx context.Context, bookingID uuid.UUID) ([]*schema.Payment, error)
	ListPaymentsByBusinessID(ctx context.Context, businessID uuid.UUID, limit, offset int) ([]*schema.Payment, error)
	ListPaymentsByStatus(ctx context.Context, status schema.PaymentStatus, limit, offset int) ([]*schema.Payment, error)

	// Transaction support
	WithinTransaction(ctx context.Context, fn func(tx *gorm.DB) error) error
}

// TransactionRepository defines the interface for transaction data access
type TransactionRepository interface {
	// Transaction CRUD operations
	CreateTransaction(ctx context.Context, tx *schema.Transaction) error
	CreateTransactionTx(ctx context.Context, db *gorm.DB, tx *schema.Transaction) error
	GetTransactionByID(ctx context.Context, id uuid.UUID) (*schema.Transaction, error)
	GetTransactionByReference(ctx context.Context, reference string) (*schema.Transaction, error)
	GetTransactionByProviderTxID(ctx context.Context, providerTxID string) (*schema.Transaction, error)
	UpdateTransaction(ctx context.Context, tx *schema.Transaction) error

	// Transaction queries
	ListTransactionsByPaymentID(ctx context.Context, paymentID uuid.UUID) ([]*schema.Transaction, error)
	ListTransactionsByBookingID(ctx context.Context, bookingID uuid.UUID) ([]*schema.Transaction, error)
	ListTransactionsByType(ctx context.Context, txType schema.TransactionType, limit, offset int) ([]*schema.Transaction, error)
}

// PaymentMethodRepository defines the interface for payment method data access
type PaymentMethodRepository interface {
	// PaymentMethod CRUD operations
	CreatePaymentMethod(ctx context.Context, pm *schema.PaymentMethod) error
	GetPaymentMethodByID(ctx context.Context, id uuid.UUID) (*schema.PaymentMethod, error)
	GetPaymentMethodByAuthCode(ctx context.Context, authCode string) (*schema.PaymentMethod, error)
	UpdatePaymentMethod(ctx context.Context, pm *schema.PaymentMethod) error
	DeletePaymentMethod(ctx context.Context, id uuid.UUID) error

	// PaymentMethod queries
	ListPaymentMethodsByUserID(ctx context.Context, userID uuid.UUID) ([]*schema.PaymentMethod, error)
	GetDefaultPaymentMethod(ctx context.Context, userID uuid.UUID) (*schema.PaymentMethod, error)
	SetDefaultPaymentMethod(ctx context.Context, userID, methodID uuid.UUID) error
	DeactivatePaymentMethod(ctx context.Context, id uuid.UUID) error
}

// PayoutDetailRepository defines the interface for payout detail data access
type PayoutDetailRepository interface {
	// PayoutDetail CRUD operations
	CreatePayoutDetail(ctx context.Context, pd *schema.PayoutDetail) error
	GetPayoutDetailByID(ctx context.Context, id uuid.UUID) (*schema.PayoutDetail, error)
	GetPayoutDetailByRecipientCode(ctx context.Context, recipientCode string) (*schema.PayoutDetail, error)
	UpdatePayoutDetail(ctx context.Context, pd *schema.PayoutDetail) error
	DeletePayoutDetail(ctx context.Context, id uuid.UUID) error

	// PayoutDetail queries
	ListPayoutDetailsByUserID(ctx context.Context, userID uuid.UUID) ([]*schema.PayoutDetail, error)
	ListPayoutDetailsByBusinessID(ctx context.Context, businessID uuid.UUID) ([]*schema.PayoutDetail, error)
	GetDefaultPayoutDetail(ctx context.Context, userID *uuid.UUID, businessID *uuid.UUID) (*schema.PayoutDetail, error)
	SetDefaultPayoutDetail(ctx context.Context, id uuid.UUID, userID *uuid.UUID, businessID *uuid.UUID) error
}

// Repository aggregates all payment-related repositories
type Repository interface {
	PaymentRepository
	TransactionRepository
	PaymentMethodRepository
	PayoutDetailRepository
}
