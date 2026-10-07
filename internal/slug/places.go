package slug

import "github.com/pocketbase/pocketbase/core"

// PlaceSlugTaken reports whether a place already uses the slug.
func PlaceSlugTaken(app core.App) func(string) bool {
	return func(s string) bool {
		_, err := app.FindFirstRecordByFilter("places", "slug = {:s}", map[string]any{"s": s})
		return err == nil
	}
}

// ForNewPlace derives a readable, unique slug from a place name:
// "The Space" -> "the-space", then "the-space-2" for the next one.
func ForNewPlace(app core.App, name string) string {
	return Unique(Slugify(name), PlaceSlugTaken(app))
}

// BackfillPlaces gives every place without a slug one, oldest first so the
// first venue with a name gets the clean slug. Idempotent.
func BackfillPlaces(app core.App) error {
	places, err := app.FindRecordsByFilter("places", "slug = ''", "created", 10000, 0)
	if err != nil {
		return err
	}
	for _, p := range places {
		p.Set("slug", ForNewPlace(app, p.GetString("name")))
		if err := app.SaveNoValidate(p); err != nil {
			return err
		}
	}
	return nil
}
