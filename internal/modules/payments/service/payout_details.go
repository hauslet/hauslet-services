package service

import (
	"context"
	"fmt"
	"hauslet/internal/modules/payments/domain"
	"hauslet/internal/modules/payments/repository/schema"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
)

// AddPayoutDetail adds payout details for a user or business
func (s *PaymentServiceImpl) AddPayoutDetail(ctx context.Context, input domain.CreatePayoutDetailInput) (*domain.PayoutDetail, error) {
	s.log.Info(" adding payout detail", "user_id", input.UserID, "business_id", input.BusinessID)

	// Validate input
	if err := input.Validate(); err != nil {
		s.log.Error("payout detail validation failed", "error", err)
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Verify bank account with provider
	s.log.Info(" verifying bank account", "bank_code", input.BankCode, "account_number", input.AccountNumber)
	accountName, err := s.paymentClient.ValidateAccount(
		ctx,
		input.Currency,
		input.BankCode,
		input.AccountNumber,
	)
	if err != nil {
		s.log.Error("bank account validation failed", "error", err)
		return nil, fmt.Errorf("bank account validation failed: %w", err)
	}

	// Verify account name matches
	if !nameMatches(input.AccountName, accountName) {
		s.log.Warn("account name mismatch", "expected", input.AccountName, "actual", accountName)
		return nil, domain.ErrAccountNameMismatch
	}

	// Create recipient with provider
	s.log.Info(" creating recipient with provider")
	recipientCode, err := s.paymentClient.CreateRecipient(
		ctx,
		input.Currency,
		input.BankCode,
		input.AccountNumber,
		input.AccountName,
	)
	if err != nil {
		s.log.Error("failed to create recipient", "error", err)
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
			s.log.Warn("failed to set default payout detail", "error", err)
			// Continue anyway
		}
	}

	// Save to database
	if err := s.repo.CreatePayoutDetail(ctx, domain.MapPayoutDetailToSchema(pd)); err != nil {
		s.log.Error("failed to save payout detail", "error", err)
		return nil, fmt.Errorf("failed to save payout detail: %w", err)
	}

	s.log.Info(" payout detail saved successfully", "id", pd.ID)

	return pd, nil
}

// GetPayoutDetail retrieves a payout detail
func (s *PaymentServiceImpl) GetPayoutDetail(ctx context.Context, id uuid.UUID) (*domain.PayoutDetail, error) {
	s.log.Info(" fetching payout detail", "id", id)

	schemaPD, err := s.repo.GetPayoutDetailByID(ctx, id)
	if err != nil {
		s.log.Error("failed to get payout detail", "id", id, "error", err)
		return nil, fmt.Errorf("failed to get payout detail: %w", err)
	}

	if schemaPD == nil {
		return nil, domain.ErrPayoutDetailNotFound
	}

	return domain.MapPayoutDetailFromSchema(schemaPD), nil
}

// ListPayoutDetails lists payout details for user or business
func (s *PaymentServiceImpl) ListPayoutDetails(ctx context.Context, userID *uuid.UUID, businessID *uuid.UUID) ([]domain.PayoutDetail, error) {
	s.log.Info(" listing payout details", "user_id", userID, "business_id", businessID)

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
		s.log.Error("failed to list payout details", "error", err)
		return nil, fmt.Errorf("failed to list payout details: %w", err)
	}

	return domain.MapPayoutDetailsFromSchema(schemaDetails), nil
}

// SetDefaultPayoutDetail sets a payout detail as default
func (s *PaymentServiceImpl) SetDefaultPayoutDetail(ctx context.Context, detailID uuid.UUID, userID *uuid.UUID, businessID *uuid.UUID) error {
	s.log.Info(" setting default payout detail", "id", detailID)

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
		s.log.Error("failed to set default payout detail", "error", err)
		return fmt.Errorf("failed to set default: %w", err)
	}

	s.log.Info(" default payout detail set successfully", "id", detailID)
	return nil
}

