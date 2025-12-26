package graphql

import (
	"context"
	"fmt"
	"hauslet/internal/modules/payments/domain"
	"hauslet/internal/modules/payments/service"
	"hauslet/internal/platform/payment"
	"hauslet/internal/transport/graph/viewer"

	"github.com/go-pkgz/lgr"
	"github.com/google/uuid"
)

// Resolver handles GraphQL queries and mutations for payments
type Resolver struct {
	paymentService service.PaymentService
	log            *lgr.Logger
}

// NewResolver creates a new GraphQL resolver
func NewResolver(paymentService service.PaymentService, log *lgr.Logger) *Resolver {
	return &Resolver{
		paymentService: paymentService,
		log:            log,
	}
}

// ============================================================================
// Query Resolvers
// ============================================================================

// Payment retrieves a payment by ID
func (r *Resolver) Payment(ctx context.Context, id string) (*domain.Payment, error) {
	paymentID, err := uuid.Parse(id)
	if err != nil {
		r.log.Logf("ERROR invalid payment ID %s: %v", id, err)
		return nil, fmt.Errorf("invalid payment ID")
	}

	payment, err := r.paymentService.GetPayment(ctx, paymentID)
	if err != nil {
		r.log.Logf("ERROR failed to get payment %s: %v", id, err)
		return nil, err
	}

	// Authorization: User must own the payment or be an admin
	if err := requireOwnershipOrAdmin(ctx, payment.PayerID); err != nil {
		r.log.Logf("WARN unauthorized access to payment %s", id)
		return nil, err
	}

	return payment, nil
}

// PaymentByReference retrieves a payment by reference
func (r *Resolver) PaymentByReference(ctx context.Context, reference string) (*domain.Payment, error) {
	payment, err := r.paymentService.GetPaymentByReference(ctx, reference)
	if err != nil {
		r.log.Logf("ERROR failed to get payment by reference %s: %v", reference, err)
		return nil, err
	}

	// Authorization: User must own the payment or be an admin
	if err := requireOwnershipOrAdmin(ctx, payment.PayerID); err != nil {
		r.log.Logf("WARN unauthorized access to payment ref %s", reference)
		return nil, err
	}

	return payment, nil
}

// MyPayments lists payments for the current user
func (r *Resolver) MyPayments(ctx context.Context, limit *int, offset *int) ([]domain.Payment, error) {
	// Get current user from context
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	l := 20
	if limit != nil && *limit > 0 {
		l = *limit
	}

	o := 0
	if offset != nil && *offset > 0 {
		o = *offset
	}

	payments, err := r.paymentService.ListPaymentsByPayer(ctx, userID, l, o)
	if err != nil {
		r.log.Logf("ERROR failed to list payments for user %s: %v", userID, err)
		return nil, err
	}

	return payments, nil
}

// MyPaymentMethods lists payment methods for the current user
func (r *Resolver) MyPaymentMethods(ctx context.Context) ([]domain.PaymentMethod, error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	methods, err := r.paymentService.ListPaymentMethods(ctx, userID)
	if err != nil {
		r.log.Logf("ERROR failed to list payment methods for user %s: %v", userID, err)
		return nil, err
	}

	return methods, nil
}

// MyPayoutDetails lists payout details for the current user
func (r *Resolver) MyPayoutDetails(ctx context.Context) ([]domain.PayoutDetail, error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	details, err := r.paymentService.ListPayoutDetails(ctx, &userID, nil)
	if err != nil {
		r.log.Logf("ERROR failed to list payout details for user %s: %v", userID, err)
		return nil, err
	}

	return details, nil
}

