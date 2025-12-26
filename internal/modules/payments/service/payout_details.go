package service

import (
	"context"
	"fmt"
	"hauslet/internal/modules/payments/domain"
	"hauslet/internal/modules/payments/repository/schema"
	"time"

	"github.com/google/uuid"
)

// AddPayoutDetail adds payout details for a user or business
func (s *PaymentServiceImpl) AddPayoutDetail(ctx context.Context, input domain.CreatePayoutDetailInput) (*domain.PayoutDetail, error) {
	s.log.Logf("INFO adding payout detail: user=%v, business=%v", input.UserID, input.BusinessID)

	// Validate input
	if err := input.Validate(); err != nil {
		s.log.Logf("ERROR payout detail validation failed: %v", err)
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Verify bank account with provider
	s.log.Logf("INFO verifying bank account: bank=%s, account=%s", input.BankCode, input.AccountNumber)
	accountName, err := s.paymentClient.ValidateAccount(
		ctx,
		input.Currency,
		input.BankCode,
		input.AccountNumber,
	)
	if err != nil {
		s.log.Logf("ERROR bank account validation failed: %v", err)
		return nil, fmt.Errorf("bank account validation failed: %w", err)
	}

	// Verify account name matches
	if accountName != input.AccountName {
		s.log.Logf("WARN account name mismatch: expected=%s, got=%s", input.AccountName, accountName)
		// Allow it but log warning
	}

	// Create recipient with provider
	s.log.Logf("INFO creating recipient with provider")
	recipientCode, err := s.paymentClient.CreateRecipient(
		ctx,
		input.Currency,
		input.BankCode,
		input.AccountNumber,
		input.AccountName,
	)
	if err != nil {
		s.log.Logf("ERROR failed to create recipient: %v", err)
		return nil, fmt.Errorf("failed to create recipient: %w", err)
	}

	// Create payout detail
	now := time.Now()
	pd := &domain.PayoutDetail{
		ID:            uuid.New(),
		UserID:        input.UserID,
		BusinessID:    input.BusinessID,
		BankCode:      input.BankCode,
		BankName:      "", // TODO: Get bank name from bank code
		AccountNumber: input.AccountNumber,
		AccountName:   accountName,
		Currency:      input.Currency,
		Market:        input.Market,
		Provider:      "paystack",
		RecipientCode: recipientCode,
		IsVerified:    true,
		VerifiedAt:    &now,
		IsDefault:     input.SetAsDefault,
		IsActive:      true,
		Metadata:      make(map[string]string),
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	// If setting as default, handle default logic
	if input.SetAsDefault {
		if err := s.repo.SetDefaultPayoutDetail(ctx, pd.ID, input.UserID, input.BusinessID); err != nil {
			s.log.Logf("WARN failed to set default payout detail: %v", err)
			// Continue anyway
		}
	}

	// Save to database
	if err := s.repo.CreatePayoutDetail(ctx, domain.MapPayoutDetailToSchema(pd)); err != nil {
		s.log.Logf("ERROR failed to save payout detail: %v", err)
		return nil, fmt.Errorf("failed to save payout detail: %w", err)
	}

	s.log.Logf("INFO payout detail saved successfully: id=%s", pd.ID)

	return pd, nil
}

// GetPayoutDetail retrieves a payout detail
func (s *PaymentServiceImpl) GetPayoutDetail(ctx context.Context, id uuid.UUID) (*domain.PayoutDetail, error) {
	s.log.Logf("INFO fetching payout detail: id=%s", id)

	schemaPD, err := s.repo.GetPayoutDetailByID(ctx, id)
	if err != nil {
		s.log.Logf("ERROR failed to get payout detail %s: %v", id, err)
		return nil, fmt.Errorf("failed to get payout detail: %w", err)
	}

	if schemaPD == nil {
		return nil, domain.ErrPayoutDetailNotFound
	}

	return domain.MapPayoutDetailFromSchema(schemaPD), nil
}

// ListPayoutDetails lists payout details for user or business
func (s *PaymentServiceImpl) ListPayoutDetails(ctx context.Context, userID *uuid.UUID, businessID *uuid.UUID) ([]domain.PayoutDetail, error) {
	s.log.Logf("INFO listing payout details: user=%v, business=%v", userID, businessID)

	var schemaDetails []*schema.PayoutDetail
	var err error

	if userID != nil {
		schemaDetails, err = s.repo.ListPayoutDetailsByUserID(ctx, *userID)
	} else if businessID != nil {
		schemaDetails, err = s.repo.ListPayoutDetailsByBusinessID(ctx, *businessID)
	} else {
		return nil, domain.ErrMissingRequiredField
	}

	if err != nil {
		s.log.Logf("ERROR failed to list payout details: %v", err)
		return nil, fmt.Errorf("failed to list payout details: %w", err)
	}

	return domain.MapPayoutDetailsFromSchema(schemaDetails), nil
}

// SetDefaultPayoutDetail sets a payout detail as default
func (s *PaymentServiceImpl) SetDefaultPayoutDetail(ctx context.Context, detailID uuid.UUID, userID *uuid.UUID, businessID *uuid.UUID) error {
	s.log.Logf("INFO setting default payout detail: id=%s", detailID)

	// Verify detail exists and is active
	pd, err := s.GetPayoutDetail(ctx, detailID)
	if err != nil {
		return err
	}

	if !pd.IsActive {
		return domain.ErrInvalidBankDetails
	}

	// Set as default
	if err := s.repo.SetDefaultPayoutDetail(ctx, detailID, userID, businessID); err != nil {
		s.log.Logf("ERROR failed to set default payout detail: %v", err)
		return fmt.Errorf("failed to set default: %w", err)
	}

	s.log.Logf("INFO default payout detail set successfully: id=%s", detailID)

	return nil
}

// RemovePayoutDetail removes (soft deletes) a payout detail
func (s *PaymentServiceImpl) RemovePayoutDetail(ctx context.Context, detailID uuid.UUID) error {
	s.log.Logf("INFO removing payout detail: id=%s", detailID)

	// Delete
	if err := s.repo.DeletePayoutDetail(ctx, detailID); err != nil {
		s.log.Logf("ERROR failed to delete payout detail: %v", err)
		return fmt.Errorf("failed to remove payout detail: %w", err)
	}

	s.log.Logf("INFO payout detail removed successfully: id=%s", detailID)

	return nil
}

// VerifyBankAccount verifies a bank account without saving it
func (s *PaymentServiceImpl) VerifyBankAccount(ctx context.Context, market domain.Market, bankCode, accountNumber string) (string, error) {
	s.log.Logf("INFO verifying bank account: market=%s, bank=%s", market, bankCode)

	currency := domain.GetDefaultCurrency(market)

	accountName, err := s.paymentClient.ValidateAccount(ctx, currency, bankCode, accountNumber)
	if err != nil {
		s.log.Logf("ERROR bank account verification failed: %v", err)
		return "", fmt.Errorf("verification failed: %w", err)
	}

	s.log.Logf("INFO bank account verified: name=%s", accountName)

	return accountName, nil
}
