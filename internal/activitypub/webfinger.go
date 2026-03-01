package activitypub

import (
	"encoding/json"
	"fmt"
	"strings"
)

type WebfingerResponse struct {
	Subject string          `json:"subject"`
	Links   []WebfingerLink `json:"links"`
}

type WebfingerLink struct {
	Rel  string `json:"rel"`
	Type string `json:"type,omitempty"`
	Href string `json:"href,omitempty"`
}

func HandleWebfinger(resource string, baseURL string) ([]byte, error) {
	// Parse resource (format: acct:events@domain)
	if !strings.HasPrefix(resource, "acct:") {
		return nil, fmt.Errorf("invalid resource format")
	}

	parts := strings.SplitN(strings.TrimPrefix(resource, "acct:"), "@", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid resource format")
	}

	username := parts[0]
	if username != "events" {
		return nil, fmt.Errorf("user not found")
	}

	response := WebfingerResponse{
		Subject: resource,
		Links: []WebfingerLink{
			{
				Rel:  "self",
				Type: "application/activity+json",
				Href: baseURL + "/ap/actor",
			},
		},
	}

	return json.MarshalIndent(response, "", "  ")
}
