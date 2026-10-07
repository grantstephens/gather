package seo

import "github.com/pocketbase/pocketbase/core"

// ResolvePlace finds a place by slug, falling back to record id (the old URL
// form). When it was found by id and the place has a slug, redirectTo is the
// canonical path so the caller can answer 301. Unknown refs return nil.
func ResolvePlace(app core.App, ref string) (place *core.Record, redirectTo string) {
	if p, err := app.FindFirstRecordByFilter("places", "slug = {:s}", map[string]any{"s": ref}); err == nil {
		return p, ""
	}
	p, err := app.FindRecordById("places", ref)
	if err != nil {
		return nil, ""
	}
	if s := p.GetString("slug"); s != "" && s != ref {
		return p, "/place/" + s
	}
	return p, ""
}
