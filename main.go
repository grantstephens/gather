package main

import (
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"

	dbx "github.com/pocketbase/dbx"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"gather/internal/activitypub"
	"gather/internal/hooks"
	"gather/internal/ical"
	"gather/internal/rss"
	"gather/internal/seo"
	_ "gather/migrations"
)

func main() {
	app := pocketbase.New()

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		// Security headers middleware
		se.Router.BindFunc(func(e *core.RequestEvent) error {
			path := e.Request.URL.Path
			h := e.Response.Header()

			// Apply to all routes
			h.Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
			h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("X-Frame-Options", "DENY")
			h.Set("X-Permitted-Cross-Domain-Policies", "none")
			h.Set("Cross-Origin-Opener-Policy", "same-origin")
			h.Set("Cross-Origin-Resource-Policy", "same-origin")
			h.Set("Permissions-Policy", "accelerometer=(), autoplay=(), camera=(), display-capture=(), encrypted-media=(), fullscreen=(), geolocation=(), gyroscope=(), magnetometer=(), microphone=(), midi=(), payment=(), picture-in-picture=(), publickey-credentials-get=(), screen-wake-lock=(), usb=(), web-share=()")

			// CSP only for frontend routes — skip admin UI and API
			if !strings.HasPrefix(path, "/_/") && !strings.HasPrefix(path, "/api/") {
				h.Set("Content-Security-Policy",
					"default-src 'self'; "+
						"img-src 'self' https://*.tile.openstreetmap.org data: blob:; "+
						"style-src 'self' 'unsafe-inline'; "+
						"connect-src 'self'; "+
						"frame-ancestors 'none'; "+
						"object-src 'none'; "+
						"base-uri 'self'")
			}

			return e.Next()
		})

		// Initialize AP keypair on first run
		if err := activitypub.EnsureKeypair(se.App); err != nil {
			log.Println("Warning: failed to ensure AP keypair:", err)
		}

		// Serve embedded frontend
		frontend, err := frontendFS()
		if err != nil {
			return err
		}

		baseURL := os.Getenv("BASE_URL")
		if baseURL == "" {
			baseURL = se.App.Settings().Meta.AppURL
		}
		if baseURL == "" {
			baseURL = "http://127.0.0.1:8090"
		}

		// RSS feeds
		se.Router.GET("/feed.rss", func(re *core.RequestEvent) error {
			data, err := rss.GenerateFeed(se.App, baseURL, "")
			if err != nil {
				return re.InternalServerError("Failed to generate feed", err)
			}
			re.Response.Header().Set("Content-Type", "application/rss+xml")
			re.Response.Header().Set("Cache-Control", "public, max-age=300, stale-while-revalidate=60")
			return re.Blob(200, "application/rss+xml", data)
		})

		se.Router.GET("/feed/tag/{tagname}", func(re *core.RequestEvent) error {
			tagname := re.Request.PathValue("tagname")
			// Strip .rss extension if present
			tagname = strings.TrimSuffix(tagname, ".rss")
			tag, err := se.App.FindFirstRecordByFilter("tags", "name = {:name}", map[string]any{"name": tagname})
			if err != nil {
				return re.NotFoundError("Tag not found", err)
			}
			filter := "tags.id ?= '" + tag.Id + "'"
			data, err := rss.GenerateFeed(se.App, baseURL, filter)
			if err != nil {
				return re.InternalServerError("Failed to generate feed", err)
			}
			re.Response.Header().Set("Content-Type", "application/rss+xml")
			re.Response.Header().Set("Cache-Control", "public, max-age=300, stale-while-revalidate=60")
			return re.Blob(200, "application/rss+xml", data)
		})

		// ICS feeds
		se.Router.GET("/feed.ics", func(re *core.RequestEvent) error {
			data, err := ical.GenerateFeed(se.App, baseURL, "")
			if err != nil {
				return re.InternalServerError("Failed to generate feed", err)
			}
			re.Response.Header().Set("Content-Type", "text/calendar")
			re.Response.Header().Set("Cache-Control", "public, max-age=300, stale-while-revalidate=60")
			return re.Blob(200, "text/calendar", data)
		})

		se.Router.GET("/ics/tag/{tagname}", func(re *core.RequestEvent) error {
			tagname := re.Request.PathValue("tagname")
			tag, err := se.App.FindFirstRecordByFilter("tags", "name = {:name}", map[string]any{"name": tagname})
			if err != nil {
				return re.NotFoundError("Tag not found", err)
			}
			filter := "tags.id ?= '" + tag.Id + "'"
			data, err := ical.GenerateFeed(se.App, baseURL, filter)
			if err != nil {
				return re.InternalServerError("Failed to generate feed", err)
			}
			re.Response.Header().Set("Content-Type", "text/calendar")
			re.Response.Header().Set("Cache-Control", "public, max-age=300, stale-while-revalidate=60")
			return re.Blob(200, "text/calendar", data)
		})

		se.Router.GET("/ics/event/{id}", func(re *core.RequestEvent) error {
			id := re.Request.PathValue("id")
			data, err := ical.GenerateSingleEvent(se.App, baseURL, id)
			if err != nil {
				return re.NotFoundError("Event not found", err)
			}
			re.Response.Header().Set("Content-Type", "text/calendar")
			re.Response.Header().Set("Cache-Control", "public, max-age=300, stale-while-revalidate=60")
			return re.Blob(200, "text/calendar", data)
		})

		// Sitemap
		se.Router.GET("/sitemap.xml", func(re *core.RequestEvent) error {
			data, err := seo.GenerateSitemap(se.App, baseURL)
			if err != nil {
				return re.InternalServerError("Failed to generate sitemap", err)
			}
			re.Response.Header().Set("Cache-Control", "public, max-age=3600")
			return re.Blob(200, "application/xml", data)
		})

		// Robots.txt
		se.Router.GET("/robots.txt", func(re *core.RequestEvent) error {
			content := seo.BuildRobotsTxt(baseURL)
			re.Response.Header().Set("Cache-Control", "public, max-age=86400")
			return re.String(200, content)
		})

		// ActivityPub actor
		se.Router.GET("/ap/actor", func(re *core.RequestEvent) error {
			actor, err := activitypub.GetActor(se.App, baseURL)
			if err != nil {
				return re.InternalServerError("Failed to get actor", err)
			}
			data, err := actor.ToJSON()
			if err != nil {
				return re.InternalServerError("Failed to serialize actor", err)
			}
			re.Response.Header().Set("Content-Type", "application/activity+json")
			re.Response.Header().Set("Cache-Control", "public, max-age=3600")
			return re.Blob(200, "application/activity+json", data)
		})

		// Webfinger
		se.Router.GET("/.well-known/webfinger", func(re *core.RequestEvent) error {
			resource := re.Request.URL.Query().Get("resource")
			if resource == "" {
				return re.BadRequestError("Missing resource parameter", nil)
			}
			data, err := activitypub.HandleWebfinger(resource, baseURL)
			if err != nil {
				return re.NotFoundError("Resource not found", err)
			}
			re.Response.Header().Set("Content-Type", "application/jrd+json")
			re.Response.Header().Set("Cache-Control", "public, max-age=3600")
			return re.Blob(200, "application/jrd+json", data)
		})

		// ActivityPub outbox
		se.Router.GET("/ap/outbox", func(re *core.RequestEvent) error {
			data, err := activitypub.GetOutbox(se.App, baseURL)
			if err != nil {
				return re.InternalServerError("Failed to get outbox", err)
			}
			re.Response.Header().Set("Content-Type", "application/activity+json")
			re.Response.Header().Set("Cache-Control", "public, max-age=300")
			return re.Blob(200, "application/activity+json", data)
		})

		// ActivityPub inbox
		se.Router.POST("/ap/inbox", func(re *core.RequestEvent) error {
			if err := activitypub.HandleInbox(se.App, baseURL, re.Request.Body); err != nil {
				return re.InternalServerError("Failed to process activity", err)
			}
			return re.NoContent(202)
		})

		// Favicon route - serves logo from settings
		se.Router.GET("/favicon.ico", func(re *core.RequestEvent) error {
			settings, err := se.App.FindFirstRecordByFilter("settings", "")
			if err != nil {
				return re.NotFoundError("No favicon configured", nil)
			}

			logo := settings.GetString("logo")
			if logo == "" {
				return re.NotFoundError("No favicon configured", nil)
			}

			fs, err := se.App.NewFilesystem()
			if err != nil {
				return re.NotFoundError("No favicon configured", nil)
			}
			defer fs.Close()

			filePath := settings.BaseFilesPath() + "/" + logo
			reader, err := fs.GetReader(filePath)
			if err != nil {
				return re.NotFoundError("No favicon configured", nil)
			}
			defer reader.Close()

			contentType := "image/x-icon"
			if strings.HasSuffix(logo, ".png") {
				contentType = "image/png"
			} else if strings.HasSuffix(logo, ".webp") {
				contentType = "image/webp"
			} else if strings.HasSuffix(logo, ".jpg") || strings.HasSuffix(logo, ".jpeg") {
				contentType = "image/jpeg"
			}

			re.Response.Header().Set("Content-Type", contentType)
			re.Response.Header().Set("Cache-Control", "public, max-age=3600, stale-while-revalidate=300")
			re.Response.WriteHeader(http.StatusOK)
			_, err = io.Copy(re.Response, reader)
			return err
		})

		// Tag event counts — single SQL query using json_each to unnest relation arrays
		se.Router.GET("/api/tags/counts", func(re *core.RequestEvent) error {
			today := time.Now().UTC().Format("2006-01-02")
			type row struct {
				ID    string `db:"id"    json:"id"`
				Count int    `db:"count" json:"count"`
			}
			var rows []row
			err := se.App.DB().NewQuery(`
				SELECT t.id, COALESCE(cnt.count, 0) AS count
				FROM tags t
				LEFT JOIN (
					SELECT je.value AS tag_id, COUNT(*) AS count
					FROM events e, json_each(e.tags) je
					WHERE e.status = 'published'
					  AND e.start_datetime >= {:today}
					GROUP BY je.value
				) cnt ON cnt.tag_id = t.id
				WHERE t.status = 'approved'
			`).Bind(dbx.Params{"today": today}).All(&rows)
			if err != nil {
				return re.InternalServerError("Failed to query tag counts", err)
			}
			re.Response.Header().Set("Cache-Control", "public, max-age=300, stale-while-revalidate=60")
			return re.JSON(200, rows)
		})

		// Register hooks
		hooks.RegisterUserHooks(se.App)
		hooks.RegisterEventHooks(se.App, baseURL)
		hooks.RegisterModerationHooks(se.App)
		hooks.RegisterImageConversionHooks(se.App)
		hooks.RegisterSlugHooks(se.App)

		// Dev mode: proxy to Vite dev server
		devMode := os.Getenv("DEV") != ""
		var viteProxy *httputil.ReverseProxy
		if devMode {
			viteURL, _ := url.Parse("http://localhost:5173")
			viteProxy = httputil.NewSingleHostReverseProxy(viteURL)
			log.Println("Dev mode: proxying frontend to http://localhost:5173")
		}

		// Schema.org Event metadata for bots
		se.Router.GET("/event/{id}", func(re *core.RequestEvent) error {
			userAgent := re.Request.Header.Get("User-Agent")

			// Non-bots get the SPA
			if !seo.IsBot(userAgent) {
				return re.FileFS(frontend, "index.html")
			}

			id := re.Request.PathValue("id")

			// Try slug lookup first, then fall back to ID (backward compat)
			event, err := se.App.FindFirstRecordByFilter("events", "slug = {:slug}", map[string]any{"slug": id})
			if err != nil {
				event, err = se.App.FindRecordById("events", id)
			}
			if err != nil {
				return re.FileFS(frontend, "index.html") // Let SPA handle 404
			}

			// Only serve metadata for published events
			if event.GetString("status") != "published" {
				return re.FileFS(frontend, "index.html")
			}

			// Generate HTML with metadata
			html, err := seo.GenerateEventHTML(se.App, event, baseURL)
			if err != nil {
				log.Println("Failed to generate SEO HTML:", err)
				return re.FileFS(frontend, "index.html") // Fallback to SPA
			}

			re.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
			re.Response.Header().Set("Cache-Control", "public, max-age=3600, stale-while-revalidate=300")
			return re.Blob(200, "text/html", html)
		})

		// Home page: SSR for bots, SPA/Vite for humans
		// Use /{$} to match only exact root — GET / conflicts with GET /{path...} in Go 1.22+
		se.Router.GET("/{$}", func(re *core.RequestEvent) error {
			userAgent := re.Request.Header.Get("User-Agent")
			if !seo.IsBot(userAgent) {
				if devMode && viteProxy != nil {
					viteProxy.ServeHTTP(re.Response, re.Request)
					return nil
				}
				return re.FileFS(frontend, "index.html")
			}

			html, err := seo.GenerateHomeHTML(se.App, baseURL)
			if err != nil {
				log.Println("Failed to generate home SEO HTML:", err)
				return re.FileFS(frontend, "index.html")
			}

			re.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
			re.Response.Header().Set("Cache-Control", "public, max-age=300, stale-while-revalidate=60")
			return re.Blob(200, "text/html", html)
		})

		// Serve static files, fallback to index.html for SPA routing
		se.Router.GET("/{path...}", func(re *core.RequestEvent) error {
			path := re.Request.PathValue("path")

			// Skip API and admin routes
			if strings.HasPrefix(path, "api/") || strings.HasPrefix(path, "_/") {
				return re.Next()
			}

			// Dev mode: proxy to Vite
			if devMode && viteProxy != nil {
				viteProxy.ServeHTTP(re.Response, re.Request)
				return nil
			}

			// Try to serve the file
			f, err := frontend.Open(path)
			if err == nil {
				f.Close()
				// Vite assets have content hashes — safe to cache for 1 year
				if strings.HasPrefix(path, "assets/") {
					re.Response.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
				}
				return re.FileFS(frontend, path)
			}

			// Fallback to index.html for SPA
			return re.FileFS(frontend, "index.html")
		})

		return se.Next()
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
