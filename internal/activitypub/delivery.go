package activitypub

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

// DeliverActivity loads the private key from app settings and delivers the activity.
func DeliverActivity(app core.App, activity Activity, inboxURL string) error {
	settings, err := app.FindFirstRecordByFilter("settings", "id != ''")
	if err != nil {
		return err
	}

	privateKeyPem := settings.GetString("ap_private_key")
	if privateKeyPem == "" {
		return fmt.Errorf("no private key configured")
	}

	privateKey, err := ParsePrivateKey(privateKeyPem)
	if err != nil {
		return err
	}

	return deliverActivityWithKey(privateKey, activity.Actor+"#main-key", activity, inboxURL)
}

// deliverActivityWithKey delivers an activity using the provided key — avoids a DB round-trip
// when the caller already holds the key (e.g. the delivery queue worker).
func deliverActivityWithKey(privateKey *rsa.PrivateKey, keyID string, activity Activity, inboxURL string) error {
	body, err := json.Marshal(activity)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", inboxURL, bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/activity+json")
	req.Header.Set("Accept", "application/activity+json")

	if err := signRequest(req, privateKey, keyID, body); err != nil {
		return err
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("delivery failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	return nil
}

func signRequest(req *http.Request, privateKey *rsa.PrivateKey, keyID string, body []byte) error {
	date := time.Now().UTC().Format(http.TimeFormat)
	req.Header.Set("Date", date)

	h := sha256.Sum256(body)
	digest := "SHA-256=" + base64.StdEncoding.EncodeToString(h[:])
	req.Header.Set("Digest", digest)

	signedString := fmt.Sprintf(
		"(request-target): post %s\nhost: %s\ndate: %s\ndigest: %s",
		req.URL.Path,
		req.URL.Host,
		date,
		digest,
	)

	hashed := sha256.Sum256([]byte(signedString))
	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, hashed[:])
	if err != nil {
		return err
	}

	sigHeader := fmt.Sprintf(
		`keyId="%s",algorithm="rsa-sha256",headers="(request-target) host date digest",signature="%s"`,
		keyID,
		base64.StdEncoding.EncodeToString(signature),
	)
	req.Header.Set("Signature", sigHeader)

	return nil
}

// signGetRequest signs a GET request with the provided private key — used for authorized fetch.
func signGetRequest(privateKey *rsa.PrivateKey, req *http.Request, keyID string) error {
	date := time.Now().UTC().Format(http.TimeFormat)
	req.Header.Set("Date", date)

	signedString := fmt.Sprintf(
		"(request-target): get %s\nhost: %s\ndate: %s",
		req.URL.Path,
		req.URL.Host,
		date,
	)

	hashed := sha256.Sum256([]byte(signedString))
	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, hashed[:])
	if err != nil {
		return err
	}

	req.Header.Set("Signature", fmt.Sprintf(
		`keyId="%s",algorithm="rsa-sha256",headers="(request-target) host date",signature="%s"`,
		keyID,
		base64.StdEncoding.EncodeToString(signature),
	))
	return nil
}

// QueueDelivery enqueues a single delivery to the given inbox.
func QueueDelivery(app core.App, activity Activity, inboxURL string) error {
	collection, err := app.FindCollectionByNameOrId("ap_delivery_queue")
	if err != nil {
		return err
	}

	record := core.NewRecord(collection)
	activityJSON, _ := json.Marshal(activity)
	record.Set("activity", string(activityJSON))
	record.Set("inbox_url", inboxURL)
	record.Set("attempts", 0)
	record.Set("next_retry", time.Now())
	return app.Save(record)
}

// QueueDeliveryToFollowers enqueues delivery to all follower inboxes (deduplicated by shared inbox).
func QueueDeliveryToFollowers(app core.App, activity Activity) error {
	followers, err := app.FindRecordsByFilter("ap_followers", "id != ''", "", 0, 0)
	if err != nil {
		return err
	}

	inboxes := make(map[string]bool)
	for _, follower := range followers {
		inbox := follower.GetString("shared_inbox_url")
		if inbox == "" {
			inbox = follower.GetString("inbox_url")
		}
		inboxes[inbox] = true
	}

	for inbox := range inboxes {
		if err := QueueDelivery(app, activity, inbox); err != nil {
			return err
		}
	}

	return nil
}
