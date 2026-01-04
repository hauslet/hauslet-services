package service

import (
	"context"
	"hauslet/internal/modules/payments/domain"

	"github.com/google/uuid"
)

// PaymentService defines the interface for payment business logic
type PaymentService interface {
	// Payment operations
	CreatePayment(ctx context.Context, input domain.CreatePaymentInput) (*domain.Payment, error)
	GetPayment(ctx context.Context, id uuid.UUID) (*domain.Payment, error)
	GetPaymentByReference(ctx context.Context, reference string) (*domain.Payment, error)
	VerifyPayment(ctx context.Context, reference string) (*domain.Payment, error)
	ListPaymentsByPayer(ctx context.Context, payerID uuid.UUID, limit, offset int) ([]domain.Payment, error)
	ListPaymentsByBooking(ctx context.Context, bookingID uuid.UUID) ([]domain.Payment, error)

	// Refund operations
	RefundPayment(ctx context.Context, input domain.RefundPaymentInput) (*domain.Payment, error)

	// Payment method operations
	AuthorizePaymentMethod(ctx context.Context, userID uuid.UUID, email string, market domain.Market) (authURL string, reference string, err error)
	SavePaymentMethod(ctx context.Context, input domain.CreatePaymentMethodInput) (*domain.PaymentMethod, error)
	GetPaymentMethod(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*domain.PaymentMethod, error)
	GetDefaultPaymentMethod(ctx context.Context, userID uuid.UUID) (*domain.PaymentMethod, error)
	ListPaymentMethods(ctx context.Context, userID uuid.UUID) ([]domain.PaymentMethod, error)
	SetDefaultPaymentMethod(ctx context.Context, methodID, userID uuid.UUID) error
	RemovePaymentMethod(ctx context.Context, methodID, userID uuid.UUID) error

	// Payout detail operations
	AddPayoutDetail(ctx context.Context, input domain.CreatePayoutDetailInput) (*domain.PayoutDetail, error)
	GetPayoutDetail(ctx context.Context, id uuid.UUID) (*domain.PayoutDetail, error)
	ListPayoutDetails(ctx context.Context, userID *uuid.UUID, businessID *uuid.UUID) ([]domain.PayoutDetail, error)
	ListPayoutDetailsByUserID(ctx context.Context, ownerID uuid.UUID) ([]domain.PayoutDetail, error)
	SetDefaultPayoutDetail(ctx context.Context, detailID uuid.UUID, userID *uuid.UUID, businessID *uuid.UUID) error
	RemovePayoutDetail(ctx context.Context, detailID uuid.UUID) error
	VerifyBankAccount(ctx context.Context, market domain.Market, bankCode, accountNumber string) (string, error)

	// Payout operations
	ProcessPayout(ctx context.Context, input domain.ProcessPayoutInput) (*domain.Transaction, error)

	// Transaction queries
	GetTransaction(ctx context.Context, id uuid.UUID) (*domain.Transaction, error)
	ListTransactionsByPayment(ctx context.Context, paymentID uuid.UUID) ([]domain.Transaction, error)
	ListTransactionsByBooking(ctx context.Context, bookingID uuid.UUID) ([]domain.Transaction, error)
}
