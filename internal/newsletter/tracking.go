package newsletter

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"
	"time"

	"gather/internal/seo"

	"github.com/pocketbase/pocketbase/core"
)

// A real browser string: Umami's /api/send silently drops bot user agents, and
// Go's default one looks like a bot.
const umamiUserAgent = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"

var (
	pickIDRe = regexp.MustCompile(`^[a-z0-9]{15}$`)
	// 1x1 transparent GIF.
	pixelGIF, _ = base64.StdEncoding.DecodeString("R0lGODlhAQABAIAAAAAAAP///yH5BAEAAAAALAAAAAABAAEAAAIBRAA7")
	// Bounds concurrent outbound Umami calls; opens beyond this are dropped.
	openSem = make(chan struct{}, 4)
)

// umamiTarget returns where to send server-side events and the website id, or
// empty strings when Umami isn't configured. The host defaults to the origin of
// the tracker script, exactly like the tracker itself. Reverse-proxy setups use
// relative paths (e.g. umami_host_url "/stats"); those are resolved against
// this site's own base URL.
func umamiTarget(app core.App, baseURL string) (endpoint, websiteID string) {
	rec, err := app.FindFirstRecordByFilter("settings", "")
	if err != nil {
		return "", ""
	}
	websiteID = strings.TrimSpace(rec.GetString("umami_website_id"))
	hostURL := strings.TrimRight(strings.TrimSpace(rec.GetString("umami_host_url")), "/")
	src := strings.TrimSpace(rec.GetString("umami_src"))
	switch {
	case strings.HasPrefix(hostURL, "/"):
		endpoint = baseOf(baseURL) + hostURL
	case hostURL != "":
		endpoint = hostURL
	case strings.HasPrefix(src, "/") && !strings.HasPrefix(src, "//"):
		// "/stats/script.js" -> "/stats"; "/script.js" -> the site root
		dir := path.Dir(src)
		if dir == "/" || dir == "." {
			dir = ""
		}
		endpoint = baseOf(baseURL) + dir
	default:
		endpoint = seo.OriginFromURL(src)
	}
	if endpoint == "" || websiteID == "" {
		return "", ""
	}
	return endpoint, websiteID
}

// ReportOpen sends a newsletter-open event to Umami. The endpoint is
// admin-configured, so it is restricted to http(s) and never follows redirects.
func ReportOpen(endpoint, websiteID, baseURL, slug string) error {
	u, err := url.Parse(endpoint)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return fmt.Errorf("umami endpoint %q must be http(s)", endpoint)
	}
	hostname := ""
	if b, err := url.Parse(baseURL); err == nil {
		hostname = b.Hostname()
	}
	body, err := json.Marshal(map[string]any{
		"type": "event",
		"payload": map[string]any{
			"website":  websiteID,
			"hostname": hostname,
			"url":      "/picks/" + slug,
			"name":     "newsletter-open",
			"data":     map[string]any{"campaign": slug},
		},
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, strings.TrimRight(endpoint, "/")+"/api/send", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", umamiUserAgent)
	client := &http.Client{
		Timeout:       3 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return errors.New("umami responded " + resp.Status)
	}
	return nil
}

// PixelHandler serves the per-campaign open-tracking pixel. It always answers
// with the GIF; it forwards to Umami only for a real, already-sent picks post.
func PixelHandler(app core.App, baseURL string) func(*core.RequestEvent) error {
	return func(re *core.RequestEvent) error {
		id := strings.TrimSuffix(re.Request.PathValue("id"), ".gif")
		if pickIDRe.MatchString(id) {
			// The DB lookup happens here, before returning, so no goroutine ever
			// touches the app afterwards.
			if pick, err := app.FindRecordById("picks", id); err == nil && !pick.GetDateTime("newsletter_sent_at").IsZero() {
				if endpoint, websiteID := umamiTarget(app, baseURL); endpoint != "" {
					slug := pick.GetString("slug")
					select {
					case openSem <- struct{}{}:
						go func() {
							defer func() { <-openSem }()
							_ = ReportOpen(endpoint, websiteID, baseURL, slug)
						}()
					default: // saturated: drop this open
					}
				}
			}
		}
		re.Response.Header().Set("Cache-Control", "no-store")
		return re.Blob(http.StatusOK, "image/gif", pixelGIF)
	}
}
