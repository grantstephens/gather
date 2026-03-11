package hooks

import (
	"github.com/pocketbase/pocketbase/core"
)

func RegisterUserHooks(app core.App) {
	// Prevent users from setting their own role during registration
	app.OnRecordCreateRequest("users").BindFunc(func(e *core.RecordRequestEvent) error {
		// Allow superusers to set any role
		if e.HasSuperuserAuth() {
			return e.Next()
		}
		// Force role to "user" for regular registration
		e.Record.Set("role", "user")
		return e.Next()
	})
}
