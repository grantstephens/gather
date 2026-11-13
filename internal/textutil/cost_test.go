package textutil

import "testing"

func TestStripBoilerplate(t *testing.T) {
	cases := []struct{ in, want string }{
		{"A talk.<br><br>Cost: £5<br>Source: https://example.com/e/1", "A talk."},
		{"A talk.\nCost: Free\nSource: https://example.com/x", "A talk."},
		{"Cost: Adults £14.30, Children £2.25", ""},
		{"No boilerplate here", "No boilerplate here"},
		{"Join us.<br>Booking: Required", "Join us.<br>Booking: Required"},
		{"Source: not-a-url", "Source: not-a-url"},
		{"Mentions cost: of living in the text", "Mentions cost: of living in the text"},
	}
	for _, c := range cases {
		if got := StripBoilerplate(c.in); got != c.want {
			t.Errorf("StripBoilerplate(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestParseBoilerplate(t *testing.T) {
	cases := []struct{ in, cost, src string }{
		{"A talk.<br><br>Cost: £5<br>Source: https://example.com/e/1", "£5", "https://example.com/e/1"},
		{"A talk.\nCost: Free\nSource: https://example.com/x", "Free", "https://example.com/x"},
		{"Cost: Adults £14.30, Children £2.25", "Adults £14.30, Children £2.25", ""},
		{"No boilerplate here", "", ""},
		{"Join us.<br>Booking: Required", "", ""},
		{"Source: not-a-url", "", ""},
		{"Mentions cost: of living in the text", "", ""},
	}
	for _, c := range cases {
		cost, src := ParseBoilerplate(c.in)
		if cost != c.cost || src != c.src {
			t.Errorf("%q => (%q, %q), want (%q, %q)", c.in, cost, src, c.cost, c.src)
		}
	}
}
