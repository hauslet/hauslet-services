package repository

import (
	"context"
	"fmt"
	"hauslet/internal/modules/payments/repository/schema"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// RepositoryImpl implements all payment repository interfaces
type RepositoryImpl struct {
	db *gorm.DB
}

// NewRepository creates a new payment repository
func NewRepository(db *gorm.DB) Repository {
	return &RepositoryImpl{db: db}
}

// ============================================================================
// PaymentRepository Implementation
// ============================================================================

// CreatePayment creates a new payment record
func (r *RepositoryImpl) CreatePayment(ctx context.Context, payment *schema.Payment) error {
	if err := r.db.WithContext(ctx).Create(payment).Error; err != nil {
		return fmt.Errorf("failed to create payment: %w", err)
	}
	return nil
}

// GetPaymentByID retrieves a payment by ID
func (r *RepositoryImpl) GetPaymentByID(ctx context.Context, id uuid.UUID) (*schema.Payment, error) {
	var payment schema.Payment
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&payment).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get payment: %w", err)
	}
	return &payment, nil
}

// GetPaymentByReference retrieves a payment by reference
func (r *RepositoryImpl) GetPaymentByReference(ctx context.Context, reference string) (*schema.Payment, error) {
	var payment schema.Payment
	if err := r.db.WithContext(ctx).Where("reference = ?", reference).First(&payment).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get payment by reference: %w", err)
	}
	return &payment, nil
}

// UpdatePayment updates an existing payment
func (r *RepositoryImpl) UpdatePayment(ctx context.Context, payment *schema.Payment) error {
	if err := r.db.WithContext(ctx).Save(payment).Error; err != nil {
		return fmt.Errorf("failed to update payment: %w", err)
	}
	return nil
}

// DeletePayment soft deletes a payment
func (r *RepositoryImpl) DeletePayment(ctx context.Context, id uuid.UUID) error {
	if err := r.db.WithContext(ctx).Where("id = ?", id).Delete(&schema.Payment{}).Error; err != nil {
		return fmt.Errorf("failed to delete payment: %w", err)
	}
	return nil
}

// ListPaymentsByPayerID lists payments by payer ID with pagination
func (r *RepositoryImpl) ListPaymentsByPayerID(ctx context.Context, payerID uuid.UUID, limit, offset int) ([]*schema.Payment, error) {
	var payments []*schema.Payment
	query := r.db.WithContext(ctx).
		Where("payer_id = ?", payerID).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Find(&payments).Error; err != nil {
		return nil, fmt.Errorf("failed to list payments by payer: %w", err)
	}
	return payments, nil
}

// ListPaymentsByBookingID lists payments for a booking
func (r *RepositoryImpl) ListPaymentsByBookingID(ctx context.Context, bookingID uuid.UUID) ([]*schema.Payment, error) {
	var payments []*schema.Payment
	if err := r.db.WithContext(ctx).
		Where("booking_id = ?", bookingID).
		Order("created_at DESC").
		Find(&payments).Error; err != nil {
		return nil, fmt.Errorf("failed to list payments by booking: %w", err)
	}
	return payments, nil
}

// ListPaymentsByBusinessID lists payments for a business with pagination
func (r *RepositoryImpl) ListPaymentsByBusinessID(ctx context.Context, businessID uuid.UUID, limit, offset int) ([]*schema.Payment, error) {
	var payments []*schema.Payment
	query := r.db.WithContext(ctx).
		Where("business_id = ?", businessID).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Find(&payments).Error; err != nil {
		return nil, fmt.Errorf("failed to list payments by business: %w", err)
	}
	return payments, nil
}

// ListPaymentsByStatus lists payments by status with pagination
func (r *RepositoryImpl) ListPaymentsByStatus(ctx context.Context, status schema.PaymentStatus, limit, offset int) ([]*schema.Payment, error) {
	var payments []*schema.Payment
	query := r.db.WithContext(ctx).
		Where("status = ?", status).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Find(&payments).Error; err != nil {
		return nil, fmt.Errorf("failed to list payments by status: %w", err)
	}
	return payments, nil
}

