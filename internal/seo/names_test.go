package seo

import "testing"

func TestRegionFromInstance(t *testing.T) {
	cases := map[string]string{
		"Perthshire Events": "Perthshire",
		"Gather":            "Gather",
		"Events":            "Events",
		"  Fife Events  ":   "Fife",
	}
	for in, want := range cases {
		if got := regionFromInstance(in); got != want {
			t.Errorf("regionFromInstance(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestHumanizeTag(t *testing.T) {
	cases := map[string]string{
		"strathearn-arts": "Strathearn Arts",
		"theatre":         "Theatre",
		"live_music":      "Live Music",
		"kids  &  family": "Kids & Family",
		"":                "",
		"-":               "",
	}
	for in, want := range cases {
		if got := humanizeTag(in); got != want {
			t.Errorf("humanizeTag(%q) = %q, want %q", in, got, want)
		}
	}
}
