package seo

import (
	"fmt"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"gather/internal/recurrence"
)

// GenerateHomeShell renders a lightweight HTML fragment mirroring the home page's
// above-the-fold content (header + first event card) for real (non-bot) visitors.
//
// This is NOT hydrated by Preact. It's injected into a sibling `#app-shell` div
// next to the empty `#app` mount point; main.tsx removes it synchronously right
// after the client app mounts, in the same task, before the browser paints again.
// That means it only has to look reasonably close to the real layout for the
// (often throttled-mobile) window before JS takes over — it never needs to match
// exactly, and a mismatch can't cause a hydration bug because Preact never touches
// this DOM. Its purpose is purely to get real content (especially the first event's
// image, our LCP candidate) painted before JS finishes loading.
//
// Known simplification: omits the sidebar (calendar/tags/towns) and the Editor's
// Picks teaser to keep this to a single, well-tested query
// (recurrence.ListExpandedEvents — the same one the client calls) rather than
// reimplementing the picks day-window logic server-side. On desktop, and on
// weekends when picks are shown, this means a small layout shift once JS mounts.
func GenerateHomeShell(app core.App, baseURL string) (string, error) {
	instanceName := "Gather"
	subtitle := ""
	if settings, err := app.FindFirstRecordByFilter("settings", ""); err == nil {
		if name := settings.GetString("instance_name"); name != "" {
			instanceName = name
		}
		subtitle = settings.GetString("subtitle")
	}

	now := time.Now().UTC()
	events, _, err := recurrence.ListExpandedEvents(app, now, now.AddDate(0, 3, 0), "", nil, 1, 1)
	if err != nil {
		return "", err
	}

	var b strings.Builder

	b.WriteString(`<header class="site-header"><nav class="site-nav">`)
	b.WriteString(`<div class="nav-brand"><a href="/" class="nav-wordmark">`)
	b.WriteString(brandNameHTML(instanceName))
	b.WriteString(`</a>`)
	if subtitle != "" {
		b.WriteString(fmt.Sprintf(`<span class="nav-subtitle">%s</span>`, htmlEscape(subtitle)))
	}
	b.WriteString(`</div>`)
	b.WriteString(`<div class="nav-search"><form><svg class="nav-search-icon" width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><circle cx="7" cy="7" r="4.5"></circle><line x1="10.5" y1="10.5" x2="14" y2="14"></line></svg><input type="search" class="nav-search-input" placeholder="Search..." readonly></form></div>`)
	b.WriteString(`<button class="nav-hamburger" aria-label="Toggle navigation" aria-expanded="false"><svg width="18" height="18" viewBox="0 0 18 18" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><line x1="2" y1="5" x2="16" y2="5"></line><line x1="2" y1="9" x2="16" y2="9"></line><line x1="2" y1="13" x2="16" y2="13"></line></svg></button>`)
	b.WriteString(`<div class="nav-links"><a href="/submit" class="nav-link nav-link--cta">Submit Event</a></div>`)
	b.WriteString(`</nav></header>`)

	b.WriteString(`<div class="app"><main><div class="home"><div class="home-main">`)
	b.WriteString(mobileFilterBarHTML)
	b.WriteString(`<div class="events-header"><h2>Upcoming Events</h2></div>`)

	if len(events) > 0 {
		b.WriteString(`<div class="timeline"><section class="timeline-day"><div class="timeline-day-events">`)
		b.WriteString(featuredEventCardHTML(events[0], baseURL))
		b.WriteString(`</div></section></div>`)
	}

	b.WriteString(`</div></div></main>`)
	b.WriteString(`<footer class="app-footer"><div class="footer-inner"><div class="footer-links">`)
	b.WriteString(`<a href="/feed.rss" class="footer-link">RSS</a><a href="/feed.ics" class="footer-link">iCal</a>`)
	b.WriteString(`</div></div></footer>`)
	b.WriteString(`</div>`)

	return b.String(), nil
}

// brandNameHTML mirrors app.tsx's BrandName component: split on the first space
// and drop a styled "." right after the first word.
func brandNameHTML(name string) string {
	const dot = `<span class="brand-dot">.</span>`
	idx := strings.IndexByte(name, ' ')
	if idx == -1 {
		return htmlEscape(name) + dot
	}
	return htmlEscape(name[:idx]) + dot + htmlEscape(name[idx:])
}

