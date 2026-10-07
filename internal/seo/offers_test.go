package seo

import (
	"encoding/json"
	"strings"
	"testing"
)

func offerJSON(cost string) string {
	o := buildOffers(cost, "https://src.test/e", "https://site.test/event/x")
	if o == nil {
		return ""
	}
	b, _ := json.Marshal(o)
	return string(b)
}

func TestBuildOffers(t *testing.T) {
	if got := offerJSON("Pay what you can"); got != "" {
		t.Errorf("unparseable cost must give no offer, got %s", got)
	}
	if got := offerJSON("Free"); got != `{"@type":"Offer","price":"0","priceCurrency":"GBP","availability":"https://schema.org/InStock","url":"https://src.test/e"}` {
		t.Errorf("free: %s", got)
	}
	if got := offerJSON("£5"); got != `{"@type":"Offer","price":"5","priceCurrency":"GBP","availability":"https://schema.org/InStock","url":"https://src.test/e"}` {
		t.Errorf("£5: %s", got)
	}
	if got := offerJSON("£14.30"); !strings.Contains(got, `"price":"14.30"`) {
		t.Errorf("£14.30: %s", got)
	}
	agg := offerJSON("Adults £14.30, Children £2.25")
	for _, want := range []string{`"@type":"AggregateOffer"`, `"lowPrice":"2.25"`, `"highPrice":"14.30"`, `"offerCount":2`} {
		if !strings.Contains(agg, want) {
			t.Errorf("aggregate missing %s in %s", want, agg)
		}
	}
	mixed := offerJSON("Free (Colour Run from £9)")
	for _, want := range []string{`"lowPrice":"0"`, `"highPrice":"9"`} {
		if !strings.Contains(mixed, want) {
			t.Errorf("mixed missing %s in %s", want, mixed)
		}
	}
}

func TestBuildOffersFallsBackToEventURL(t *testing.T) {
	b, _ := json.Marshal(buildOffers("£5", "", "https://site.test/event/x"))
	if !strings.Contains(string(b), `"url":"https://site.test/event/x"`) {
		t.Errorf("got %s", b)
	}
}
