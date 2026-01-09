package http

import (
	"context"
	"encoding/json"
	"fmt"
	paymentdomain "hauslet/internal/modules/payments/domain"
	"strings"
)

// extractAndSaveAuthorization extracts authorization code from Paystack webhook and saves it
func (h *WebhookHandler) extractAndSaveAuthorization(ctx context.Context, rawData json.RawMessage, pmt *paymentdomain.Payment) error {
	var webhook struct {
		Event string `json:"event"`
		Data  struct {
			Authorization struct {
				AuthorizationCode string `json:"authorization_code"`
				CardType          string `json:"card_type"`
				Last4             string `json:"last4"`
				ExpMonth          string `json:"exp_month"`
				ExpYear           string `json:"exp_year"`
				Bank              string `json:"bank"`
				Brand             string `json:"brand"`
				Reusable          bool   `json:"reusable"`
			} `json:"authorization"`
			Customer struct {
				ID           int    `json:"id"`
				CustomerCode string `json:"customer_code"`
				Email        string `json:"email"`
			} `json:"customer"`
		} `json:"data"`
	}

	if err := json.Unmarshal(rawData, &webhook); err != nil {
		return fmt.Errorf("failed to parse webhook data: %w", err)
	}

	auth := webhook.Data.Authorization
	customer := webhook.Data.Customer

	if !auth.Reusable || auth.AuthorizationCode == "" {
		h.log.Info(" authorization not reusable or empty, skipping save")
		return nil
	}

	h.log.Info(" extracting reusable authorization", "code", auth.AuthorizationCode, "last4", auth.Last4)

	expMonth, expYear, err := parseExpiry(auth.ExpMonth, auth.ExpYear)
	if err != nil {
		h.log.Warn("failed to parse expiry dates", "error", err)
	}

	customerCode := customer.CustomerCode
	input := paymentdomain.CreatePaymentMethodInput{
		UserID:            pmt.PayerID,
		AuthorizationCode: auth.AuthorizationCode,
		Provider:          "paystack",
		Currency:          pmt.Currency,
		Last4Digits:       &auth.Last4,
		CardType:          &auth.CardType,
		Brand:             &auth.Brand,
		ExpiryMonth:       expMonth,
		ExpiryYear:        expYear,
		BankName:          &auth.Bank,
		CustomerCode:      &customerCode,
		SetAsDefault:      false,
	}

	savedPM, err := h.paymentService.SavePaymentMethod(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to save payment method: %w", err)
	}

	cardDisplay := "unknown card"
	if savedPM.Last4Digits != nil && savedPM.Brand != nil {
		cardDisplay = fmt.Sprintf("%s ending in %s", *savedPM.Brand, *savedPM.Last4Digits)
	}
	h.log.Info(" payment method saved successfully", "id", savedPM.ID, "user", savedPM.UserID, "card", cardDisplay)

	return nil
}

// parseExpiry converts string expiry values to integers
func parseExpiry(monthStr, yearStr string) (*int, *int, error) {
	if monthStr == "" || yearStr == "" {
		return nil, nil, fmt.Errorf("empty expiry values")
	}

	var month, year int
	if _, err := fmt.Sscanf(monthStr, "%d", &month); err != nil {
		return nil, nil, fmt.Errorf("invalid month: %w", err)
	}
	if _, err := fmt.Sscanf(yearStr, "%d", &year); err != nil {
		return nil, nil, fmt.Errorf("invalid year: %w", err)
	}

	if month < 1 || month > 12 {
		return nil, nil, fmt.Errorf("month must be between 1 and 12")
	}

	return &month, &year, nil
}

func isPaymentEvent(eventType string) bool {
	return strings.HasPrefix(eventType, "charge.") || strings.HasPrefix(eventType, "refund.")
}
