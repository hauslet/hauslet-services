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
	s.log.Logf("INFO creating payment: amount=%d, currency=%s, payer=%s", input.Amount, input.Currency, input.PayerID)

	// Validate input
	if err := input.Validate(); err != nil {
		s.log.Logf("ERROR payment validation failed: %v", err)
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Auto-sync polymorphic fields with explicit fields for consistency
	if input.BookingID != nil && (input.ResourceType == domain.ResourceTypeGeneral || input.ResourceType == "") {
		s.log.Logf("INFO auto-syncing booking payment: setting ResourceType=booking, ResourceID=%s", input.BookingID)
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
			s.log.Logf("ERROR failed to get payment method %s: %v", input.PaymentMethodID, err)
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
		s.log.Logf("INFO charging saved payment method for payment=%s", reference)
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
		s.log.Logf("INFO initializing new payment for payment=%s", reference)
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
		s.log.Logf("ERROR payment processing failed for payment=%s: %v", reference, err)
		pmt.Status = domain.PaymentStatusFailed

		// Save failed payment
		if saveErr := s.repo.CreatePayment(ctx, domain.MapPaymentToSchema(pmt)); saveErr != nil {
			s.log.Logf("ERROR failed to save failed payment: %v", saveErr)
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
		s.log.Logf("ERROR failed to save payment: %v", err)
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
		s.log.Logf("WARN failed to create transaction record: %v", err)
		// Non-critical, don't fail the payment
	}

	s.log.Logf("INFO payment created successfully: id=%s, status=%s", pmt.ID, pmt.Status)

	// Send notification if payment succeeded immediately
	if pmt.Status == domain.PaymentStatusSucceeded {
		s.notificationSvc.SendPaymentReceipt(pmt)
	}

	return pmt, nil
}

// GetPayment retrieves a payment by ID
func (s *PaymentServiceImpl) GetPayment(ctx context.Context, id uuid.UUID) (*domain.Payment, error) {
	s.log.Logf("INFO fetching payment: id=%s", id)

	schemaPmt, err := s.repo.GetPaymentByID(ctx, id)
	if err != nil {
		s.log.Logf("ERROR failed to get payment %s: %v", id, err)
		return nil, fmt.Errorf("failed to get payment: %w", err)
	}

	if schemaPmt == nil {
		return nil, domain.ErrPaymentNotFound
	}

	return domain.MapPaymentFromSchema(schemaPmt), nil
}

// GetPaymentByReference retrieves a payment by reference
func (s *PaymentServiceImpl) GetPaymentByReference(ctx context.Context, reference string) (*domain.Payment, error) {
	s.log.Logf("INFO fetching payment by reference: ref=%s", reference)

	schemaPmt, err := s.repo.GetPaymentByReference(ctx, reference)
	if err != nil {
		s.log.Logf("ERROR failed to get payment by reference %s: %v", reference, err)
		return nil, fmt.Errorf("failed to get payment: %w", err)
	}

	if schemaPmt == nil {
		return nil, domain.ErrPaymentNotFound
	}

	return domain.MapPaymentFromSchema(schemaPmt), nil
}

// VerifyPayment verifies a payment with the provider and updates status
func (s *PaymentServiceImpl) VerifyPayment(ctx context.Context, reference string) (*domain.Payment, error) {
	s.log.Logf("INFO verifying payment: ref=%s", reference)

	// Get payment from database
	pmt, err := s.GetPaymentByReference(ctx, reference)
	if err != nil {
		return nil, err
	}

	// If already succeeded or failed, return as is
	if pmt.Status == domain.PaymentStatusSucceeded || pmt.Status == domain.PaymentStatusFailed {
		s.log.Logf("INFO payment %s already finalized with status=%s", reference, pmt.Status)
		return pmt, nil
	}

	// Verify with provider
	s.log.Logf("INFO verifying payment with provider: ref=%s", reference)
	providerResp, err := s.paymentClient.Verify(ctx, pmt.Currency, reference)
	if err != nil {
		s.log.Logf("ERROR provider verification failed for payment=%s: %v", reference, err)
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
		s.log.Logf("ERROR failed to update payment status: %v", err)
		return nil, fmt.Errorf("failed to update payment: %w", err)
	}

	// Update transaction if status changed
	if oldStatus != pmt.Status {
		s.updateTransactionStatus(ctx, pmt)
	}

	s.log.Logf("INFO payment verified: id=%s, status=%s", pmt.ID, pmt.Status)

	// Send receipt if payment succeeded
	if pmt.Status == domain.PaymentStatusSucceeded && oldStatus != domain.PaymentStatusSucceeded {
		s.notificationSvc.SendPaymentReceipt(pmt)
	}

	return pmt, nil
}

// ListPaymentsByPayer lists payments for a payer
func (s *PaymentServiceImpl) ListPaymentsByPayer(ctx context.Context, payerID uuid.UUID, limit, offset int) ([]domain.Payment, error) {
	s.log.Logf("INFO listing payments for payer=%s", payerID)

	schemaPayments, err := s.repo.ListPaymentsByPayerID(ctx, payerID, limit, offset)
	if err != nil {
		s.log.Logf("ERROR failed to list payments for payer %s: %v", payerID, err)
		return nil, fmt.Errorf("failed to list payments: %w", err)
	}

	return domain.MapPaymentsFromSchema(schemaPayments), nil
}

// ListPaymentsByBooking lists payments for a booking
func (s *PaymentServiceImpl) ListPaymentsByBooking(ctx context.Context, bookingID uuid.UUID) ([]domain.Payment, error) {
	s.log.Logf("INFO listing payments for booking=%s", bookingID)

	schemaPayments, err := s.repo.ListPaymentsByBookingID(ctx, bookingID)
	if err != nil {
		s.log.Logf("ERROR failed to list payments for booking %s: %v", bookingID, err)
		return nil, fmt.Errorf("failed to list payments: %w", err)
	}

	return domain.MapPaymentsFromSchema(schemaPayments), nil
}

// updateTransactionStatus updates the transaction status when payment status changes
func (s *PaymentServiceImpl) updateTransactionStatus(ctx context.Context, pmt *domain.Payment) {
	transactions, err := s.repo.ListTransactionsByPaymentID(ctx, pmt.ID)
	if err != nil {
		s.log.Logf("WARN failed to get transactions for payment %s: %v", pmt.ID, err)
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
				s.log.Logf("WARN failed to update transaction %s: %v", tx.ID, err)
			}
		}
	}
}
