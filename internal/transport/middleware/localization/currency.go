package localization

import (
	"context"
	"net/http"
	"strings"

	"hauslet/internal/platform/xchange"
)

type currencyKey struct{}

const preferredCurrencyHeader = "X-Currency"

// WithPreferredCurrency extracts the preferred currency from headers or query params and stores it in context.
func WithPreferredCurrency(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		preferred := strings.TrimSpace(r.Header.Get(preferredCurrencyHeader))
		if preferred == "" {
			preferred = strings.TrimSpace(r.URL.Query().Get("currency"))
		}

		if preferred != "" {
			normalized := xchange.NormalizeCurrency(preferred)
			ctx := context.WithValue(r.Context(), currencyKey{}, normalized)
			r = r.WithContext(ctx)
		}

		next.ServeHTTP(w, r)
	})
}

// PreferredCurrency returns the normalized currency stored on the context, if any.
func PreferredCurrency(ctx context.Context) (string, bool) {
	val, ok := ctx.Value(currencyKey{}).(string)
	return val, ok && val != ""
}