// WithinTransaction executes a function within a database transaction
func (r *RepositoryImpl) WithinTransaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return r.db.WithContext(ctx).Transaction(fn)
}

// ============================================================================
// TransactionRepository Implementation
// ============================================================================

// CreateTransaction creates a new transaction record
func (r *RepositoryImpl) CreateTransaction(ctx context.Context, tx *schema.Transaction) error {
	if err := r.db.WithContext(ctx).Create(tx).Error; err != nil {
		return fmt.Errorf("failed to create transaction: %w", err)
	}
	return nil
}

// CreateTransactionTx creates a transaction within an existing DB transaction
func (r *RepositoryImpl) CreateTransactionTx(ctx context.Context, db *gorm.DB, tx *schema.Transaction) error {
	if err := db.WithContext(ctx).Create(tx).Error; err != nil {
		return fmt.Errorf("failed to create transaction: %w", err)
	}
	return nil
}

// GetTransactionByID retrieves a transaction by ID
func (r *RepositoryImpl) GetTransactionByID(ctx context.Context, id uuid.UUID) (*schema.Transaction, error) {
	var tx schema.Transaction
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&tx).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}
	return &tx, nil
}

// GetTransactionByReference retrieves a transaction by reference
func (r *RepositoryImpl) GetTransactionByReference(ctx context.Context, reference string) (*schema.Transaction, error) {
	var tx schema.Transaction
	if err := r.db.WithContext(ctx).Where("reference = ?", reference).First(&tx).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get transaction by reference: %w", err)
	}
	return &tx, nil
}

// UpdateTransaction updates an existing transaction
func (r *RepositoryImpl) UpdateTransaction(ctx context.Context, tx *schema.Transaction) error {
	if err := r.db.WithContext(ctx).Save(tx).Error; err != nil {
		return fmt.Errorf("failed to update transaction: %w", err)
	}
	return nil
}

// ListTransactionsByPaymentID lists transactions for a payment
func (r *RepositoryImpl) ListTransactionsByPaymentID(ctx context.Context, paymentID uuid.UUID) ([]*schema.Transaction, error) {
	var transactions []*schema.Transaction
	if err := r.db.WithContext(ctx).
		Where("payment_id = ?", paymentID).
		Order("created_at DESC").
		Find(&transactions).Error; err != nil {
		return nil, fmt.Errorf("failed to list transactions by payment: %w", err)
	}
	return transactions, nil
}

// ListTransactionsByBookingID lists transactions for a booking
func (r *RepositoryImpl) ListTransactionsByBookingID(ctx context.Context, bookingID uuid.UUID) ([]*schema.Transaction, error) {
	var transactions []*schema.Transaction
	if err := r.db.WithContext(ctx).
		Where("booking_id = ?", bookingID).
		Order("created_at DESC").
		Find(&transactions).Error; err != nil {
		return nil, fmt.Errorf("failed to list transactions by booking: %w", err)
	}
	return transactions, nil
}

// ListTransactionsByType lists transactions by type with pagination
func (r *RepositoryImpl) ListTransactionsByType(ctx context.Context, txType schema.TransactionType, limit, offset int) ([]*schema.Transaction, error) {
	var transactions []*schema.Transaction
	query := r.db.WithContext(ctx).
		Where("type = ?", txType).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Find(&transactions).Error; err != nil {
		return nil, fmt.Errorf("failed to list transactions by type: %w", err)
	}
	return transactions, nil
}

// ============================================================================
// PaymentMethodRepository Implementation
// ============================================================================

// CreatePaymentMethod creates a new payment method
func (r *RepositoryImpl) CreatePaymentMethod(ctx context.Context, pm *schema.PaymentMethod) error {
	if err := r.db.WithContext(ctx).Create(pm).Error; err != nil {
		return fmt.Errorf("failed to create payment method: %w", err)
	}
	return nil
}