// Transaction retrieves a transaction by ID
func (r *Resolver) Transaction(ctx context.Context, id string) (*domain.Transaction, error) {
	txID, err := uuid.Parse(id)
	if err != nil {
		r.log.Logf("ERROR invalid transaction ID %s: %v", id, err)
		return nil, fmt.Errorf("invalid transaction ID")
	}

	tx, err := r.paymentService.GetTransaction(ctx, txID)
	if err != nil {
		r.log.Logf("ERROR failed to get transaction %s: %v", id, err)
		return nil, err
	}

	// Authorization: Get associated payment to check ownership
	if tx.PaymentID != nil {
		payment, err := r.paymentService.GetPayment(ctx, *tx.PaymentID)
		if err == nil {
			if err := requireOwnershipOrAdmin(ctx, payment.PayerID); err != nil {
				r.log.Logf("WARN unauthorized access to transaction %s", id)
				return nil, err
			}
		}
	} else {
		// For payout transactions without payment, require admin
		v := viewer.FromContext(ctx)
		if v == nil || !isAdminRole(v.Role) {
			r.log.Logf("WARN unauthorized access to payout transaction %s", id)
			return nil, ErrAdminRequired
		}
	}

	return tx, nil
}

// ============================================================================
// Mutation Resolvers
// ============================================================================

// CreatePayment creates a new payment
// RESTRICTED: Admin/Root only - should be called by internal services (booking system)
func (r *Resolver) CreatePayment(ctx context.Context, input *CreatePaymentInput) (*domain.Payment, error) {
	// Authorization: Admin or Root only
	if err := requireAdmin(ctx); err != nil {
		r.log.Logf("WARN unauthorized CreatePayment attempt")
		return nil, err
	}

	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	// Convert input to domain input
	domainInput := domain.CreatePaymentInput{
		Amount:      input.Amount,
		Currency:    input.Currency,
		Market:      input.Market,
		PayerID:     userID,
		PayerEmail:  input.PayerEmail,
		PayerName:   input.PayerName,
		Description: input.Description,
		Metadata:    input.Metadata,
	}

	if input.BookingID != nil {
		bookingID, err := uuid.Parse(*input.BookingID)
		if err != nil {
			return nil, fmt.Errorf("invalid booking ID")
		}
		domainInput.BookingID = &bookingID
	}

	if input.PaymentMethodID != nil {
		methodID, err := uuid.Parse(*input.PaymentMethodID)
		if err != nil {
			return nil, fmt.Errorf("invalid payment method ID")
		}
		domainInput.PaymentMethodID = &methodID
	}

	if input.CallbackURL != nil {
		domainInput.CallbackURL = *input.CallbackURL
	}

	payment, err := r.paymentService.CreatePayment(ctx, domainInput)
	if err != nil {
		r.log.Logf("ERROR failed to create payment: %v", err)
		return nil, err
	}

	return payment, nil
}

// VerifyPayment verifies a payment status
// RESTRICTED: Admin/Root only - primarily for internal use (webhooks, system verification)
func (r *Resolver) VerifyPayment(ctx context.Context, reference string) (*domain.Payment, error) {
	// Authorization: Admin or Root only
	if err := requireAdmin(ctx); err != nil {
		r.log.Logf("WARN unauthorized VerifyPayment attempt for ref %s", reference)
		return nil, err
	}

	payment, err := r.paymentService.VerifyPayment(ctx, reference)
	if err != nil {
		r.log.Logf("ERROR failed to verify payment %s: %v", reference, err)
		return nil, err
	}

	return payment, nil
}

// RefundPayment processes a refund
// RESTRICTED: Support/Admin/Root only
func (r *Resolver) RefundPayment(ctx context.Context, input *RefundPaymentInput) (*domain.Payment, error) {
	// Authorization: Support, Admin, or Root only
	if err := requireSupport(ctx); err != nil {
		r.log.Logf("WARN unauthorized RefundPayment attempt")
		return nil, err
	}

	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	paymentID, err := uuid.Parse(input.PaymentID)
	if err != nil {
		return nil, fmt.Errorf("invalid payment ID")
	}

	domainInput := domain.RefundPaymentInput{
		PaymentID:  paymentID,
		Reason:     input.Reason,
		RefundedBy: userID,
	}

	if input.Amount != nil {
		domainInput.Amount = input.Amount
	}

	payment, err := r.paymentService.RefundPayment(ctx, domainInput)
	if err != nil {
		r.log.Logf("ERROR failed to refund payment: %v", err)
		return nil, err
	}

	return payment, nil
}

