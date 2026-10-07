package seo

import "github.com/pocketbase/pocketbase/core"

// ResolvePlace finds a place by record id or slug. The record id is tried first
// so a slug can never shadow another place's id. When it was found by id and
// the place has a slug, redirectTo is the canonical path so the caller can
// answer 301. Unknown refs return nil.
func ResolvePlace(app core.App, ref string) (place *core.Record, redirectTo string) {
	if p, err := app.FindRecordById("places", ref); err == nil {
		if s := p.GetString("slug"); s != "" && s != ref {
			return p, "/place/" + s
		}
		return p, ""
	}
	if p, err := app.FindFirstRecordByFilter("places", "slug = {:s}", map[string]any{"s": ref}); err == nil {
		return p, ""
	}
	return nil, ""
}
