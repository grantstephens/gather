// Package testutil holds helpers shared by tests that need a migrated PocketBase app.
package testutil

import (
	"os"
	"sync"

	"github.com/pocketbase/pocketbase/tests"

	_ "gather/migrations"
)

var (
	templateOnce sync.Once
	templateDir  string
	templateErr  error
)

// NewApp returns a test app with every migration applied. Running ~30 migrations costs
// seconds per app (much more under -race), so they run once per test binary and each app
// is a clone of that migrated data dir. Callers must still call Cleanup on the app.
func NewApp() (*tests.TestApp, error) {
	templateOnce.Do(func() {
		var app *tests.TestApp
		if app, templateErr = tests.NewTestApp(); templateErr != nil {
			return
		}
		defer app.Cleanup()
		if templateErr = app.RunAllMigrations(); templateErr != nil {
			return
		}
		// fold the WAL into the db file so the clone sees every migration
		for _, db := range []string{"data", "auxiliary"} {
			conn := app.DB()
			if db == "auxiliary" {
				conn = app.AuxDB()
			}
			if _, templateErr = conn.NewQuery("PRAGMA wal_checkpoint(TRUNCATE)").Execute(); templateErr != nil {
				return
			}
		}
		templateDir, templateErr = tests.TempDirClone(app.DataDir())
	})
	if templateErr != nil {
		return nil, templateErr
	}
	return tests.NewTestApp(templateDir)
}

// Cleanup removes the migrated template. Call it from TestMain after m.Run.
func Cleanup() {
	if templateDir != "" {
		_ = os.RemoveAll(templateDir)
	}
}
