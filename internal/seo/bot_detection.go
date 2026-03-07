package seo

import "strings"

var botUserAgents = []string{
	"Googlebot",
	"facebookexternalhit",
	"Twitterbot",
	"LinkedInBot",
	"Slackbot",
	"TelegramBot",
	"WhatsApp",
	"Discordbot",
	"bingbot",
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
