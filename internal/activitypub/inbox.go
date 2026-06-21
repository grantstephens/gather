package activitypub

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

var ErrUnauthorized = errors.New("unauthorized")

type IncomingActivity struct {
	Type   string          `json:"type"`
	ID     string          `json:"id"`
	Actor  string          `json:"actor"`
	Object json.RawMessage `json:"object"`
}

func HandleInbox(app core.App, baseURL string, r *http.Request) error {
	data, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		return err
	}

	if err := verifyHTTPSignature(app, baseURL, r, data); err != nil {
		return fmt.Errorf("%w: %v", ErrUnauthorized, err)
	}

	var activity IncomingActivity
	if err := json.Unmarshal(data, &activity); err != nil {
		app.Logger().Error("ap-inbox: failed to parse activity", "error", err, "body", string(data[:min(len(data), 512)]))
		return err
	}

	app.Logger().Info("ap-inbox: received activity", "type", activity.Type, "actor", activity.Actor, "id", activity.ID)

	switch activity.Type {
	case "Follow":
		go func() {
			if err := handleFollow(app, baseURL, activity); err != nil {
				app.Logger().Error("ap-inbox: failed to handle Follow", "actor", activity.Actor, "error", err)
			}
		}()
	case "Undo":
		go func() {
			if err := handleUndo(app, activity); err != nil {
				app.Logger().Error("ap-inbox: failed to handle Undo", "actor", activity.Actor, "error", err)
			}
		}()
	default:
		app.Logger().Info("ap-inbox: ignoring activity type", "type", activity.Type, "actor", activity.Actor)
	}

	return nil
}

// verifyHTTPSignature checks the HTTP Signature header on an incoming inbox request.
func verifyHTTPSignature(app core.App, baseURL string, r *http.Request, body []byte) error {
	sigHeader := r.Header.Get("Signature")
	if sigHeader == "" {
		return fmt.Errorf("missing Signature header")
	}

	params := parseSignatureHeader(sigHeader)
	keyID := params["keyId"]
	if keyID == "" {
		return fmt.Errorf("missing keyId in Signature header")
	}
	headersList := strings.Fields(params["headers"])
	sigBytes, err := base64.StdEncoding.DecodeString(params["signature"])
	if err != nil {
		return fmt.Errorf("decode signature: %w", err)
	}

	// Reconstruct the signed string
	parts := make([]string, 0, len(headersList))
	for _, h := range headersList {
		switch h {
		case "(request-target)":
			parts = append(parts, fmt.Sprintf("(request-target): post %s", r.URL.RequestURI()))
		case "digest":
			digest := r.Header.Get("Digest")
			// Also verify the digest matches the body
			if strings.HasPrefix(digest, "SHA-256=") {
				expected := "SHA-256=" + base64.StdEncoding.EncodeToString(func() []byte { h := sha256.Sum256(body); return h[:] }())
				if digest != expected {
					return fmt.Errorf("digest mismatch")
				}
			}
			parts = append(parts, fmt.Sprintf("digest: %s", digest))
		default:
			parts = append(parts, fmt.Sprintf("%s: %s", h, r.Header.Get(http.CanonicalHeaderKey(h))))
		}
	}
	signedString := strings.Join(parts, "\n")

	pubKey, err := fetchPublicKey(app, baseURL, keyID)
	if err != nil {
		return fmt.Errorf("fetch public key for %s: %w", keyID, err)
	}

	hashed := sha256.Sum256([]byte(signedString))
	if err := rsa.VerifyPKCS1v15(pubKey, crypto.SHA256, hashed[:], sigBytes); err != nil {
		return fmt.Errorf("signature verification failed: %w", err)
	}

	return nil
}

// parseSignatureHeader parses `key="value",key="value"` into a map.
func parseSignatureHeader(header string) map[string]string {
	params := make(map[string]string)
	for _, part := range strings.Split(header, ",") {
		part = strings.TrimSpace(part)
		idx := strings.IndexByte(part, '=')
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(part[:idx])
		val := strings.TrimSpace(part[idx+1:])
		val = strings.Trim(val, `"`)
		params[key] = val
	}
	return params
}

