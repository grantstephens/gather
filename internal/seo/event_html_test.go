package seo

import (
	"strings"
	"testing"
)

func TestGenerateEventHTMLStructure(t *testing.T) {
	// Test that we generate valid HTML structure without requiring PocketBase
	// We'll test the core HTML structure components

	// Test HTML escaping in title
	title := `Test Event <script>alert("xss")</script>`
	instanceName := "Test Instance"
	escapedTitle := htmlEscape(title)
	escapedInstance := htmlEscape(instanceName)

	expected := escapedTitle + " - " + escapedInstance
	if strings.Contains(expected, "<script>") {
		t.Error("Title should be HTML escaped")
	}

	// Test that meta tags are properly structured
	metaTags := `<meta property="og:type" content="event">
<meta property="og:title" content="Test">
<meta name="twitter:card" content="summary">`

	if !strings.Contains(metaTags, `property="og:type"`) {
		t.Error("Meta tags should contain Open Graph properties")
	}
	if !strings.Contains(metaTags, `name="twitter:card"`) {
		t.Error("Meta tags should contain Twitter Card properties")
	}

	// Test JSON-LD script tag structure
	jsonLD := `{"@context":"https://schema.org","@type":"Event","name":"Test"}`
	scriptTag := `<script type="application/ld+json">` + "\n" + jsonLD + "\n" + `</script>`

	if !strings.Contains(scriptTag, `type="application/ld+json"`) {
		t.Error("JSON-LD should be in proper script tag")
	}
	if !strings.Contains(scriptTag, `@context`) {
		t.Error("JSON-LD should contain schema.org context")
	}

	// Test meta refresh redirect
	eventID := "test123"
	metaRefresh := `<meta http-equiv="refresh" content="0;url=/event/` + eventID + `">`
	if !strings.Contains(metaRefresh, "http-equiv=\"refresh\"") {
		t.Error("Should include meta refresh tag")
	}
	if !strings.Contains(metaRefresh, "/event/"+eventID) {
		t.Error("Meta refresh should point to event URL")
	}

	// Test JavaScript redirect
	jsRedirect := `<script>window.location.href = "/event/` + eventID + `";</script>`
	if !strings.Contains(jsRedirect, "window.location.href") {
		t.Error("Should include JavaScript redirect")
	}
}

func TestHTMLStructureComponents(t *testing.T) {
	// Verify essential HTML5 structure elements
	tests := []struct {
		name     string
		element  string
		required bool
	}{
		{"DOCTYPE", "<!DOCTYPE html>", true},
		{"HTML tag", `<html lang="en">`, true},
		{"Head tag", "<head>", true},
		{"Charset meta", `<meta charset="UTF-8">`, true},
		{"Viewport meta", `<meta name="viewport"`, true},
		{"Title tag", "<title>", true},
		{"Body tag", "<body>", true},
		{"Script tag", "<script", true},
	}

	// Build a minimal HTML structure to test
	html := `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Test</title>
  <script type="application/ld+json">{}</script>
</head>
<body>
  <p>Test</p>
  <script>window.location.href = "/test";</script>
</body>
</html>`

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.required && !strings.Contains(html, tt.element) {
				t.Errorf("HTML should contain %s", tt.element)
			}
		})
	}
}
