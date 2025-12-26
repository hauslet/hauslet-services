package service

import (
	"context"
	"fmt"
	"hauslet/internal/modules/payments/domain"
	"hauslet/internal/platform/payment"
	"time"

	"github.com/google/uuid"
)

// AuthorizePaymentMethod initiates a card authorization flow
// Returns authorization URL for user to complete and the reference to track it
func (s *PaymentServiceImpl) AuthorizePaymentMethod(ctx context.Context, userID uuid.UUID, email string, market domain.Market) (string, string, error) {
	s.log.Logf("INFO authorizing payment method for user=%s, market=%s", userID, market)

	// Get default currency for the market
	currency := domain.GetDefaultCurrency(market)

	// Prepare authorization request
	req := payment.AuthorizationRequest{
		Email:    email,
		Currency: string(currency),
		Metadata: map[string]string{
			"user_id": userID.String(),
			"purpose": "payment_method_authorization",
		},
	}

	// Call platform layer to initiate authorization
	resp, err := s.paymentClient.AuthorizePayment(ctx, currency, req)
	if err != nil {
		s.log.Logf("ERROR authorization failed: %v", err)
		return "", "", fmt.Errorf("failed to authorize payment method: %w", err)
	}

	// Extract authorization URL from raw response
	authURL := ""
	if rawMap, ok := resp.Raw.(map[string]any); ok {
		if url, ok := rawMap["authorization_url"].(string); ok {
			authURL = url
		}
		if ref, ok := rawMap["reference"].(string); ok {
			s.log.Logf("INFO authorization initiated: url=%s, reference=%s", authURL, ref)
			return authURL, ref, nil
		}
	}

	s.log.Logf("ERROR failed to extract authorization details from response")
	return "", "", fmt.Errorf("invalid authorization response format")
}

