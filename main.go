package main

import (
	"log"
	"strings"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"gather/internal/activitypub"
	"gather/internal/hooks"
	"gather/internal/ical"
	"gather/internal/rss"
	_ "gather/migrations"
)

func main() {
	app := pocketbase.New()

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		// Initialize AP keypair on first run
		if err := activitypub.EnsureKeypair(se.App); err != nil {
			log.Println("Warning: failed to ensure AP keypair:", err)
		}

		// Serve embedded frontend
		frontend, err := frontendFS()
		if err != nil {
			return err
		}

		baseURL := se.App.Settings().Meta.AppURL
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

		// Register event hooks for ActivityPub delivery
		hooks.RegisterEventHooks(se.App, baseURL)

		// Serve static files, fallback to index.html for SPA routing
		se.Router.GET("/{path...}", func(re *core.RequestEvent) error {
			path := re.Request.PathValue("path")

			// Skip API and admin routes
			if strings.HasPrefix(path, "api/") || strings.HasPrefix(path, "_/") {
				return re.Next()
			}

			// Try to serve the file
			f, err := frontend.Open(path)
			if err == nil {
				f.Close()
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