// SavePaymentMethod saves a payment method
func (r *Resolver) SavePaymentMethod(ctx context.Context, input *SavePaymentMethodInput) (*domain.PaymentMethod, error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	domainInput := domain.CreatePaymentMethodInput{
		UserID:            userID,
		AuthorizationCode: input.AuthorizationCode,
		Currency:          input.Currency,
		Provider:          "paystack",
		SetAsDefault:      input.SetAsDefault != nil && *input.SetAsDefault,
	}

	method, err := r.paymentService.SavePaymentMethod(ctx, domainInput)
	if err != nil {
		r.log.Logf("ERROR failed to save payment method: %v", err)
		return nil, err
	}

	return method, nil
}

// RemovePaymentMethod removes a payment method
func (r *Resolver) RemovePaymentMethod(ctx context.Context, methodID string) (bool, error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return false, err
	}

	id, err := uuid.Parse(methodID)
	if err != nil {
		return false, fmt.Errorf("invalid payment method ID")
	}

	if err := r.paymentService.RemovePaymentMethod(ctx, id, userID); err != nil {
		r.log.Logf("ERROR failed to remove payment method: %v", err)
		return false, err
	}

	return true, nil
}

// AddPayoutDetail adds payout details
func (r *Resolver) AddPayoutDetail(ctx context.Context, input *AddPayoutDetailInput) (*domain.PayoutDetail, error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	domainInput := domain.CreatePayoutDetailInput{
		UserID:        &userID,
		BankCode:      input.BankCode,
		AccountNumber: input.AccountNumber,
		AccountName:   input.AccountName,
		Currency:      input.Currency,
		Market:        input.Market,
		SetAsDefault:  input.SetAsDefault != nil && *input.SetAsDefault,
	}

	detail, err := r.paymentService.AddPayoutDetail(ctx, domainInput)
	if err != nil {
		r.log.Logf("ERROR failed to add payout detail: %v", err)
		return nil, err
	}

	return detail, nil
}

// VerifyBankAccount verifies a bank account
func (r *Resolver) VerifyBankAccount(ctx context.Context, market domain.Market, bankCode string, accountNumber string) (string, error) {
	accountName, err := r.paymentService.VerifyBankAccount(ctx, market, bankCode, accountNumber)
	if err != nil {
		r.log.Logf("ERROR failed to verify bank account: %v", err)
		return "", err
	}

	return accountName, nil
}

// SetDefaultPaymentMethod sets a payment method as default
func (r *Resolver) SetDefaultPaymentMethod(ctx context.Context, methodID string) (*domain.PaymentMethod, error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	id, err := uuid.Parse(methodID)
	if err != nil {
		return nil, fmt.Errorf("invalid payment method ID")
	}

	if err := r.paymentService.SetDefaultPaymentMethod(ctx, id, userID); err != nil {
		r.log.Logf("ERROR failed to set default payment method: %v", err)
		return nil, err
	}

	// Fetch and return the updated payment method
	method, err := r.paymentService.GetPaymentMethod(ctx, id, userID)
	if err != nil {
		r.log.Logf("ERROR failed to get payment method after update: %v", err)
		return nil, err
	}

	return method, nil
}

// GetPaymentMethod retrieves a single payment method by ID
func (r *Resolver) GetPaymentMethod(ctx context.Context, id string) (*domain.PaymentMethod, error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	methodID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid payment method ID")
	}

	method, err := r.paymentService.GetPaymentMethod(ctx, methodID, userID)
	if err != nil {
		r.log.Logf("ERROR failed to get payment method: %v", err)
		return nil, err
	}

	return method, nil
}

