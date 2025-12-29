package domain

import (
	"hauslet/internal/platform/payment"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResourceType_IsValid(t *testing.T) {
	tests := []struct {
		name         string
		resourceType ResourceType
		expected     bool
	}{
		{"general", ResourceTypeGeneral, true},
		{"booking", ResourceTypeBooking, true},
		{"id_verification", ResourceTypeIDVerification, true},
		{"subscription", ResourceTypeSubscription, true},
		{"rental_draft", ResourceTypeRentalDraft, true},
		{"invalid", ResourceType("invalid"), false},
		{"empty", ResourceType(""), false},
		{"random", ResourceType("random_type"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.resourceType.IsValid()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetMarketCurrencies(t *testing.T) {
	tests := []struct {
		name             string
		market           Market
		expectedCurrency payment.Currency
		shouldContain    []payment.Currency
	}{
		{
			name:             "Ghana",
			market:           MarketGhana,
			expectedCurrency: payment.GHS,
			shouldContain:    []payment.Currency{payment.GHS},
		},
		{
			name:   "Kenya",
			market: MarketKenya,
			shouldContain: []payment.Currency{"KES", payment.USD},
		},
		{
			name:   "Nigeria",
			market: MarketNigeria,
			shouldContain: []payment.Currency{payment.NGN, payment.USD},
		},
		{
			name:   "South Africa",
			market: MarketSouthAfrica,
			shouldContain: []payment.Currency{"ZAR", payment.USD},
		},
		{
			name:   "Other",
			market: MarketOther,
			shouldContain: []payment.Currency{payment.USD},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			currencies := GetMarketCurrencies(tt.market)
			assert.NotEmpty(t, currencies)

			for _, expectedCurr := range tt.shouldContain {
				assert.Contains(t, currencies, expectedCurr)
			}
		})
	}
}

func TestGetDefaultCurrency(t *testing.T) {
	tests := []struct {
		name     string
		market   Market
		expected payment.Currency
	}{
		{"Ghana", MarketGhana, payment.GHS},
		{"Kenya", MarketKenya, "KES"},
		{"Nigeria", MarketNigeria, payment.NGN},
		{"South Africa", MarketSouthAfrica, "ZAR"},
		{"Other", MarketOther, payment.USD},
		{"Unknown market", Market("UNKNOWN"), payment.USD},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetDefaultCurrency(tt.market)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsCurrencySupported(t *testing.T) {
	tests := []struct {
		name     string
		market   Market
		currency payment.Currency
		expected bool
	}{
		// Ghana
		{"GHS in Ghana", MarketGhana, payment.GHS, true},
		{"USD in Ghana", MarketGhana, payment.USD, false},
		{"NGN in Ghana", MarketGhana, payment.NGN, false},

		// Kenya
		{"KES in Kenya", MarketKenya, "KES", true},
		{"USD in Kenya", MarketKenya, payment.USD, true},
		{"NGN in Kenya", MarketKenya, payment.NGN, false},

		// Nigeria
		{"NGN in Nigeria", MarketNigeria, payment.NGN, true},
		{"USD in Nigeria", MarketNigeria, payment.USD, true},
		{"GHS in Nigeria", MarketNigeria, payment.GHS, false},

		// South Africa
		{"ZAR in South Africa", MarketSouthAfrica, "ZAR", true},
		{"USD in South Africa", MarketSouthAfrica, payment.USD, true},
		{"NGN in South Africa", MarketSouthAfrica, payment.NGN, false},

		// Other
		{"USD in Other", MarketOther, payment.USD, true},
		{"NGN in Other", MarketOther, payment.NGN, false},

		// Unknown market
		{"USD in unknown market", Market("UNKNOWN"), payment.USD, true},
		{"NGN in unknown market", Market("UNKNOWN"), payment.NGN, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsCurrencySupported(tt.market, tt.currency)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMarketCurrencyCombinations(t *testing.T) {
	// Test all defined markets
	markets := []Market{
		MarketGhana,
		MarketKenya,
		MarketNigeria,
		MarketSouthAfrica,
		MarketOther,
	}

	for _, market := range markets {
		t.Run(string(market), func(t *testing.T) {
			// Get currencies for market
			currencies := GetMarketCurrencies(market)
			assert.NotEmpty(t, currencies, "market should have at least one currency")

			// Get default currency
			defaultCurr := GetDefaultCurrency(market)
			assert.NotEmpty(t, defaultCurr, "market should have a default currency")

			// Default currency should be the first one in the list
			assert.Equal(t, currencies[0], defaultCurr)

			// All returned currencies should be supported for that market
			for _, curr := range currencies {
				assert.True(t, IsCurrencySupported(market, curr),
					"currency %s should be supported in market %s", curr, market)
			}
		})
	}
}

func TestPaymentStatus_AllStatuses(t *testing.T) {
	statuses := []PaymentStatus{
		PaymentStatusPending,
		PaymentStatusSucceeded,
		PaymentStatusFailed,
		PaymentStatusCancelled,
		PaymentStatusRefunded,
	}

	expectedValues := []string{
		"pending",
		"succeeded",
		"failed",
		"cancelled",
		"refunded",
	}

	for i, status := range statuses {
		assert.Equal(t, expectedValues[i], string(status))
	}
}

func TestPaymentMethodType_AllTypes(t *testing.T) {
	types := []PaymentMethodType{
		PaymentMethodCard,
		PaymentMethodBank,
	}

	expectedValues := []string{
		"card",
		"bank",
	}

	for i, pmType := range types {
		assert.Equal(t, expectedValues[i], string(pmType))
	}
}

func TestTransactionType_AllTypes(t *testing.T) {
	types := []TransactionType{
		TransactionTypePayment,
		TransactionTypeRefund,
		TransactionTypePayout,
	}

	expectedValues := []string{
		"payment",
		"refund",
		"payout",
	}

	for i, txType := range types {
		assert.Equal(t, expectedValues[i], string(txType))
	}
}

func TestTransactionStatus_AllStatuses(t *testing.T) {
	statuses := []TransactionStatus{
		TransactionStatusPending,
		TransactionStatusSucceeded,
		TransactionStatusFailed,
	}

	expectedValues := []string{
		"pending",
		"succeeded",
		"failed",
	}

	for i, status := range statuses {
		assert.Equal(t, expectedValues[i], string(status))
	}
}

func TestMarket_AllMarkets(t *testing.T) {
	markets := []Market{
		MarketGhana,
		MarketKenya,
		MarketNigeria,
		MarketSouthAfrica,
		MarketOther,
	}

	expectedValues := []string{
		"GH",
		"KE",
		"NG",
		"ZA",
		"OTHER",
	}

	for i, market := range markets {
		assert.Equal(t, expectedValues[i], string(market))
	}
}

func TestResourceType_AllTypes(t *testing.T) {
	types := []ResourceType{
		ResourceTypeGeneral,
		ResourceTypeBooking,
		ResourceTypeIDVerification,
		ResourceTypeSubscription,
		ResourceTypeRentalDraft,
	}

	expectedValues := []string{
		"general",
		"booking",
		"id_verification",
		"subscription",
		"rental_draft",
	}

	for i, resType := range types {
		assert.Equal(t, expectedValues[i], string(resType))
	}

	// All defined types should be valid
	for _, resType := range types {
		assert.True(t, resType.IsValid(), "resource type %s should be valid", resType)
	}
}

func TestMultiCurrencySupport(t *testing.T) {
	t.Run("Nigeria supports NGN and USD", func(t *testing.T) {
		assert.True(t, IsCurrencySupported(MarketNigeria, payment.NGN))
		assert.True(t, IsCurrencySupported(MarketNigeria, payment.USD))
		assert.False(t, IsCurrencySupported(MarketNigeria, payment.GHS))
	})

	t.Run("Kenya supports KES and USD", func(t *testing.T) {
		assert.True(t, IsCurrencySupported(MarketKenya, "KES"))
		assert.True(t, IsCurrencySupported(MarketKenya, payment.USD))
		assert.False(t, IsCurrencySupported(MarketKenya, payment.NGN))
	})

	t.Run("South Africa supports ZAR and USD", func(t *testing.T) {
		assert.True(t, IsCurrencySupported(MarketSouthAfrica, "ZAR"))
		assert.True(t, IsCurrencySupported(MarketSouthAfrica, payment.USD))
		assert.False(t, IsCurrencySupported(MarketSouthAfrica, payment.NGN))
	})

	t.Run("Ghana supports only GHS", func(t *testing.T) {
		assert.True(t, IsCurrencySupported(MarketGhana, payment.GHS))
		assert.False(t, IsCurrencySupported(MarketGhana, payment.USD))
		assert.False(t, IsCurrencySupported(MarketGhana, payment.NGN))
	})

	t.Run("Other market supports only USD", func(t *testing.T) {
		assert.True(t, IsCurrencySupported(MarketOther, payment.USD))
		assert.False(t, IsCurrencySupported(MarketOther, payment.NGN))
		assert.False(t, IsCurrencySupported(MarketOther, payment.GHS))
	})
}