const mobileFilterBarHTML = `<div class="mobile-filter-bar">` +
	`<button class="mobile-filter-btn"><svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="4" width="18" height="18" rx="2" ry="2"></rect><line x1="16" y1="2" x2="16" y2="6"></line><line x1="8" y1="2" x2="8" y2="6"></line><line x1="3" y1="10" x2="21" y2="10"></line></svg>Dates</button>` +
	`<button class="mobile-filter-btn"><svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M20.59 13.41l-7.17 7.17a2 2 0 01-2.83 0L2 12V2h10l8.59 8.59a2 2 0 010 2.82z"></path><line x1="7" y1="7" x2="7.01" y2="7"></line></svg>Tags</button>` +
	`<button class="mobile-filter-btn"><svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 2C8.13 2 5 5.13 5 9c0 5.25 7 13 7 13s7-7.75 7-13c0-3.87-3.13-7-7-7z"></path><circle cx="12" cy="9" r="2.5"></circle></svg>Locations</button>` +
	`</div>`

// featuredEventCardHTML mirrors EventCard.tsx's "featured" variant markup exactly
// (same classes/structure) so the shared stylesheet renders it correctly, with the
// image eager-loaded at high priority since this is the page's LCP candidate.
func featuredEventCardHTML(e recurrence.ExpandedEvent, baseURL string) string {
	slug := e.Slug
	if slug == "" {
		slug = e.ID
	}
	path := "/event/" + slug

	img400 := imageFileURL(baseURL, e, "400x300")
	cardClass := "event-card event-card--featured"
	if img400 == "" {
		cardClass += " event-card--no-image"
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf(`<a href="%s" class="%s">`, htmlEscape(path), cardClass))

	if img400 != "" {
		img200 := imageFileURL(baseURL, e, "200x150")
		img800 := imageFileURL(baseURL, e, "800x600")
		b.WriteString(`<div class="event-card-thumb"><img src="` + htmlEscape(img400) + `"`)
		b.WriteString(fmt.Sprintf(` srcset="%s 200w, %s 400w, %s 800w" sizes="(max-width: 600px) 300px, 400px"`,
			htmlEscape(img200), htmlEscape(img400), htmlEscape(img800)))
		b.WriteString(fmt.Sprintf(` alt="%s" width="400" height="300" loading="eager" fetchpriority="high"></div>`, htmlEscape(e.Title)))
	} else {
		b.WriteString(`<div class="event-card-thumb"><div class="event-card-thumb-fallback"></div></div>`)
	}

	b.WriteString(`<div class="event-card-body">`)
	b.WriteString(fmt.Sprintf(`<time class="event-card-date">%s</time>`, htmlEscape(formatEventDate(e.StartDatetime))))
	b.WriteString(fmt.Sprintf(`<h3 class="event-card-title">%s</h3>`, htmlEscape(e.Title)))
	if placeName := expandPlaceName(e); placeName != "" {
		b.WriteString(`<div class="event-card-place"><svg class="icon-pin" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 2C8.13 2 5 5.13 5 9c0 5.25 7 13 7 13s7-7.75 7-13c0-3.87-3.13-7-7-7z"></path><circle cx="12" cy="9" r="2.5"></circle></svg>`)
		b.WriteString(htmlEscape(placeName))
		b.WriteString(`</div>`)
	}
	b.WriteString(`</div></a>`)

	return b.String()
}

func imageFileURL(baseURL string, e recurrence.ExpandedEvent, thumb string) string {
	if e.Image == "" {
		return ""
	}
	realID := e.BaseEventID
	if realID == "" {
		realID = e.ID
	}
	return fmt.Sprintf("%s/api/files/%s/%s/%s?thumb=%s", baseURL, e.CollectionID, realID, e.Image, thumb)
}

func expandPlaceName(e recurrence.ExpandedEvent) string {
	m, ok := e.Expand.(map[string]any)
	if !ok {
		return ""
	}
	place, ok := m["place"].(map[string]any)
	if !ok {
		return ""
	}
	name, _ := place["name"].(string)
	return name
}

// formatEventDate mirrors the client's date-fns format('EEE, MMM d · h:mm a').
// Rendered in Europe/London as a reasonable approximation of the typical
// visitor's timezone — purely cosmetic since this markup is replaced the
// instant the client app mounts and reformats in the viewer's actual locale.
func formatEventDate(startDatetime string) string {
	t, err := time.Parse("2006-01-02 15:04:05.000Z", startDatetime)
	if err != nil {
		return ""
	}
	loc, err := time.LoadLocation("Europe/London")
	if err != nil {
		loc = time.UTC
	}
	return t.In(loc).Format("Mon, Jan 2 · 3:04 PM")
}
