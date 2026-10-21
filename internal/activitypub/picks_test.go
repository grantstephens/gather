package activitypub

import (
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

const longBlurb = `Birnam Book Festival turns five this weekend, with poetry and a pastry on Saturday and a Hogmanay ghost story on Sunday.

### Poetry Brunch, Birnam Arts, Birnam

Saturday morning at Birnam Arts, you get a free cuppa and a pastry. [Poetry Brunch](/event/poetry-brunch-l9lbpw21) is led by Perthshire poet Morag Anderson.

**Also on:** [Swan Lake](/event/swan-lake-hmruxf7x) is in Pitlochry on Sunday night.`

func TestPicksIntro(t *testing.T) {
	long := strings.Repeat("word ", 200) // 1000 chars, no sentence end
	sentences := strings.Repeat("A fairly short sentence here. ", 30)
	tests := []struct {
		name, in, want string
		check          func(t *testing.T, got string)
	}{
		{name: "first paragraph only", in: longBlurb,
			want: "Birnam Book Festival turns five this weekend, with poetry and a pastry on Saturday and a Hogmanay ghost story on Sunday."},
		{name: "single sentence", in: "Just **one** line.", want: "Just one line."},
		{name: "empty", in: " \n\n ", want: ""},
		{name: "link text kept, url dropped", in: "Try [Swan Lake](/event/swan) now.", want: "Try Swan Lake now."},
		{name: "no sentence end cuts at word", in: long, check: func(t *testing.T, got string) {
			if !strings.HasSuffix(got, "…") || len([]rune(got)) > 501 || strings.Contains(got, "wor…") {
				t.Errorf("bad word cut: %q (%d)", got, len(got))
			}
		}},
		{name: "long paragraph cuts at sentence end", in: sentences, check: func(t *testing.T, got string) {
			if !strings.HasSuffix(got, "here.") || len(got) > 500 {
				t.Errorf("bad sentence cut: %q (%d)", got, len(got))
			}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := picksIntro(tt.in, 500)
			if tt.check != nil {
				tt.check(t, got)
				return
			}
			if got != tt.want {
				t.Errorf("picksIntro() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPicksToNote_LongBlurb(t *testing.T) {
	rec := core.NewRecord(core.NewBaseCollection("picks"))
	rec.Set("title", "Weekend <picks> & more")
	rec.Set("slug", "weekend-1")
	rec.Set("blurb", longBlurb)

	note := picksToNote(rec, "https://example.com")

	for _, want := range []string{
		"<strong>Weekend &lt;picks&gt; &amp; more</strong>",
		"Birnam Book Festival turns five",
		`href="https://example.com/picks/weekend-1"`,
		"Read more",
	} {
		if !strings.Contains(note.Content, want) {
			t.Errorf("missing %q in %q", want, note.Content)
		}
	}
	for _, bad := range []string{"###", "Swan Lake", "/event/", "Also on", "Poetry Brunch", "**"} {
		if strings.Contains(note.Content, bad) {
			t.Errorf("unexpected %q in %q", bad, note.Content)
		}
	}
}
