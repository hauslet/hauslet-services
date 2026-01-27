package oauth

import "strings"

// maskEmail masks email addresses for non-debug logs to reduce PII exposure.
// Example: "user@example.com" -> "u***@example.com"
func maskEmail(email string) string {
	if email == "" {
		return "***"
	}
	if idx := strings.Index(email, "@"); idx > 0 {
		return email[:1] + "***@" + email[idx+1:]
	}
	return "***"
}
