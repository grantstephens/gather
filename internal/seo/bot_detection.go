package seo

import "strings"

var botUserAgents = []string{
	"Googlebot",
	"Google-InspectionTool",
	"Google-Read-Aloud",
	"facebookexternalhit",
	"Twitterbot",
	"LinkedInBot",
	"Slackbot",
	"TelegramBot",
	"WhatsApp",
	"Discordbot",
	"bingbot",
	"Applebot",
	"DuckDuckBot",
	"YandexBot",
	"ia_archiver",
	"AhrefsBot",
	"SemrushBot",
}

// IsBot checks if the User-Agent header belongs to a known bot/crawler
func IsBot(userAgent string) bool {
	ua := strings.ToLower(userAgent)
	for _, bot := range botUserAgents {
		if strings.Contains(ua, strings.ToLower(bot)) {
			return true
		}
	}
	return false
}
