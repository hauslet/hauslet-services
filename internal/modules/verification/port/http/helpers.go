package http

import "net/http"

// extractSignature extracts the webhook signature based on provider conventions
func (h *WebhookHandler) extractSignature(r *http.Request, providerName string) string {
	switch providerName {
	case "dojah":
		// Dojah uses x-dojah-signature or x-dojah-signature-v2 header
		sig := r.Header.Get("x-dojah-signature")
		if sig == "" {
			sig = r.Header.Get("x-dojah-signature-v2")
		}
		return sig
	case "veriff":
		// Veriff uses x-hmac-signature header
		return r.Header.Get("x-hmac-signature")
	default:
		// Generic signature header
		return r.Header.Get("x-webhook-signature")
	}
}