// GetPaymentMethodByID retrieves a payment method by ID
func (r *RepositoryImpl) GetPaymentMethodByID(ctx context.Context, id uuid.UUID) (*schema.PaymentMethod, error) {
	var pm schema.PaymentMethod
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&pm).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get payment method: %w", err)
	}
	return &pm, nil
}

// GetPaymentMethodByAuthCode retrieves a payment method by authorization code
func (r *RepositoryImpl) GetPaymentMethodByAuthCode(ctx context.Context, authCode string) (*schema.PaymentMethod, error) {
	var pm schema.PaymentMethod
	if err := r.db.WithContext(ctx).Where("authorization_code = ?", authCode).First(&pm).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get payment method by auth code: %w", err)
	}
	return &pm, nil
}

// UpdatePaymentMethod updates an existing payment method
func (r *RepositoryImpl) UpdatePaymentMethod(ctx context.Context, pm *schema.PaymentMethod) error {
	if err := r.db.WithContext(ctx).Save(pm).Error; err != nil {
		return fmt.Errorf("failed to update payment method: %w", err)
	}
	return nil
}

// DeletePaymentMethod soft deletes a payment method
func (r *RepositoryImpl) DeletePaymentMethod(ctx context.Context, id uuid.UUID) error {
	if err := r.db.WithContext(ctx).Where("id = ?", id).Delete(&schema.PaymentMethod{}).Error; err != nil {
		return fmt.Errorf("failed to delete payment method: %w", err)
	}
	return nil
}

// ListPaymentMethodsByUserID lists payment methods for a user
func (r *RepositoryImpl) ListPaymentMethodsByUserID(ctx context.Context, userID uuid.UUID) ([]*schema.PaymentMethod, error) {
	var methods []*schema.PaymentMethod
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND is_active = ?", userID, true).
		Order("is_default DESC, created_at DESC").
		Find(&methods).Error; err != nil {
		return nil, fmt.Errorf("failed to list payment methods: %w", err)
	}
	return methods, nil
}

// GetDefaultPaymentMethod gets the default payment method for a user
func (r *RepositoryImpl) GetDefaultPaymentMethod(ctx context.Context, userID uuid.UUID) (*schema.PaymentMethod, error) {
	var pm schema.PaymentMethod
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND is_default = ? AND is_active = ?", userID, true, true).
		First(&pm).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get default payment method: %w", err)
	}
	return &pm, nil
}

// SetDefaultPaymentMethod sets a payment method as default
func (r *RepositoryImpl) SetDefaultPaymentMethod(ctx context.Context, userID, methodID uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Unset all other defaults for this user
		if err := tx.Model(&schema.PaymentMethod{}).
			Where("user_id = ?", userID).
			Update("is_default", false).Error; err != nil {
			return fmt.Errorf("failed to unset default payment methods: %w", err)
		}

		// Set the new default
		if err := tx.Model(&schema.PaymentMethod{}).
			Where("id = ?", methodID).
			Update("is_default", true).Error; err != nil {
			return fmt.Errorf("failed to set default payment method: %w", err)
		}

		return nil
	})
}

// DeactivatePaymentMethod deactivates a payment method
func (r *RepositoryImpl) DeactivatePaymentMethod(ctx context.Context, id uuid.UUID) error {
	if err := r.db.WithContext(ctx).
		Model(&schema.PaymentMethod{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"is_active":  false,
			"is_default": false,
		}).Error; err != nil {
		return fmt.Errorf("failed to deactivate payment method: %w", err)
	}
	return nil
}

// ============================================================================
// PayoutDetailRepository Implementation
// ============================================================================

// CreatePayoutDetail creates a new payout detail
func (r *RepositoryImpl) CreatePayoutDetail(ctx context.Context, pd *schema.PayoutDetail) error {
	if err := r.db.WithContext(ctx).Create(pd).Error; err != nil {
		return fmt.Errorf("failed to create payout detail: %w", err)
	}
	return nil
}