// SavePaymentMethod saves a payment method for a user
func (s *PaymentServiceImpl) SavePaymentMethod(ctx context.Context, input domain.CreatePaymentMethodInput) (*domain.PaymentMethod, error) {
	s.log.Logf("INFO saving payment method for user=%s", input.UserID)

	// Validate input
	if err := input.Validate(); err != nil {
		s.log.Logf("ERROR payment method validation failed: %v", err)
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Check if authorization code already exists
	existing, err := s.repo.GetPaymentMethodByAuthCode(ctx, input.AuthorizationCode)
	if err != nil {
		s.log.Logf("ERROR failed to check existing payment method: %v", err)
		return nil, fmt.Errorf("failed to check existing payment method: %w", err)
	}
	if existing != nil {
		s.log.Logf("WARN payment method already exists: auth_code=%s", input.AuthorizationCode)
		return domain.MapPaymentMethodFromSchema(existing), nil
	}

	// Create payment method
	pm := &domain.PaymentMethod{
		ID:                uuid.New(),
		UserID:            input.UserID,
		Type:              domain.PaymentMethodCard,
		Provider:          input.Provider,
		AuthorizationCode: input.AuthorizationCode,
		Currency:          input.Currency,
		Last4Digits:       input.Last4Digits,
		CardType:          input.CardType,
		Brand:             input.Brand,
		ExpiryMonth:       input.ExpiryMonth,
		ExpiryYear:        input.ExpiryYear,
		BankName:          input.BankName,
		CustomerCode:      input.CustomerCode,
		IsDefault:         input.SetAsDefault,
		IsActive:          true,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	// If setting as default, unset other defaults first
	if input.SetAsDefault {
		if err := s.repo.SetDefaultPaymentMethod(ctx, input.UserID, pm.ID); err != nil {
			s.log.Logf("WARN failed to set default payment method: %v", err)
			// Continue anyway
		}
	}

	// Save to database
	if err := s.repo.CreatePaymentMethod(ctx, domain.MapPaymentMethodToSchema(pm)); err != nil {
		s.log.Logf("ERROR failed to save payment method: %v", err)
		return nil, fmt.Errorf("failed to save payment method: %w", err)
	}

	s.log.Logf("INFO payment method saved successfully: id=%s, user=%s", pm.ID, input.UserID)

	return pm, nil
}

// GetPaymentMethod retrieves a payment method
func (s *PaymentServiceImpl) GetPaymentMethod(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*domain.PaymentMethod, error) {
	s.log.Logf("INFO fetching payment method: id=%s", id)

	schemaPM, err := s.repo.GetPaymentMethodByID(ctx, id)
	if err != nil {
		s.log.Logf("ERROR failed to get payment method %s: %v", id, err)
		return nil, fmt.Errorf("failed to get payment method: %w", err)
	}

	if schemaPM == nil {
		return nil, domain.ErrPaymentMethodNotFound
	}

	// Check ownership
	if schemaPM.UserID != userID {
		return nil, domain.ErrUnauthorized
	}

	return domain.MapPaymentMethodFromSchema(schemaPM), nil
}

// GetDefaultPaymentMethod returns the default payment method for a user.
func (s *PaymentServiceImpl) GetDefaultPaymentMethod(ctx context.Context, userID uuid.UUID) (*domain.PaymentMethod, error) {
	s.log.Logf("INFO fetching default payment method for user=%s", userID)

	schemaPM, err := s.repo.GetDefaultPaymentMethod(ctx, userID)
	if err != nil {
		s.log.Logf("ERROR failed to get default payment method for user %s: %v", userID, err)
		return nil, fmt.Errorf("failed to get default payment method: %w", err)
	}
	if schemaPM == nil {
		return nil, nil
	}

	return domain.MapPaymentMethodFromSchema(schemaPM), nil
}

// ListPaymentMethods lists all payment methods for a user
func (s *PaymentServiceImpl) ListPaymentMethods(ctx context.Context, userID uuid.UUID) ([]domain.PaymentMethod, error) {
	s.log.Logf("INFO listing payment methods for user=%s", userID)

	schemaMethods, err := s.repo.ListPaymentMethodsByUserID(ctx, userID)
	if err != nil {
		s.log.Logf("ERROR failed to list payment methods for user %s: %v", userID, err)
		return nil, fmt.Errorf("failed to list payment methods: %w", err)
	}

	return domain.MapPaymentMethodsFromSchema(schemaMethods), nil
}

// SetDefaultPaymentMethod sets a payment method as default
func (s *PaymentServiceImpl) SetDefaultPaymentMethod(ctx context.Context, methodID, userID uuid.UUID) error {
	s.log.Logf("INFO setting default payment method: id=%s, user=%s", methodID, userID)

	// Verify ownership
	pm, err := s.GetPaymentMethod(ctx, methodID, userID)
	if err != nil {
		return err
	}

	if !pm.IsActive {
		return domain.ErrInvalidPaymentMethod
	}

	// Set as default
	if err := s.repo.SetDefaultPaymentMethod(ctx, userID, methodID); err != nil {
		s.log.Logf("ERROR failed to set default payment method: %v", err)
		return fmt.Errorf("failed to set default: %w", err)
	}

	s.log.Logf("INFO default payment method set successfully: id=%s", methodID)

	return nil
}

// RemovePaymentMethod removes (deactivates) a payment method
func (s *PaymentServiceImpl) RemovePaymentMethod(ctx context.Context, methodID, userID uuid.UUID) error {
	s.log.Logf("INFO removing payment method: id=%s, user=%s", methodID, userID)

	// Verify ownership
	_, err := s.GetPaymentMethod(ctx, methodID, userID)
	if err != nil {
		return err
	}

	// Deactivate
	if err := s.repo.DeactivatePaymentMethod(ctx, methodID); err != nil {
		s.log.Logf("ERROR failed to deactivate payment method: %v", err)
		return fmt.Errorf("failed to remove payment method: %w", err)
	}

	s.log.Logf("INFO payment method removed successfully: id=%s", methodID)

	return nil
}
