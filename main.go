package main

import (
	"log"
	"strings"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	_ "gather/migrations"
)

func main() {
	app := pocketbase.New()

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		// Serve embedded frontend
		frontend, err := frontendFS()
		if err != nil {
			return err
		}

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