// GetPayoutDetailByID retrieves a payout detail by ID
func (r *RepositoryImpl) GetPayoutDetailByID(ctx context.Context, id uuid.UUID) (*schema.PayoutDetail, error) {
	var pd schema.PayoutDetail
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&pd).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get payout detail: %w", err)
	}
	return &pd, nil
}

// GetPayoutDetailByRecipientCode retrieves a payout detail by recipient code
func (r *RepositoryImpl) GetPayoutDetailByRecipientCode(ctx context.Context, recipientCode string) (*schema.PayoutDetail, error) {
	var pd schema.PayoutDetail
	if err := r.db.WithContext(ctx).Where("recipient_code = ?", recipientCode).First(&pd).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get payout detail by recipient code: %w", err)
	}
	return &pd, nil
}

// UpdatePayoutDetail updates an existing payout detail
func (r *RepositoryImpl) UpdatePayoutDetail(ctx context.Context, pd *schema.PayoutDetail) error {
	if err := r.db.WithContext(ctx).Save(pd).Error; err != nil {
		return fmt.Errorf("failed to update payout detail: %w", err)
	}
	return nil
}

// DeletePayoutDetail soft deletes a payout detail
func (r *RepositoryImpl) DeletePayoutDetail(ctx context.Context, id uuid.UUID) error {
	if err := r.db.WithContext(ctx).Where("id = ?", id).Delete(&schema.PayoutDetail{}).Error; err != nil {
		return fmt.Errorf("failed to delete payout detail: %w", err)
	}
	return nil
}

// ListPayoutDetailsByUserID lists payout details for a user
func (r *RepositoryImpl) ListPayoutDetailsByUserID(ctx context.Context, userID uuid.UUID) ([]*schema.PayoutDetail, error) {
	var details []*schema.PayoutDetail
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND is_active = ?", userID, true).
		Order("is_default DESC, created_at DESC").
		Find(&details).Error; err != nil {
		return nil, fmt.Errorf("failed to list payout details by user: %w", err)
	}
	return details, nil
}

// ListPayoutDetailsByBusinessID lists payout details for a business
func (r *RepositoryImpl) ListPayoutDetailsByBusinessID(ctx context.Context, businessID uuid.UUID) ([]*schema.PayoutDetail, error) {
	var details []*schema.PayoutDetail
	if err := r.db.WithContext(ctx).
		Where("business_id = ? AND is_active = ?", businessID, true).
		Order("is_default DESC, created_at DESC").
		Find(&details).Error; err != nil {
		return nil, fmt.Errorf("failed to list payout details by business: %w", err)
	}
	return details, nil
}

// GetDefaultPayoutDetail gets the default payout detail for user or business
func (r *RepositoryImpl) GetDefaultPayoutDetail(ctx context.Context, userID *uuid.UUID, businessID *uuid.UUID) (*schema.PayoutDetail, error) {
	var pd schema.PayoutDetail
	query := r.db.WithContext(ctx).Where("is_default = ? AND is_active = ?", true, true)

	if userID != nil {
		query = query.Where("user_id = ?", *userID)
	}
	if businessID != nil {
		query = query.Where("business_id = ?", *businessID)
	}

	if err := query.First(&pd).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get default payout detail: %w", err)
	}
	return &pd, nil
}

// SetDefaultPayoutDetail sets a payout detail as default
func (r *RepositoryImpl) SetDefaultPayoutDetail(ctx context.Context, id uuid.UUID, userID *uuid.UUID, businessID *uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Build query to unset other defaults
		query := tx.Model(&schema.PayoutDetail{})
		if userID != nil {
			query = query.Where("user_id = ?", *userID)
		}
		if businessID != nil {
			query = query.Where("business_id = ?", *businessID)
		}

		// Unset all other defaults
		if err := query.Update("is_default", false).Error; err != nil {
			return fmt.Errorf("failed to unset default payout details: %w", err)
		}

		// Set the new default
		if err := tx.Model(&schema.PayoutDetail{}).
			Where("id = ?", id).
			Update("is_default", true).Error; err != nil {
			return fmt.Errorf("failed to set default payout detail: %w", err)
		}

		return nil
	})
}