// fetchPublicKey fetches and parses the RSA public key for a given keyId URL.
func fetchPublicKey(app core.App, baseURL, keyID string) (*rsa.PublicKey, error) {
	// Strip fragment (#main-key) to get the actor URL
	actorURL := keyID
	if idx := strings.IndexByte(keyID, '#'); idx >= 0 {
		actorURL = keyID[:idx]
	}

	privKey, err := loadPrivateKey(app)
	if err != nil {
		return nil, fmt.Errorf("load private key: %w", err)
	}

	req, err := http.NewRequest("GET", actorURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/activity+json")

	if err := signGetRequest(privKey, req, baseURL+"/ap/actor#main-key"); err != nil {
		return nil, fmt.Errorf("sign actor fetch: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("actor fetch returned %d", resp.StatusCode)
	}

	var actor struct {
		PublicKey struct {
			PublicKeyPem string `json:"publicKeyPem"`
		} `json:"publicKey"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&actor); err != nil {
		return nil, fmt.Errorf("decode actor JSON: %w", err)
	}

	block, _ := pem.Decode([]byte(actor.PublicKey.PublicKeyPem))
	if block == nil {
		return nil, fmt.Errorf("failed to decode public key PEM")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse public key: %w", err)
	}

	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("public key is not RSA")
	}

	return rsaPub, nil
}

func handleFollow(app core.App, baseURL string, activity IncomingActivity) error {
	actorInfo, err := fetchActor(app, baseURL, activity.Actor)
	if err != nil {
		return fmt.Errorf("fetchActor %s: %w", activity.Actor, err)
	}

	collection, err := app.FindCollectionByNameOrId("ap_followers")
	if err != nil {
		return err
	}

	existing, _ := app.FindFirstRecordByFilter("ap_followers", "actor_url = {:url}", map[string]any{"url": activity.Actor})
	if existing != nil {
		app.Logger().Info("ap-inbox: already following, re-sending Accept", "actor", activity.Actor)
	} else {
		record := core.NewRecord(collection)
		record.Set("actor_url", activity.Actor)
		record.Set("inbox_url", actorInfo.Inbox)
		record.Set("shared_inbox_url", actorInfo.SharedInbox)

		if err := app.Save(record); err != nil {
			return fmt.Errorf("save follower: %w", err)
		}
		app.Logger().Info("ap-inbox: saved new follower", "actor", activity.Actor, "inbox", actorInfo.Inbox)
	}

	accept := Activity{
		Context: "https://www.w3.org/ns/activitystreams",
		Type:    "Accept",
		ID:      fmt.Sprintf("%s/ap/activities/accept/%d", baseURL, time.Now().UnixNano()),
		Actor:   baseURL + "/ap/actor",
		To:      []string{activity.Actor},
		Object:  activity,
	}

	if err := QueueDelivery(app, accept, actorInfo.Inbox); err != nil {
		return fmt.Errorf("queue Accept delivery: %w", err)
	}
	app.Logger().Info("ap-inbox: Accept queued", "actor", activity.Actor, "inbox", actorInfo.Inbox)
	return nil
}

func handleUndo(app core.App, activity IncomingActivity) error {
	var undoObject struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(activity.Object, &undoObject); err != nil {
		return err
	}

	if undoObject.Type == "Follow" {
		follower, err := app.FindFirstRecordByFilter("ap_followers", "actor_url = {:url}", map[string]any{"url": activity.Actor})
		if err != nil {
			return nil // already gone
		}
		if err := app.Delete(follower); err != nil {
			return err
		}
		app.Logger().Info("ap-inbox: removed follower", "actor", activity.Actor)
	}

	return nil
}

type ActorInfo struct {
	Inbox       string
	SharedInbox string
}

func fetchActor(app core.App, baseURL, actorURL string) (*ActorInfo, error) {
	privKey, err := loadPrivateKey(app)
	if err != nil {
		return nil, fmt.Errorf("load private key: %w", err)
	}

	req, err := http.NewRequest("GET", actorURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/activity+json")

	if err := signGetRequest(privKey, req, baseURL+"/ap/actor#main-key"); err != nil {
		return nil, fmt.Errorf("sign actor fetch: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("actor fetch returned %d", resp.StatusCode)
	}

	var actor struct {
		Inbox     string `json:"inbox"`
		Endpoints struct {
			SharedInbox string `json:"sharedInbox"`
		} `json:"endpoints"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&actor); err != nil {
		return nil, fmt.Errorf("decode actor JSON: %w", err)
	}

	if actor.Inbox == "" {
		return nil, fmt.Errorf("actor has no inbox URL")
	}

	return &ActorInfo{
		Inbox:       actor.Inbox,
		SharedInbox: actor.Endpoints.SharedInbox,
	}, nil
}

func loadPrivateKey(app core.App) (*rsa.PrivateKey, error) {
	settings, err := app.FindFirstRecordByFilter("settings", "id != ''")
	if err != nil {
		return nil, err
	}
	pem := settings.GetString("ap_private_key")
	if pem == "" {
		return nil, fmt.Errorf("no private key configured")
	}
	return ParsePrivateKey(pem)
}
