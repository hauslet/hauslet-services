package service

import (
	"regexp"
	"strings"
)

// SpamDetector performs pattern-based spam detection on lead submissions
type SpamDetector struct {
	spamPatterns []*regexp.Regexp
	spamKeywords []string
}

// NewSpamDetector creates a new instance of SpamDetector
func NewSpamDetector() *SpamDetector {
	return &SpamDetector{
		spamPatterns: []*regexp.Regexp{
			// Common spam words
			regexp.MustCompile(`(?i)(viagra|cialis|casino|lottery|winner|prize)`),
			// Marketing spam
			regexp.MustCompile(`(?i)(click here|buy now|limited offer|act now|call now)`),
			// Suspicious long URLs
			regexp.MustCompile(`https?://[^\s]{50,}`),
			// Multiple consecutive URLs
			regexp.MustCompile(`(https?://[^\s]+\s*){3,}`),
			// Excessive repeated characters
			regexp.MustCompile(`(.)\1{10,}`),
		},
		spamKeywords: []string{
			"bitcoin", "crypto", "investment scheme", "guaranteed profit",
			"make money fast", "work from home", "free money",
			"congratulations", "you've won", "claim your prize",
		},
	}
}

// CalculateSpamScore calculates a spam probability score for a lead submission
// Returns a score between 0.0 (clean) and 1.0 (definitely spam)
func (sd *SpamDetector) CalculateSpamScore(name, email, message string) float64 {
	score := 0.0

	// Check message against spam patterns
	for _, pattern := range sd.spamPatterns {
		if pattern.MatchString(message) {
			score += 0.3
		}
	}

	// Check for spam keywords
	messageLower := strings.ToLower(message)
	for _, keyword := range sd.spamKeywords {
		if strings.Contains(messageLower, keyword) {
			score += 0.2
		}
	}

	// Suspicious email patterns
	emailLower := strings.ToLower(email)

	// Too many dots or plus signs in email
	dotCount := strings.Count(emailLower, ".")
	plusCount := strings.Count(emailLower, "+")
	if dotCount > 3 || plusCount > 1 {
		score += 0.1
	}

	// Suspicious email domains (disposable email services)
	suspiciousDomains := []string{
		"tempmail", "throwaway", "guerrillamail", "10minutemail",
		"mailinator", "trashmail", "fakeinbox",
	}
	for _, domain := range suspiciousDomains {
		if strings.Contains(emailLower, domain) {
			score += 0.3
		}
	}

	// All caps message (but ignore short messages)
	if len(message) > 50 && strings.ToUpper(message) == message {
		score += 0.2
	}

	// Excessive punctuation
	exclamationCount := strings.Count(message, "!")
	questionCount := strings.Count(message, "?")
	if exclamationCount > 5 || questionCount > 5 {
		score += 0.15
	}

	// Very short message (likely not genuine inquiry)
	if len(strings.TrimSpace(message)) < 20 {
		score += 0.1
	}

	// Excessive links in message
	linkCount := strings.Count(strings.ToLower(message), "http")
	if linkCount > 2 {
		score += 0.25
	}

	// Name is just random characters or single letter
	if len(strings.TrimSpace(name)) <= 1 {
		score += 0.15
	}

	// Name and email are very similar (possible bot)
	nameClean := strings.ToLower(strings.ReplaceAll(name, " ", ""))
	emailPrefix := strings.Split(emailLower, "@")[0]
	if len(nameClean) > 0 && nameClean == emailPrefix {
		score += 0.1
	}

	// Cap at 1.0
	if score > 1.0 {
		score = 1.0
	}

	return score
}

// IsSpam returns true if the spam score exceeds the threshold
func (sd *SpamDetector) IsSpam(score float64) bool {
	// Threshold for automatic spam classification
	const spamThreshold = 0.7
	return score >= spamThreshold
}

// GetSpamLevel returns a human-readable spam level
func (sd *SpamDetector) GetSpamLevel(score float64) string {
	if score >= 0.7 {
		return "high"
	} else if score >= 0.5 {
		return "medium"
	} else if score >= 0.3 {
		return "low"
	}
	return "clean"
}
