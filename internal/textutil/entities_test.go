package textutil

import "testing"

func TestDecodeEntities(t *testing.T) {
	cases := map[string]string{
		"Beatrix Potter&#039;s Birnam Walking Tour": "Beatrix Potter's Birnam Walking Tour",
		"Beatrix Potter&amp;#039;s":                 "Beatrix Potter's",
		"Tom &amp; Jerry":                           "Tom & Jerry",
		"Tom &amp;amp; Jerry":                       "Tom & Jerry",
		"St John&#39;s Kirk":                        "St John's Kirk",
		"It&#x27;s":                                 "It's",
		"&quot;Hi&quot;":                            `"Hi"`,
		"a&nbsp;b":                                  "a b",
		"no entities here":                          "no entities here",
		"R&D":                                       "R&D",
		// never introduce markup
		"&lt;script&gt;alert(1)&lt;/script&gt;": "&lt;script&gt;alert(1)&lt;/script&gt;",
		"&#60;b&#62;":                           "&#60;b&#62;",
		"&amp;lt;b&amp;gt;":                     "&lt;b&gt;",
		// invalid code points stay as-is
		"&#0;": "&#0;", "&#1114112;": "&#1114112;", "&#xD800;": "&#xD800;",
	}
	for in, want := range cases {
		if got := DecodeEntities(in); got != want {
			t.Errorf("DecodeEntities(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestDecodeEntitiesIsIdempotent(t *testing.T) {
	once := DecodeEntities("Beatrix Potter&amp;#039;s")
	if twice := DecodeEntities(once); twice != once {
		t.Fatalf("not idempotent: %q then %q", once, twice)
	}
}
