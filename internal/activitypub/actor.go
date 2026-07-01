package activitypub

import (
	"encoding/json"
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

type Actor struct {
	Context           any        `json:"@context"`
	Type              string     `json:"type"`
	ID                string     `json:"id"`
	PreferredUsername string     `json:"preferredUsername"`
	Name              string     `json:"name"`
	Summary           string     `json:"summary,omitempty"`
	URL               string     `json:"url"`
	Inbox             string     `json:"inbox"`
	Outbox            string     `json:"outbox"`
	Followers         string     `json:"followers"`
	Discoverable      bool       `json:"discoverable"`
	Icon              *APImage   `json:"icon,omitempty"`
	PublicKey         *PublicKey `json:"publicKey,omitempty"`
}

type APImage struct {
	Type      string `json:"type"`
	MediaType string `json:"mediaType,omitempty"`
	URL       string `json:"url"`
}

type PublicKey struct {
	ID           string `json:"id"`
	Owner        string `json:"owner"`
	PublicKeyPem string `json:"publicKeyPem"`
}

func GetActor(app core.App, baseURL string) (*Actor, error) {
	settings, err := app.FindFirstRecordByFilter("settings", "id != ''")
	if err != nil {
		return nil, err
	}

	name := settings.GetString("instance_name")
	if name == "" {
		name = "Gather"
	}
	summary := settings.GetString("instance_description")
	publicKey := settings.GetString("ap_public_key")

	actorID := baseURL + "/ap/actor"

	// Build icon from favicon — only use raster formats; Mastodon ignores SVG avatars.
	var icon *APImage
	if favicon := settings.GetString("favicon"); favicon != "" {
		var mediaType string
		switch {
		case strings.HasSuffix(favicon, ".png"):
			mediaType = "image/png"
		case strings.HasSuffix(favicon, ".webp"):
			mediaType = "image/webp"
		case strings.HasSuffix(favicon, ".jpg"), strings.HasSuffix(favicon, ".jpeg"):
			mediaType = "image/jpeg"
		}
		if mediaType != "" {
			icon = &APImage{
				Type:      "Image",
				MediaType: mediaType,
				URL:       baseURL + "/api/files/" + settings.BaseFilesPath() + "/" + favicon,
			}
		}
	}

	actor := &Actor{
		Context: []any{
			"https://www.w3.org/ns/activitystreams",
			"https://w3id.org/security/v1",
		},
		Type:              "Service",
		ID:                actorID,
		PreferredUsername: "events",
		Name:              name,
		Summary:           summary,
		URL:               baseURL,
		Inbox:             baseURL + "/ap/inbox",
		Outbox:            baseURL + "/ap/outbox",
		Followers:         baseURL + "/ap/actor/followers",
		Discoverable:      true,
		Icon:              icon,
	}

	if publicKey != "" {
		actor.PublicKey = &PublicKey{
			ID:           actorID + "#main-key",
			Owner:        actorID,
			PublicKeyPem: publicKey,
		}
	}

	return actor, nil
}

func (a *Actor) ToJSON() ([]byte, error) {
	return json.MarshalIndent(a, "", "  ")
}

// EnsureKeypair creates AP keys if they don't exist
func EnsureKeypair(app core.App) error {
	settings, err := app.FindFirstRecordByFilter("settings", "id != ''")
	if err != nil {
		// Create settings record
		collection, err := app.FindCollectionByNameOrId("settings")
		if err != nil {
			return err
		}
		settings = core.NewRecord(collection)
	}

	if settings.GetString("ap_private_key") != "" {
		return nil // Already has keys
	}

	privateKey, publicKey, err := GenerateKeyPair()
	if err != nil {
		return err
	}

	settings.Set("ap_private_key", privateKey)
	settings.Set("ap_public_key", publicKey)
	settings.Set("ap_enabled", true)

	return app.Save(settings)
}
