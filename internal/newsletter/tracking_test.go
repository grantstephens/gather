package newsletter

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"gather/internal/api"

	"github.com/pocketbase/pocketbase/core"
)

func TestUmamiTarget(t *testing.T) {
	app := api.NewTestApp(t)
	defer app.Cleanup()

	if ep, id := umamiTarget(app); ep != "" || id != "" {
		t.Errorf("no settings: want empty, got %q %q", ep, id)
	}
	setNewsletterSettings(t, app, true, map[string]any{
		"umami_src":        "https://stats.example.com/script.js",
		"umami_website_id": "site-1",
	})
	if ep, id := umamiTarget(app); ep != "https://stats.example.com" || id != "site-1" {
		t.Errorf("fallback to umami_src origin: got %q %q", ep, id)
	}
	setNewsletterSettings(t, app, true, map[string]any{"umami_host_url": "https://umami.internal/"})
	if ep, _ := umamiTarget(app); ep != "https://umami.internal" {
		t.Errorf("umami_host_url must win and lose its trailing slash: got %q", ep)
	}
	setNewsletterSettings(t, app, true, map[string]any{"umami_website_id": ""})
	if ep, id := umamiTarget(app); ep != "" || id != "" {
		t.Errorf("no website id: want empty, got %q %q", ep, id)
	}
}

func TestReportOpenPostsToUmami(t *testing.T) {
	var gotPath, gotUA string
	var body map[string]any
	umami := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotUA = r.URL.Path, r.UserAgent()
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &body)
		w.WriteHeader(200)
	}))
	defer umami.Close()

	if err := ReportOpen(umami.URL, "site-1", "https://gather.test", "weekend-picks"); err != nil {
		t.Fatal(err)
	}
	if gotPath != "/api/send" {
		t.Errorf("path = %q", gotPath)
	}
	if !strings.HasPrefix(gotUA, "Mozilla/5.0") || strings.Contains(gotUA, "Go-http-client") {
		t.Errorf("User-Agent must look like a browser (Umami drops bots): %q", gotUA)
	}
	payload, _ := body["payload"].(map[string]any)
	data, _ := payload["data"].(map[string]any)
	if body["type"] != "event" || payload["name"] != "newsletter-open" || payload["website"] != "site-1" ||
		payload["hostname"] != "gather.test" || payload["url"] != "/picks/weekend-picks" || data["campaign"] != "weekend-picks" {
		t.Errorf("unexpected payload: %v", body)
	}
}

func TestReportOpenRefusesRedirectsAndBadSchemes(t *testing.T) {
	var followed atomic.Int32
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { followed.Add(1) }))
	defer target.Close()
	redirector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusFound)
	}))
	defer redirector.Close()

	if err := ReportOpen(redirector.URL, "site-1", "https://gather.test", "x"); err == nil {
		t.Error("a redirect response should be reported as an error")
	}
	if followed.Load() != 0 {
		t.Error("redirects must not be followed")
	}
	for _, bad := range []string{"file:///etc/passwd", "ftp://x.test", "gopher://x.test", ""} {
		if err := ReportOpen(bad, "site-1", "https://gather.test", "x"); err == nil {
			t.Errorf("endpoint %q must be rejected", bad)
		}
	}
}

// pixelRequest runs the handler directly with a fake request.
func pixelRequest(t *testing.T, app core.App, id string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/newsletter/open/"+id, nil)
	req.SetPathValue("id", id)
	rr := httptest.NewRecorder()
	re := new(core.RequestEvent)
	re.App = app
	re.Request = req
	re.Response = rr
	if err := PixelHandler(app, testBase)(re); err != nil {
		t.Fatal(err)
	}
	return rr
}

func TestPixelHandler(t *testing.T) {
	hits := make(chan string, 8)
	umami := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		hits <- string(raw)
	}))
	defer umami.Close()

	f := newTickFixture(t)
	defer f.app.Cleanup()
	setNewsletterSettings(t, f.app, true, map[string]any{"umami_host_url": umami.URL, "umami_website_id": "site-1"})

	sent := mkPick(t, f.app, pickSpec{slug: "sent-one", start: "2026-11-06", sent: true})
	unsent := f.pick(t)

	rr := pixelRequest(t, f.app, sent.Id+".gif")
	if rr.Code != 200 || rr.Header().Get("Content-Type") != "image/gif" || !strings.HasPrefix(rr.Body.String(), "GIF89a") {
		t.Fatalf("pixel response wrong: %d %q", rr.Code, rr.Header().Get("Content-Type"))
	}
	if rr.Header().Get("Cache-Control") != "no-store" {
		t.Errorf("Cache-Control = %q", rr.Header().Get("Cache-Control"))
	}
	select {
	case body := <-hits:
		if !strings.Contains(body, "newsletter-open") || !strings.Contains(body, "sent-one") {
			t.Errorf("unexpected forward: %s", body)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("open for a sent pick was not forwarded to Umami")
	}

	// Unsent pick, unknown id and malformed ids still get a GIF but no forward.
	for _, id := range []string{unsent.Id + ".gif", "aaaaaaaaaaaaaaa.gif", "../../etc/passwd", "short.gif", strings.Repeat("a", 200) + ".gif"} {
		rr := pixelRequest(t, f.app, id)
		if rr.Code != 200 || !strings.HasPrefix(rr.Body.String(), "GIF89a") {
			t.Errorf("id %q: want a GIF, got %d", id, rr.Code)
		}
	}
	select {
	case body := <-hits:
		t.Errorf("unexpected forward: %s", body)
	case <-time.After(300 * time.Millisecond):
	}
}
