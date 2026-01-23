package service

import "strings"

// BotDetector identifies bot traffic based on user agent
type BotDetector interface {
	IsBot(userAgent string) bool
}

// SimpleBotDetector implements basic bot detection
type SimpleBotDetector struct {
	botPatterns []string
}

// NewBotDetector creates a new bot detector
func NewBotDetector() BotDetector {
	return &SimpleBotDetector{
		botPatterns: []string{
			// Generic terms
			"bot", "crawler", "spider", "crawling", "scraper", "headless",

			// Major Search Engines
			"googlebot", "bingbot", "slurp", "duckduckbot", "baiduspider", "yandexbot",
			"sogou", "exabot", "ia_archiver",

			// Social Media
			"facebookexternalhit", "twitterbot", "pinterest", "linkedinbot", "slackbot",
			"discordbot", "whatsapp", "telegrambot", "vkshare", "tumblr", "redditbot",

			// SEO and Tools
			"ahrefsbot", "mj12bot", "semrushbot", "dotbot", "rogerbot", "seokicks",
			"screaming frog", "w3c_validator", "lighthouse", "pagespeed",

			// Content Fetchers
			"embedly", "quora link preview", "showyoubot", "outbrain", "flipboard",
			"nuzzel", "qwantify", "bitlybot", "skypeuripreview",

			// Technical / Libraries
			"python", "java", "wget", "curl", "libwww", "urllib", "okhttp",
			"phantom", "selenium", "webdriver", "chrome-lighthouse",

			// Cloud / Monitoring
			"aws-security-scanner", "datadog", "newrelic", "pingdom",
		},
	}
}

// IsBot checks if the user agent matches known bot patterns
func (d *SimpleBotDetector) IsBot(userAgent string) bool {
	if userAgent == "" {
		return true // Empty user agent is suspicious
	}

	// Fast path for very short user agents
	if len(userAgent) < 5 {
		return true
	}

	lowerUA := strings.ToLower(userAgent)

	for _, pattern := range d.botPatterns {
		// Manual strings.Contains is often faster than regex for simple substring matches
		if strings.Contains(lowerUA, pattern) {
			return true
		}
	}

	return false
}