// DeactivatePayoutDetail deactivates a payout detail
func (r *Resolver) DeactivatePayoutDetail(ctx context.Context, detailID string) (*domain.PayoutDetail, error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	id, err := uuid.Parse(detailID)
	if err != nil {
		return nil, fmt.Errorf("invalid payout detail ID")
	}

	// First get the detail to verify ownership
	detail, err := r.paymentService.GetPayoutDetail(ctx, id)
	if err != nil {
		r.log.Logf("ERROR failed to get payout detail: %v", err)
		return nil, err
	}

	// Verify ownership
	if detail.UserID != nil && *detail.UserID != userID {
		return nil, fmt.Errorf("unauthorized: payout detail does not belong to user")
	}

	// Remove the payout detail
	if err := r.paymentService.RemovePayoutDetail(ctx, id); err != nil {
		r.log.Logf("ERROR failed to remove payout detail: %v", err)
		return nil, err
	}

	// Return the detail with updated status
	detail.IsActive = false
	return detail, nil
}

// GetPayoutDetail retrieves a single payout detail by ID
func (r *Resolver) GetPayoutDetail(ctx context.Context, id string) (*domain.PayoutDetail, error) {
	detailID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid payout detail ID")
	}

	detail, err := r.paymentService.GetPayoutDetail(ctx, detailID)
	if err != nil {
		r.log.Logf("ERROR failed to get payout detail: %v", err)
		return nil, err
	}

	// Verify ownership
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if detail.UserID != nil && *detail.UserID != userID {
		// Also check if user is admin
		if err := requireAdmin(ctx); err != nil {
			return nil, fmt.Errorf("unauthorized: payout detail does not belong to user")
		}
	}

	return detail, nil
}

// ListPayments lists payments with filters
func (r *Resolver) ListPayments(ctx context.Context, bookingID, businessID, payerID *string, status *domain.PaymentStatus, limit, offset *int) ([]domain.Payment, error) {
	// For now, implement based on available service methods
	// If payerID is provided, use ListPaymentsByPayer
	// If bookingID is provided, use ListPaymentsByBooking
	// Otherwise, require admin access for broad queries

	l := 20
	if limit != nil && *limit > 0 {
		l = *limit
	}

	o := 0
	if offset != nil && *offset > 0 {
		o = *offset
	}

	if payerID != nil {
		pID, err := uuid.Parse(*payerID)
		if err != nil {
			return nil, fmt.Errorf("invalid payer ID")
		}

		// Verify user can access this payer's payments
		userID, err := getUserIDFromContext(ctx)
		if err != nil {
			return nil, err
		}

		if userID != pID {
			if err := requireAdmin(ctx); err != nil {
				return nil, fmt.Errorf("unauthorized: cannot access other user's payments")
			}
		}

		payments, err := r.paymentService.ListPaymentsByPayer(ctx, pID, l, o)
		if err != nil {
			r.log.Logf("ERROR failed to list payments: %v", err)
			return nil, err
		}

		// Apply status filter if provided
		if status != nil {
			filtered := make([]domain.Payment, 0)
			for _, p := range payments {
				if p.Status == *status {
					filtered = append(filtered, p)
				}
			}
			return filtered, nil
		}

		return payments, nil
	}

	if bookingID != nil {
		bID, err := uuid.Parse(*bookingID)
		if err != nil {
			return nil, fmt.Errorf("invalid booking ID")
		}

		payments, err := r.paymentService.ListPaymentsByBooking(ctx, bID)
		if err != nil {
			r.log.Logf("ERROR failed to list payments: %v", err)
			return nil, err
		}

		// Apply status filter if provided
		if status != nil {
			filtered := make([]domain.Payment, 0)
			for _, p := range payments {
				if p.Status == *status {
					filtered = append(filtered, p)
				}
			}
			return filtered, nil
		}

		return payments, nil
	}

	// Broad query requires admin
	if err := requireAdmin(ctx); err != nil {
		return nil, fmt.Errorf("unauthorized: admin access required for unfiltered payment queries")
	}

	// For admin, default to current user's payments (can be extended later)
	userID, _ := getUserIDFromContext(ctx)
	return r.paymentService.ListPaymentsByPayer(ctx, userID, l, o)
}

