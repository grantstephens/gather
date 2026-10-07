package slug

import "testing"

func TestGenerateDecodesEntitiesFirst(t *testing.T) {
	if got := Generate("Beatrix Potter&#039;s Birnam Walking Tour", "lqyvo7adu66r8v7"); got != "beatrix-potter-s-birnam-walking-tour-lqyvo7ad" {
		t.Fatalf("got %q", got)
	}
}

func TestGenerateKeepsIDSuffixAndFallback(t *testing.T) {
	if got := Generate("", "abc"); got != "event-abc" {
		t.Fatalf("got %q", got)
	}
}

func TestSlugify(t *testing.T) {
	if got := Slugify("  The Space!  "); got != "the-space" {
		t.Fatalf("got %q", got)
	}
	if got := Slugify("!!!"); got != "" {
		t.Fatalf("empty input must give empty slug, got %q", got)
	}
}

func TestUnique(t *testing.T) {
	taken := map[string]bool{"the-space": true, "the-space-2": true}
	if got := Unique("the-space", func(s string) bool { return taken[s] }); got != "the-space-3" {
		t.Fatalf("got %q", got)
	}
	if got := Unique("fresh", func(s string) bool { return false }); got != "fresh" {
		t.Fatalf("got %q", got)
	}
	if got := Unique("", func(s string) bool { return false }); got != "place" {
		t.Fatalf("empty base falls back to 'place', got %q", got)
	}
}
