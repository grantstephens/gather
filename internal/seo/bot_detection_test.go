package seo

import "testing"

func TestIsBot(t *testing.T) {
	tests := []struct {
		name      string
		userAgent string
		want      bool
	}{
		{
			name:      "Googlebot",
			userAgent: "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)",
			want:      true,
		},
		{
			name:      "Facebook crawler",
			userAgent: "facebookexternalhit/1.1 (+http://www.facebook.com/externalhit_uatext.php)",
			want:      true,
		},
		{
			name:      "Twitterbot",
			userAgent: "Twitterbot/1.0",
			want:      true,
		},
		{
			name:      "Regular browser",
			userAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
			want:      false,
		},
		{
			name:      "Chrome browser",
			userAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			want:      false,
		},
		{
			name:      "Slackbot",
			userAgent: "Slackbot-LinkExpanding 1.0 (+https://api.slack.com/robots)",
			want:      true,
		},
		{
			name:      "Discordbot",
			userAgent: "Mozilla/5.0 (compatible; Discordbot/2.0; +https://discordapp.com)",
			want:      true,
		},
		{
			name:      "Empty user agent",
			userAgent: "",
			want:      false,
		},
		{
			name:      "Applebot",
			userAgent: "Applebot/0.1 (+http://www.apple.com/go/applebot)",
			want:      true,
		},
		{
			name:      "DuckDuckBot",
			userAgent: "DuckDuckBot/1.0; (+http://duckduckgo.com/duckduckbot.html)",
			want:      true,
		},
		{
			name:      "YandexBot",
			userAgent: "Mozilla/5.0 (compatible; YandexBot/3.0; +http://yandex.com/bots)",
			want:      true,
		},
		{
			name:      "Internet Archive",
			userAgent: "Mozilla/5.0 (compatible; ia_archiver/1.0)",
			want:      true,
		},
		{
			name:      "AhrefsBot",
			userAgent: "Mozilla/5.0 (compatible; AhrefsBot/7.0; +http://ahrefs.com/robot/)",
			want:      true,
		},
		{
			name:      "SemrushBot",
			userAgent: "Mozilla/5.0 (compatible; SemrushBot/7; +http://www.semrush.com/bot.html)",
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsBot(tt.userAgent); got != tt.want {
				t.Errorf("IsBot() = %v, want %v", got, tt.want)
			}
		})
	}
}