// RemovePayoutDetail removes (soft deletes) a payout detail
func (s *PaymentServiceImpl) RemovePayoutDetail(ctx context.Context, detailID uuid.UUID) error {
	s.log.Info(" removing payout detail", "id", detailID)

	// Delete
	if err := s.repo.DeletePayoutDetail(ctx, detailID); err != nil {
		s.log.Error("failed to delete payout detail", "error", err)
		return fmt.Errorf("failed to remove payout detail: %w", err)
	}

	s.log.Info(" payout detail removed successfully", "id", detailID)

	return nil
}

// VerifyBankAccount verifies a bank account without saving it
func (s *PaymentServiceImpl) VerifyBankAccount(ctx context.Context, market domain.Market, bankCode, accountNumber string) (string, error) {
	s.log.Info(" verifying bank account", "market", market, "bank", bankCode)

	currency := domain.GetDefaultCurrency(market)

	accountName, err := s.paymentClient.ValidateAccount(ctx, currency, bankCode, accountNumber)
	if err != nil {
		s.log.Error("bank account verification failed", "error", err)
		return "", fmt.Errorf("verification failed: %w", err)
	}

	s.log.Info(" bank account verified", "name", accountName)

	return accountName, nil
}
func nameMatches(inputName, bankName string) bool {
	inputTokens := normalizeNameTokens(inputName)
	bankTokens := normalizeNameTokens(bankName)

	if len(inputTokens) == 0 || len(bankTokens) == 0 {
		return false
	}

	// 1. Calculate Intersection (allowing for small typos)
	matches := 0
	for _, inToken := range inputTokens {
		for _, bankToken := range bankTokens {
			// Check for exact match OR small typo (Levenshtein)
			if inToken == bankToken || isFuzzyMatch(inToken, bankToken) {
				matches++
				break
			}
		}
	}

	// 2. Calculate Jaccard Similarity
	// Union = (len(input) + len(bank)) - matches
	union := float64(len(inputTokens) + len(bankTokens) - matches)
	if union == 0 {
		return false
	}

	score := float64(matches) / union

	// 3. Threshold
	// 0.6 is a common "good enough" baseline.
	// "John Doe" vs "John A. Doe" -> 2 matches / 3 union = 0.66 (Pass)
	// "John" vs "John Smith"      -> 1 match   / 2 union = 0.50 (Fail)
	return score >= 0.65
}

// Simple helper to allow "Ltd" == "Limited" or small typos
func isFuzzyMatch(s1, s2 string) bool {
	// Handle common abbreviations explicitly if needed
	if isAbbreviation(s1, s2) {
		return true
	}

	// Allow small edit distance (Levenshtein) for typos
	// e.g. "Jon" vs "John"
	if abs(len(s1)-len(s2)) > 2 {
		return false
	}
	return levenshtein(s1, s2) <= 1
}

func isAbbreviation(s1, s2 string) bool {
	// Map of common abbreviations
	abbr := map[string]string{
		"ltd": "limited", "inc": "incorporated", "co": "company",
		"corp": "corporation",
	}
	if val, ok := abbr[s1]; ok && val == s2 {
		return true
	}
	if val, ok := abbr[s2]; ok && val == s1 {
		return true
	}
	return false
}

func normalizeNameTokens(name string) []string {
	name = strings.ToLower(name)
	return strings.FieldsFunc(name, func(r rune) bool {
		return !(unicode.IsLetter(r) || unicode.IsNumber(r))
	})
}

// Basic Levenshtein implementation
func levenshtein(s1, s2 string) int {
	r1, r2 := []rune(s1), []rune(s2)
	n, m := len(r1), len(r2)
	if n == 0 {
		return m
	}
	if m == 0 {
		return n
	}

	// Create matrix
	d := make([][]int, n+1)
	for i := range d {
		d[i] = make([]int, m+1)
	}

	for i := 0; i <= n; i++ {
		d[i][0] = i
	}
	for j := 0; j <= m; j++ {
		d[0][j] = j
	}

	for i := 1; i <= n; i++ {
		for j := 1; j <= m; j++ {
			cost := 1
			if r1[i-1] == r2[j-1] {
				cost = 0
			}

			minVal := d[i-1][j] + 1 // deletion
			if ins := d[i][j-1] + 1; ins < minVal {
				minVal = ins
			} // insertion
			if sub := d[i-1][j-1] + cost; sub < minVal {
				minVal = sub
			} // substitution

			d[i][j] = minVal
		}
	}
	return d[n][m]
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
