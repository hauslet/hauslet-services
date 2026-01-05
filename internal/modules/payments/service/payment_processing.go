package service

import (
	"context"
	"fmt"
	"hauslet/internal/modules/payments/domain"
	"hauslet/internal/modules/payments/repository/schema"
	"hauslet/internal/platform/payment"
	"time"

	"github.com/google/uuid"
)

// CreatePayment creates a new payment and processes it
func (s *PaymentServiceImpl) CreatePayment(ctx context.Context, input domain.CreatePaymentInput) (*domain.Payment, error) {
	s.log.Info("creating payment", "amount", input.Amount, "currency", input.Currency, "payer", input.PayerID)

	// Validate input
	if err := input.Validate(); err != nil {
		s.log.Error("payment validation failed", "error", err)
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Auto-sync polymorphic fields with explicit fields for consistency
	if input.BookingID != nil && (input.ResourceType == domain.ResourceTypeGeneral || input.ResourceType == "") {
		s.log.Info("auto-syncing booking payment", "resource_type", "booking", "resource_id", input.BookingID)
		input.ResourceType = domain.ResourceTypeBooking
		input.ResourceID = input.BookingID
	}

	// Generate reference
	paymentID := uuid.New()
	reference := domain.GenerateReference("PMT", paymentID)

	// Create domain payment
	pmt := &domain.Payment{
		ID:           paymentID,
		Reference:    reference,
		BookingID:    input.BookingID,
		BusinessID:   input.BusinessID,
		ResourceType: input.ResourceType,
		ResourceID:   input.ResourceID,
		PayerID:      input.PayerID,
		PayerEmail:   input.PayerEmail,
		PayerName:    input.PayerName,
		Amount:       input.Amount,
		Currency:     input.Currency,
		Market:       input.Market,
		Status:       domain.PaymentStatusPending,
		Provider:     "paystack", // Only Paystack supported
		Description:  input.Description,
		Metadata:     input.Metadata,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Check if using saved payment method
	var paymentMethodAuthCode *string
	if input.PaymentMethodID != nil {
		method, err := s.repo.GetPaymentMethodByID(ctx, *input.PaymentMethodID)
		if err != nil {
			s.log.Error("failed to get payment method", "payment_method_id", input.PaymentMethodID, "error", err)
			return nil, fmt.Errorf("failed to get payment method: %w", err)
		}
		if method == nil {
			return nil, domain.ErrPaymentMethodNotFound
		}
		if method.UserID != input.PayerID {
			return nil, domain.ErrUnauthorized
		}
		paymentMethodAuthCode = &method.AuthorizationCode
		pmt.AuthorizationCode = paymentMethodAuthCode
		pmt.PaymentMethod = domain.PaymentMethodCard
	}

	// Process payment with provider
	var providerResp *payment.PaymentResponse
	var err error

	if paymentMethodAuthCode != nil {
		// Charge saved payment method
		s.log.Info("charging saved payment method", "reference", reference)
		providerResp, err = s.paymentClient.ChargeAuthorization(ctx, payment.PaymentRequest{
			Amount:    input.Amount,
			Currency:  input.Currency,
			Reference: reference,
			Email:     input.PayerEmail,
			AuthToken: paymentMethodAuthCode,
			Metadata:  input.Metadata,
		})
	} else {
		// Initialize new payment
		s.log.Info("initializing new payment", "reference", reference)
		providerResp, err = s.paymentClient.Initialize(ctx, payment.PaymentRequest{
			Amount:      input.Amount,
			Currency:    input.Currency,
			Reference:   reference,
			Email:       input.PayerEmail,
			CallbackURL: input.CallbackURL,
			Metadata:    input.Metadata,
		})
	}

	if err != nil {
		s.log.Error("payment processing failed", "reference", reference, "error", err)
		pmt.Status = domain.PaymentStatusFailed

		// Save failed payment
		if saveErr := s.repo.CreatePayment(ctx, domain.MapPaymentToSchema(pmt)); saveErr != nil {
			s.log.Error("failed to save failed payment", "error", saveErr)
		}

		return nil, fmt.Errorf("payment processing failed: %w", err)
	}

	// Update payment with provider response
	pmt.ProviderRef = providerResp.TransactionID
	pmt.RedirectURL = &providerResp.RedirectURL
	pmt.RequiresAction = providerResp.RequiresAction

	// Update status based on response
	switch providerResp.Status {
	case payment.StatusSuccess:
		pmt.Status = domain.PaymentStatusSucceeded
	case payment.StatusPending:
		pmt.Status = domain.PaymentStatusPending
	case payment.StatusFailed:
		pmt.Status = domain.PaymentStatusFailed
	}

	// Save payment to database
	if err := s.repo.CreatePayment(ctx, domain.MapPaymentToSchema(pmt)); err != nil {
		s.log.Error("failed to save payment", "error", err)
		return nil, fmt.Errorf("failed to save payment: %w", err)
	}

	// Create transaction record
	txID := uuid.New()
	tx := &domain.Transaction{
		ID:           txID,
		PaymentID:    &pmt.ID,
		BookingID:    pmt.BookingID,
		BusinessID:   pmt.BusinessID,
		Type:         domain.TransactionTypePayment,
		Reference:    domain.GenerateReference("TXN", txID),
		Amount:       pmt.Amount,
		Currency:     pmt.Currency,
		Status:       domain.TransactionStatus(pmt.Status),
		Provider:     pmt.Provider,
		ProviderTxID: &pmt.ProviderRef,
		Description:  pmt.Description,
		Metadata:     pmt.Metadata,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if pmt.Status == domain.PaymentStatusSucceeded {
		now := time.Now()
		tx.ProcessedAt = &now
	}

	if err := s.repo.CreateTransaction(ctx, domain.MapTransactionToSchema(tx)); err != nil {
		s.log.Warn("failed to create transaction record", "error", err)
		// Non-critical, don't fail the payment
	}

	s.log.Info("payment created successfully", "id", pmt.ID, "status", pmt.Status)

	// Send notification if payment succeeded immediately
	if pmt.Status == domain.PaymentStatusSucceeded {
		s.notificationSvc.SendPaymentReceipt(pmt)
	}

	return pmt, nil
}

// GetPayment retrieves a payment by ID
func (s *PaymentServiceImpl) GetPayment(ctx context.Context, id uuid.UUID) (*domain.Payment, error) {
	s.log.Info("fetching payment", "id", id)

	schemaPmt, err := s.repo.GetPaymentByID(ctx, id)
	if err != nil {
		s.log.Error("failed to get payment", "id", id, "error", err)
		return nil, fmt.Errorf("failed to get payment: %w", err)
	}

	if schemaPmt == nil {
		return nil, domain.ErrPaymentNotFound
	}

	return domain.MapPaymentFromSchema(schemaPmt), nil
}

// GetPaymentByReference retrieves a payment by reference
func (s *PaymentServiceImpl) GetPaymentByReference(ctx context.Context, reference string) (*domain.Payment, error) {
	s.log.Info("fetching payment by reference", "reference", reference)

	schemaPmt, err := s.repo.GetPaymentByReference(ctx, reference)
	if err != nil {
		s.log.Error("failed to get payment by reference", "reference", reference, "error", err)
		return nil, fmt.Errorf("failed to get payment: %w", err)
	}

	if schemaPmt == nil {
		return nil, domain.ErrPaymentNotFound
	}

	return domain.MapPaymentFromSchema(schemaPmt), nil
}

// VerifyPayment verifies a payment with the provider and updates status
func (s *PaymentServiceImpl) VerifyPayment(ctx context.Context, reference string) (*domain.Payment, error) {
	s.log.Info("verifying payment", "reference", reference)

	// Get payment from database
	pmt, err := s.GetPaymentByReference(ctx, reference)
	if err != nil {
		return nil, err
	}

	// If already succeeded or failed, return as is
	if pmt.Status == domain.PaymentStatusSucceeded || pmt.Status == domain.PaymentStatusFailed {
		s.log.Info("payment already finalized", "reference", reference, "status", pmt.Status)
		return pmt, nil
	}

	// Verify with provider
	s.log.Info("verifying payment with provider", "reference", reference)
	providerResp, err := s.paymentClient.Verify(ctx, pmt.Currency, reference)
	if err != nil {
		s.log.Error("provider verification failed", "reference", reference, "error", err)
		return nil, fmt.Errorf("verification failed: %w", err)
	}

	// Update payment status
	oldStatus := pmt.Status
	switch providerResp.Status {
	case payment.StatusSuccess:
		pmt.Status = domain.PaymentStatusSucceeded
	case payment.StatusFailed:
		pmt.Status = domain.PaymentStatusFailed
	case payment.StatusPending:
		pmt.Status = domain.PaymentStatusPending
	}

	pmt.ProviderRef = providerResp.TransactionID
	pmt.UpdatedAt = time.Now()

	// Save updated payment
	if err := s.repo.UpdatePayment(ctx, domain.MapPaymentToSchema(pmt)); err != nil {
		s.log.Error("failed to update payment status", "error", err)
		return nil, fmt.Errorf("failed to update payment: %w", err)
	}

	// Update transaction if status changed
	if oldStatus != pmt.Status {
		s.updateTransactionStatus(ctx, pmt)
	}

	s.log.Info("payment verified", "id", pmt.ID, "status", pmt.Status)

	// Send receipt if payment succeeded
	if pmt.Status == domain.PaymentStatusSucceeded && oldStatus != domain.PaymentStatusSucceeded {
		s.notificationSvc.SendPaymentReceipt(pmt)
	}

	return pmt, nil
}

// ListPaymentsByPayer lists payments for a payer
func (s *PaymentServiceImpl) ListPaymentsByPayer(ctx context.Context, payerID uuid.UUID, limit, offset int) ([]domain.Payment, error) {
	s.log.Info("listing payments for payer", "payer_id", payerID)

	schemaPayments, err := s.repo.ListPaymentsByPayerID(ctx, payerID, limit, offset)
	if err != nil {
		s.log.Error("failed to list payments for payer", "payer_id", payerID, "error", err)
		return nil, fmt.Errorf("failed to list payments: %w", err)
	}

	return domain.MapPaymentsFromSchema(schemaPayments), nil
}

// ListPaymentsByBooking lists payments for a booking
func (s *PaymentServiceImpl) ListPaymentsByBooking(ctx context.Context, bookingID uuid.UUID) ([]domain.Payment, error) {
	s.log.Info("listing payments for booking", "booking_id", bookingID)

	schemaPayments, err := s.repo.ListPaymentsByBookingID(ctx, bookingID)
	if err != nil {
		s.log.Error("failed to list payments for booking", "booking_id", bookingID, "error", err)
		return nil, fmt.Errorf("failed to list payments: %w", err)
	}

	return domain.MapPaymentsFromSchema(schemaPayments), nil
}

// updateTransactionStatus updates the transaction status when payment status changes
func (s *PaymentServiceImpl) updateTransactionStatus(ctx context.Context, pmt *domain.Payment) {
	transactions, err := s.repo.ListTransactionsByPaymentID(ctx, pmt.ID)
	if err != nil {
		s.log.Warn("failed to get transactions for payment", "payment_id", pmt.ID, "error", err)
		return
	}

	for _, tx := range transactions {
		if tx.Type == schema.TransactionTypePayment {
			tx.Status = schema.TransactionStatus(pmt.Status)
			if pmt.Status == domain.PaymentStatusSucceeded {
				now := time.Now()
				tx.ProcessedAt = &now
			}
			if err := s.repo.UpdateTransaction(ctx, tx); err != nil {
				s.log.Warn("failed to update transaction", "transaction_id", tx.ID, "error", err)
			}
		}
	}
}
