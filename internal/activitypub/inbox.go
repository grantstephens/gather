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
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

var ErrUnauthorized = errors.New("unauthorized")

type cachedKey struct {
	key       *rsa.PublicKey
	fetchedAt time.Time
}

var keyCache sync.Map // map[string]*cachedKey

const keyCacheTTL = 5 * time.Minute

// signatureVerifier can be replaced in tests to skip HTTP signature checking.
var signatureVerifier = verifyHTTPSignature

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

	if err := signatureVerifier(app, baseURL, r, data); err != nil {
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

// isPrivateOrLoopback returns true if host resolves to a loopback or RFC-1918/link-local address.
// This is used to block SSRF attacks via ActivityPub actor/key URLs.
func isPrivateOrLoopback(host string) (bool, error) {
	// Strip port if present
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}

	addrs, err := net.LookupHost(host)
	if err != nil {
		return false, err
	}

	privateRanges := []string{
		"127.0.0.0/8",
		"::1/128",
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"169.254.0.0/16",
		"fc00::/7",
	}

	var nets []*net.IPNet
	for _, cidr := range privateRanges {
		_, network, _ := net.ParseCIDR(cidr)
		nets = append(nets, network)
	}

	for _, addr := range addrs {
		ip := net.ParseIP(addr)
		if ip == nil {
			continue
		}
		for _, network := range nets {
			if network.Contains(ip) {
				return true, nil
			}
		}
	}

	return false, nil
}

// ssrfCheck is the SSRF guard applied before all outbound AP HTTP requests.
// It can be replaced in tests to allow loopback httptest servers.
var ssrfCheck = defaultSSRFCheck

// defaultSSRFCheck parses rawURL and rejects if the host resolves to a private or loopback address.
func defaultSSRFCheck(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL %q: %w", rawURL, err)
	}
	private, err := isPrivateOrLoopback(u.Host)
	if err != nil {
		return fmt.Errorf("DNS lookup failed for %q: %w", u.Host, err)
	}
	if private {
		return fmt.Errorf("keyId URL resolves to private/loopback address: %s", u.Host)
	}
	return nil
}

// fetchPublicKey retrieves and parses an RSA public key from a remote actor's keyId URL.
// Results are cached for keyCacheTTL to avoid hammering remote servers on every inbox request.
// The URL is checked against private/loopback IP ranges before any outbound request is made.
func fetchPublicKey(app core.App, baseURL, keyID string) (*rsa.PublicKey, error) {
	// Check cache first
	if cached, ok := keyCache.Load(keyID); ok {
		ck := cached.(*cachedKey)
		if time.Since(ck.fetchedAt) < keyCacheTTL {
			return ck.key, nil
		}
		// Expired — delete and re-fetch
		keyCache.Delete(keyID)
	}

	// SSRF protection
	if err := ssrfCheck(keyID); err != nil {
		return nil, err
	}

	privKey, err := loadPrivateKey(app)
	if err != nil {
		return nil, fmt.Errorf("load private key: %w", err)
	}

	req, err := http.NewRequest("GET", keyID, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/activity+json")

	if err := signGetRequest(privKey, req, baseURL+"/ap/actor#main-key"); err != nil {
		return nil, fmt.Errorf("sign key fetch: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("key fetch returned %d", resp.StatusCode)
	}

	var actor struct {
		PublicKey struct {
			PublicKeyPem string `json:"publicKeyPem"`
		} `json:"publicKey"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&actor); err != nil {
		return nil, fmt.Errorf("decode key JSON: %w", err)
	}

	block, _ := pem.Decode([]byte(actor.PublicKey.PublicKeyPem))
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block from %s", keyID)
	}
	pubKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse public key from %s: %w", keyID, err)
	}
	rsaKey, ok := pubKey.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("public key from %s is not RSA", keyID)
	}

	keyCache.Store(keyID, &cachedKey{key: rsaKey, fetchedAt: time.Now()})
	return rsaKey, nil
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
	// SSRF protection
	if err := ssrfCheck(actorURL); err != nil {
		return nil, err
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
