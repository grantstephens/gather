package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		users, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}

		users.Fields.Add(&core.SelectField{
			Name:     "role",
			Values:   []string{"user", "editor", "admin"},
		})
		users.Fields.Add(&core.TextField{
			Name: "display_name",
		})

		return app.Save(users)
	}, func(app core.App) error {
		users, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}

		users.Fields.RemoveByName("role")
		users.Fields.RemoveByName("display_name")

		return app.Save(users)
	})
}