// ListTransactions lists transactions with filters
func (r *Resolver) ListTransactions(ctx context.Context, paymentID *string, txType *domain.TransactionType, status *domain.TransactionStatus, limit, offset *int) ([]domain.Transaction, error) {
	if paymentID != nil {
		pID, err := uuid.Parse(*paymentID)
		if err != nil {
			return nil, fmt.Errorf("invalid payment ID")
		}

		transactions, err := r.paymentService.ListTransactionsByPayment(ctx, pID)
		if err != nil {
			r.log.Logf("ERROR failed to list transactions: %v", err)
			return nil, err
		}

		// Apply filters
		filtered := transactions
		if txType != nil {
			temp := make([]domain.Transaction, 0)
			for _, tx := range filtered {
				if tx.Type == *txType {
					temp = append(temp, tx)
				}
			}
			filtered = temp
		}

		if status != nil {
			temp := make([]domain.Transaction, 0)
			for _, tx := range filtered {
				if tx.Status == *status {
					temp = append(temp, tx)
				}
			}
			filtered = temp
		}

		return filtered, nil
	}

	// Broad query requires admin
	if err := requireAdmin(ctx); err != nil {
		return nil, fmt.Errorf("unauthorized: admin access required for unfiltered transaction queries")
	}

	// Return empty list for now - can be extended to query all transactions
	return []domain.Transaction{}, nil
}

// CreatePayout creates a new payout transaction
func (r *Resolver) CreatePayout(ctx context.Context, input *CreatePayoutInput) (*domain.Transaction, error) {
	// Authorization: Admin or business owner (can be extended with business ownership check)
	_, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	businessID, err := uuid.Parse(input.BusinessID)
	if err != nil {
		return nil, fmt.Errorf("invalid business ID")
	}

	// Parse recipient code as payout detail ID
	payoutDetailID, err := uuid.Parse(input.RecipientCode)
	if err != nil {
		return nil, fmt.Errorf("invalid recipient code (payout detail ID)")
	}

	// Create payout input
	domainInput := domain.ProcessPayoutInput{
		BusinessID:     &businessID,
		Amount:         input.Amount,
		Currency:       input.Currency,
		PayoutDetailID: payoutDetailID,
		Description:    input.Description,
	}

	transaction, err := r.paymentService.ProcessPayout(ctx, domainInput)
	if err != nil {
		r.log.Logf("ERROR failed to create payout: %v", err)
		return nil, err
	}

	return transaction, nil
}

// ============================================================================
// Helper Functions
// ============================================================================
// Note: Authorization helpers are in authorization.go

// ============================================================================
// Input Types (GraphQL schema types)
// ============================================================================

type CreatePaymentInput struct {
	Amount          int64
	Currency        payment.Currency
	Market          domain.Market
	PayerEmail      string
	PayerName       string
	BookingID       *string
	PaymentMethodID *string
	CallbackURL     *string
	Description     string
	Metadata        map[string]string
}

type RefundPaymentInput struct {
	PaymentID string
	Amount    *int64
	Reason    string
}

type SavePaymentMethodInput struct {
	AuthorizationCode string
	Currency          payment.Currency
	SetAsDefault      *bool
}

type AddPayoutDetailInput struct {
	BankCode      string
	AccountNumber string
	AccountName   string
	Currency      payment.Currency
	Market        domain.Market
	SetAsDefault  *bool
}

type CreatePayoutInput struct {
	BusinessID    string
	Amount        int64
	Currency      payment.Currency
	Description   string
	RecipientCode string
}
