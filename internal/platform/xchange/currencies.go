package xchange

import "strings"

const defaultCurrency = "USD"

var supportedCurrencies = map[string]struct{}{
	"NGN": {},
	"KES": {},
	"GHS": {},
	"ZAR": {},
	"XOF": {},
	"XAF": {},
	"EGP": {},
	"USD": {},
	"GBP": {},
	"EUR": {},
	"CAD": {},
	"AUD": {},
}

// NormalizeCurrency returns a supported code or USD if unsupported/empty.
func NormalizeCurrency(code string) string {
	c := strings.ToUpper(strings.TrimSpace(code))
	if _, ok := supportedCurrencies[c]; ok && c != "" {
		return c
	}
	return defaultCurrency
}
