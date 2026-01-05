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
			"bot",
			"crawler",
			"spider",
			"crawling",
			"scraper",
			"googlebot",
			"bingbot",
			"slurp",
			"duckduckbot",
			"baiduspider",
			"yandexbot",
			"facebookexternalhit",
			"twitterbot",
			"rogerbot",
			"linkedinbot",
			"embedly",
			"quora link preview",
			"showyoubot",
			"outbrain",
			"pinterest",
			"slackbot",
			"vkshare",
			"w3c_validator",
			"redditbot",
			"applebot",
			"whatsapp",
			"flipboard",
			"tumblr",
			"bitlybot",
			"skypeuripreview",
			"nuzzel",
			"discordbot",
			"qwantify",
			"pinterestbot",
			"telegrambot",
			"headlesschrome",
			"phantom",
			"selenium",
			"webdriver",
		},
	}
}

// IsBot checks if the user agent matches known bot patterns
func (d *SimpleBotDetector) IsBot(userAgent string) bool {
	if userAgent == "" {
		return true // Empty user agent is suspicious
	}

	lowerUA := strings.ToLower(userAgent)

	for _, pattern := range d.botPatterns {
		if strings.Contains(lowerUA, pattern) {
			return true
		}
	}

	return false
}
