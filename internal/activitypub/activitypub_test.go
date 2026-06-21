package activitypub

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"

	_ "gather/migrations"
)

// ---- test helpers ----

func newTestApp(t *testing.T) *tests.TestApp {
	t.Helper()
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("create test app: %v", err)
	}
	if err := app.RunAllMigrations(); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	return app
}

func createSettings(t *testing.T, app *tests.TestApp, name, description string) *core.Record {
	t.Helper()
	col, err := app.FindCollectionByNameOrId("settings")
	if err != nil {
		t.Fatalf("find settings collection: %v", err)
	}
	r := core.NewRecord(col)
	r.Set("instance_name", name)
	r.Set("instance_description", description)
	if err := app.Save(r); err != nil {
		t.Fatalf("save settings: %v", err)
	}
	return r
}

func createSettingsWithKeys(t *testing.T, app *tests.TestApp) *core.Record {
	t.Helper()
	priv, pub, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("generate keypair: %v", err)
	}
	col, err := app.FindCollectionByNameOrId("settings")
	if err != nil {
		t.Fatalf("find settings collection: %v", err)
	}
	r := core.NewRecord(col)
	r.Set("instance_name", "Test Instance")
	r.Set("ap_private_key", priv)
	r.Set("ap_public_key", pub)
	if err := app.Save(r); err != nil {
		t.Fatalf("save settings with keys: %v", err)
	}
	return r
}

func createPublishedEvent(t *testing.T, app *tests.TestApp, title string, start time.Time) *core.Record {
	t.Helper()
	userCol, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatalf("find users collection: %v", err)
	}
	user := core.NewRecord(userCol)
	user.Set("email", fmt.Sprintf("ap-test-%d@example.com", time.Now().UnixNano()))
	user.Set("password", "testpass123")
	user.Set("passwordConfirm", "testpass123")
	user.Set("role", "user")
	_ = app.Save(user)

	col, err := app.FindCollectionByNameOrId("events")
	if err != nil {
		t.Fatalf("find events collection: %v", err)
	}
	r := core.NewRecord(col)
	r.Set("title", title)
	r.Set("start_datetime", start)
	r.Set("status", "published")
	r.Set("author", user.Id)
	if err := app.Save(r); err != nil {
		t.Fatalf("save event: %v", err)
	}
	return r
}

// ---- actor tests ----

func TestGetActor_Fields(t *testing.T) {
	app := newTestApp(t)
	defer app.Cleanup()
	createSettings(t, app, "My Community", "Cool events")

	actor, err := GetActor(app, "https://example.com")
	if err != nil {
		t.Fatalf("GetActor: %v", err)
	}

	if actor.Type != "Service" {
		t.Errorf("type = %q, want Service", actor.Type)
	}
	if actor.ID != "https://example.com/ap/actor" {
		t.Errorf("ID = %q", actor.ID)
	}
	if actor.PreferredUsername != "events" {
		t.Errorf("preferredUsername = %q, want events", actor.PreferredUsername)
	}
	if actor.Name != "My Community" {
		t.Errorf("name = %q, want My Community", actor.Name)
	}
	if actor.Inbox != "https://example.com/ap/inbox" {
		t.Errorf("inbox = %q", actor.Inbox)
	}
	if actor.Outbox != "https://example.com/ap/outbox" {
		t.Errorf("outbox = %q", actor.Outbox)
	}
}

func TestGetActor_Context(t *testing.T) {
	app := newTestApp(t)
	defer app.Cleanup()
	createSettings(t, app, "Test", "")

	actor, err := GetActor(app, "https://example.com")
	if err != nil {
		t.Fatalf("GetActor: %v", err)
	}

	data, err := actor.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON: %v", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	ctx, ok := raw["@context"]
	if !ok || ctx == nil {
		t.Error("actor must have @context")
	}
}

func TestGetActor_PublicKeyIncludedWhenSet(t *testing.T) {
	app := newTestApp(t)
	defer app.Cleanup()
	createSettingsWithKeys(t, app)

	actor, err := GetActor(app, "https://example.com")
	if err != nil {
		t.Fatalf("GetActor: %v", err)
	}

	if actor.PublicKey == nil {
		t.Fatal("publicKey should be present when ap_public_key is set")
	}
	if actor.PublicKey.ID != "https://example.com/ap/actor#main-key" {
		t.Errorf("publicKey.id = %q", actor.PublicKey.ID)
	}
	if actor.PublicKey.Owner != "https://example.com/ap/actor" {
		t.Errorf("publicKey.owner = %q", actor.PublicKey.Owner)
	}
	if !strings.Contains(actor.PublicKey.PublicKeyPem, "BEGIN PUBLIC KEY") {
		t.Error("publicKey.publicKeyPem should be a PEM-encoded public key")
	}
}

func TestGetActor_NoPublicKeyWhenNotSet(t *testing.T) {
	app := newTestApp(t)
	defer app.Cleanup()
	createSettings(t, app, "Test", "")

	actor, err := GetActor(app, "https://example.com")
	if err != nil {
		t.Fatalf("GetActor: %v", err)
	}

	if actor.PublicKey != nil {
		t.Error("publicKey should be nil when ap_public_key is not set")
	}
}

// ---- keypair tests ----

func TestGenerateKeyPair(t *testing.T) {
	priv, pub, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair: %v", err)
	}
	if !strings.Contains(priv, "BEGIN RSA PRIVATE KEY") {
		t.Error("private key should be PEM-encoded RSA PRIVATE KEY")
	}
	if !strings.Contains(pub, "BEGIN PUBLIC KEY") {
		t.Error("public key should be PEM-encoded PUBLIC KEY")
	}
}

func TestParsePrivateKey_RoundTrip(t *testing.T) {
	priv, _, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair: %v", err)
	}
	key, err := ParsePrivateKey(priv)
	if err != nil {
		t.Fatalf("ParsePrivateKey: %v", err)
	}
	if key == nil {
		t.Fatal("expected non-nil key")
	}
	if key.N.BitLen() != 2048 {
		t.Errorf("expected 2048-bit key, got %d", key.N.BitLen())
	}
}

func TestParsePrivateKey_InvalidPEM(t *testing.T) {
	_, err := ParsePrivateKey("not a pem block")
	if err == nil {
		t.Error("expected error for invalid PEM")
	}
}

func TestEnsureKeypair_CreatesKeys(t *testing.T) {
	app := newTestApp(t)
	defer app.Cleanup()
	createSettings(t, app, "Test", "")

	if err := EnsureKeypair(app); err != nil {
		t.Fatalf("EnsureKeypair: %v", err)
	}

	settings, err := app.FindFirstRecordByFilter("settings", "id != ''")
	if err != nil {
		t.Fatalf("find settings: %v", err)
	}
	if settings.GetString("ap_private_key") == "" {
		t.Error("ap_private_key should be set after EnsureKeypair")
	}
	if settings.GetString("ap_public_key") == "" {
		t.Error("ap_public_key should be set after EnsureKeypair")
	}
}

func TestEnsureKeypair_Idempotent(t *testing.T) {
	app := newTestApp(t)
	defer app.Cleanup()
	createSettings(t, app, "Test", "")

	if err := EnsureKeypair(app); err != nil {
		t.Fatalf("first EnsureKeypair: %v", err)
	}
	settings1, _ := app.FindFirstRecordByFilter("settings", "id != ''")
	firstKey := settings1.GetString("ap_private_key")

	if err := EnsureKeypair(app); err != nil {
		t.Fatalf("second EnsureKeypair: %v", err)
	}
	settings2, _ := app.FindFirstRecordByFilter("settings", "id != ''")
	if settings2.GetString("ap_private_key") != firstKey {
		t.Error("EnsureKeypair should not overwrite existing keys")
	}
}

// ---- outbox tests ----

func TestGetOutbox_Structure(t *testing.T) {
	app := newTestApp(t)
	defer app.Cleanup()

	data, err := GetOutbox(app, "https://example.com", 0)
	if err != nil {
		t.Fatalf("GetOutbox: %v", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if raw["type"] != "OrderedCollection" {
		t.Errorf("type = %q, want OrderedCollection", raw["type"])
	}
	if raw["@context"] != "https://www.w3.org/ns/activitystreams" {
		t.Errorf("@context = %q, want activitystreams URI", raw["@context"])
	}
	if raw["id"] != "https://example.com/ap/outbox" {
		t.Errorf("id = %q", raw["id"])
	}
}

func TestGetOutbox_ActivityContext(t *testing.T) {
	app := newTestApp(t)
	defer app.Cleanup()
	createPublishedEvent(t, app, "Outbox Event", time.Now().Add(24*time.Hour))

	data, err := GetOutbox(app, "https://example.com", 1)
	if err != nil {
		t.Fatalf("GetOutbox: %v", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	items, ok := raw["orderedItems"].([]any)
	if !ok || len(items) == 0 {
		t.Fatal("expected at least one item in orderedItems")
	}

	activity, ok := items[0].(map[string]any)
	if !ok {
		t.Fatal("item is not an object")
	}

	// Each activity in the outbox must have @context — required by AP spec
	ctx := activity["@context"]
	if ctx == nil {
		t.Error("activity in outbox is missing @context — AP spec requires it on every top-level object")
	}
}

func TestGetOutbox_ActivityFields(t *testing.T) {
	app := newTestApp(t)
	defer app.Cleanup()
	start := time.Date(2026, 6, 20, 18, 0, 0, 0, time.UTC)
	ev := createPublishedEvent(t, app, "Federated Event", start)

	data, err := GetOutbox(app, "https://example.com", 1)
	if err != nil {
		t.Fatalf("GetOutbox: %v", err)
	}

	var raw map[string]any
	json.Unmarshal(data, &raw)
	items := raw["orderedItems"].([]any)
	activity := items[0].(map[string]any)

	if activity["type"] != "Create" {
		t.Errorf("activity type = %q, want Create", activity["type"])
	}
	expectedID := "https://example.com/ap/activities/" + ev.Id
	if activity["id"] != expectedID {
		t.Errorf("activity id = %q, want %q", activity["id"], expectedID)
	}
	if activity["actor"] != "https://example.com/ap/actor" {
		t.Errorf("activity actor = %q", activity["actor"])
	}
}

func TestGetOutbox_NoteFields(t *testing.T) {
	app := newTestApp(t)
	defer app.Cleanup()
	start := time.Date(2026, 6, 20, 18, 0, 0, 0, time.UTC)
	ev := createPublishedEvent(t, app, "Note Event", start)

	data, err := GetOutbox(app, "https://example.com", 1)
	if err != nil {
		t.Fatalf("GetOutbox: %v", err)
	}

	var raw map[string]any
	json.Unmarshal(data, &raw)
	items := raw["orderedItems"].([]any)
	activity := items[0].(map[string]any)
	note, ok := activity["object"].(map[string]any)
	if !ok {
		t.Fatal("activity.object is not an object")
	}

	if note["type"] != "Note" {
		t.Errorf("note type = %q, want Note", note["type"])
	}
	expectedNoteID := "https://example.com/ap/events/" + ev.Id
	if note["id"] != expectedNoteID {
		t.Errorf("note id = %q, want %q", note["id"], expectedNoteID)
	}
	expectedURL := "https://example.com/event/" + ev.Id
	if note["url"] != expectedURL {
		t.Errorf("note url = %q, want %q", note["url"], expectedURL)
	}
	if note["attributedTo"] != "https://example.com/ap/actor" {
		t.Errorf("note attributedTo = %q", note["attributedTo"])
	}

	// Must be addressed to public
	to, _ := note["to"].([]any)
	hasPublic := false
	for _, addr := range to {
		if addr == "https://www.w3.org/ns/activitystreams#Public" {
			hasPublic = true
		}
	}
	if !hasPublic {
		t.Error("note.to must include the AS#Public address for public events")
	}

	content, _ := note["content"].(string)
	if !strings.Contains(content, "Note Event") {
		t.Errorf("note content should include event title, got: %q", content)
	}
}

func TestGetOutbox_OnlyPublishedEvents(t *testing.T) {
	app := newTestApp(t)
	defer app.Cleanup()

	col, _ := app.FindCollectionByNameOrId("events")
	userCol, _ := app.FindCollectionByNameOrId("users")
	user := core.NewRecord(userCol)
	user.Set("email", "ap-draft@example.com")
	user.Set("password", "testpass123")
	user.Set("passwordConfirm", "testpass123")
	user.Set("role", "user")
	_ = app.Save(user)

	draft := core.NewRecord(col)
	draft.Set("title", "Draft Federated Event")
	draft.Set("start_datetime", time.Now().Add(24*time.Hour))
	draft.Set("status", "draft")
	draft.Set("author", user.Id)
	_ = app.Save(draft)

	data, err := GetOutbox(app, "https://example.com", 0)
	if err != nil {
		t.Fatalf("GetOutbox: %v", err)
	}

	var raw map[string]any
	json.Unmarshal(data, &raw)

	if raw["totalItems"].(float64) != 0 {
		t.Error("draft events should not appear in outbox")
	}
}

// ---- webfinger tests ----

func TestHandleWebfinger_ValidResource(t *testing.T) {
	data, err := HandleWebfinger("acct:events@example.com", "https://example.com")
	if err != nil {
		t.Fatalf("HandleWebfinger: %v", err)
	}

	var resp WebfingerResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if resp.Subject != "acct:events@example.com" {
		t.Errorf("subject = %q, want %q", resp.Subject, "acct:events@example.com")
	}
	if len(resp.Links) == 0 {
		t.Fatal("expected at least one link")
	}

	selfLink := resp.Links[0]
	if selfLink.Rel != "self" {
		t.Errorf("link rel = %q, want self", selfLink.Rel)
	}
	if selfLink.Type != "application/activity+json" {
		t.Errorf("link type = %q, want application/activity+json", selfLink.Type)
	}
	if selfLink.Href != "https://example.com/ap/actor" {
		t.Errorf("link href = %q, want %q", selfLink.Href, "https://example.com/ap/actor")
	}
}

func TestHandleWebfinger_NonAcctResource(t *testing.T) {
	_, err := HandleWebfinger("https://example.com/ap/actor", "https://example.com")
	if err == nil {
		t.Error("expected error for non-acct resource")
	}
}

func TestHandleWebfinger_WrongUsername(t *testing.T) {
	_, err := HandleWebfinger("acct:admin@example.com", "https://example.com")
	if err == nil {
		t.Error("expected error for unknown username")
	}
}

func TestHandleWebfinger_MissingAt(t *testing.T) {
	_, err := HandleWebfinger("acct:eventsexample.com", "https://example.com")
	if err == nil {
		t.Error("expected error for missing @ in resource")
	}
}

// ---- inbox tests ----

// mockActorServer returns a test HTTP server that serves a minimal AP actor response.
func mockActorServer(t *testing.T, inbox string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/activity+json")
		fmt.Fprintf(w, `{"inbox": %q, "endpoints": {"sharedInbox": ""}}`, inbox)
	}))
}

// mockKeyServer returns a test HTTP server that serves an AP actor response with the given public key PEM.
// It is used as the keyId actor endpoint so that verifyHTTPSignature can fetch the public key.
func mockKeyServer(t *testing.T, inbox, pubKeyPEM string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/activity+json")
		resp := map[string]any{
			"inbox": inbox,
			"endpoints": map[string]any{
				"sharedInbox": "",
			},
			"publicKey": map[string]any{
				"publicKeyPem": pubKeyPEM,
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
}

// newSignedInboxRequest builds a properly signed *http.Request for the AP inbox,
// using the app's stored private key. keyServerURL must point to a server that
// returns the corresponding public key so that verifyHTTPSignature can verify it.
func newSignedInboxRequest(t *testing.T, app *tests.TestApp, body []byte, keyServerURL string) *http.Request {
	t.Helper()
	privKey, err := loadPrivateKey(app)
	if err != nil {
		t.Fatalf("loadPrivateKey: %v", err)
	}
	keyID := keyServerURL + "#main-key"
	req := httptest.NewRequest("POST", "/ap/inbox", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/activity+json")
	req.Host = "example.com"
	if err := signRequest(req, privKey, keyID, body); err != nil {
		t.Fatalf("signRequest: %v", err)
	}
	return req
}

func TestHandleInbox_Follow(t *testing.T) {
	app := newTestApp(t)
	defer app.Cleanup()
	settings := createSettingsWithKeys(t, app)

	// Mock the remote inbox that will receive our Accept
	acceptSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}))
	defer acceptSrv.Close()

	// Key server: serves the public key so verifyHTTPSignature can verify our signed request,
	// and also acts as the actor server (provides inbox URL) so fetchActor works.
	keySrv := mockKeyServer(t, acceptSrv.URL, settings.GetString("ap_public_key"))
	defer keySrv.Close()

	bodyBytes := []byte(fmt.Sprintf(`{
		"type": "Follow",
		"id": "https://remote.example/follows/1",
		"actor": %q,
		"object": "https://example.com/ap/actor"
	}`, keySrv.URL))

	req := newSignedInboxRequest(t, app, bodyBytes, keySrv.URL)
	if err := HandleInbox(app, "https://example.com", req); err != nil {
		t.Fatalf("HandleInbox Follow: %v", err)
	}

	// handleFollow runs in a goroutine; wait for it to complete
	time.Sleep(100 * time.Millisecond)

	follower, err := app.FindFirstRecordByFilter("ap_followers", "actor_url != ''")
	if err != nil {
		t.Fatalf("follower record not created: %v", err)
	}
	if follower.GetString("actor_url") != keySrv.URL {
		t.Errorf("actor_url = %q, want %q", follower.GetString("actor_url"), keySrv.URL)
	}
	if follower.GetString("inbox_url") != acceptSrv.URL {
		t.Errorf("inbox_url = %q, want %q", follower.GetString("inbox_url"), acceptSrv.URL)
	}
}

func TestHandleInbox_FollowDeduplicated(t *testing.T) {
	app := newTestApp(t)
	defer app.Cleanup()
	settings := createSettingsWithKeys(t, app)

	acceptSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}))
	defer acceptSrv.Close()

	keySrv := mockKeyServer(t, acceptSrv.URL, settings.GetString("ap_public_key"))
	defer keySrv.Close()

	followBody := func() []byte {
		return []byte(fmt.Sprintf(`{
			"type": "Follow",
			"id": "https://remote.example/follows/1",
			"actor": %q,
			"object": "https://example.com/ap/actor"
		}`, keySrv.URL))
	}

	req1 := newSignedInboxRequest(t, app, followBody(), keySrv.URL)
	if err := HandleInbox(app, "https://example.com", req1); err != nil {
		t.Fatalf("first Follow: %v", err)
	}
	req2 := newSignedInboxRequest(t, app, followBody(), keySrv.URL)
	if err := HandleInbox(app, "https://example.com", req2); err != nil {
		t.Fatalf("second Follow: %v", err)
	}

	// Wait a moment for any goroutines to settle
	time.Sleep(50 * time.Millisecond)

	followers, err := app.FindRecordsByFilter("ap_followers", "id != ''", "", 0, 0)
	if err != nil {
		t.Fatalf("find followers: %v", err)
	}
	if len(followers) != 1 {
		t.Errorf("expected 1 follower after duplicate Follow, got %d", len(followers))
	}
}

func TestHandleInbox_UndoFollow(t *testing.T) {
	app := newTestApp(t)
	defer app.Cleanup()
	settings := createSettingsWithKeys(t, app)

	acceptSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}))
	defer acceptSrv.Close()

	keySrv := mockKeyServer(t, acceptSrv.URL, settings.GetString("ap_public_key"))
	defer keySrv.Close()

	// First, follow
	followBodyBytes := []byte(fmt.Sprintf(`{
		"type": "Follow",
		"id": "https://remote.example/follows/1",
		"actor": %q,
		"object": "https://example.com/ap/actor"
	}`, keySrv.URL))
	req1 := newSignedInboxRequest(t, app, followBodyBytes, keySrv.URL)
	if err := HandleInbox(app, "https://example.com", req1); err != nil {
		t.Fatalf("Follow: %v", err)
	}

	// Wait for Follow goroutine to save the follower record before sending Undo
	time.Sleep(100 * time.Millisecond)

	// Then, undo
	undoBodyBytes := []byte(fmt.Sprintf(`{
		"type": "Undo",
		"id": "https://remote.example/undos/1",
		"actor": %q,
		"object": {"type": "Follow", "object": "https://example.com/ap/actor"}
	}`, keySrv.URL))
	req2 := newSignedInboxRequest(t, app, undoBodyBytes, keySrv.URL)
	if err := HandleInbox(app, "https://example.com", req2); err != nil {
		t.Fatalf("Undo: %v", err)
	}

	// Wait for Undo goroutine to delete the follower record
	time.Sleep(100 * time.Millisecond)

	followers, err := app.FindRecordsByFilter("ap_followers", "id != ''", "", 0, 0)
	if err != nil {
		t.Fatalf("find followers: %v", err)
	}
	if len(followers) != 0 {
		t.Errorf("expected 0 followers after Undo, got %d", len(followers))
	}
}

func TestHandleInbox_UnknownActivityIgnored(t *testing.T) {
	app := newTestApp(t)
	defer app.Cleanup()
	settings := createSettingsWithKeys(t, app)

	// Need a key server so verifyHTTPSignature can fetch the public key
	keySrv := mockKeyServer(t, "https://remote.example/inbox", settings.GetString("ap_public_key"))
	defer keySrv.Close()

	bodyBytes := []byte(fmt.Sprintf(`{
		"type": "Like",
		"id": "https://remote.example/likes/1",
		"actor": %q,
		"object": "https://example.com/ap/events/abc"
	}`, keySrv.URL))

	req := newSignedInboxRequest(t, app, bodyBytes, keySrv.URL)
	if err := HandleInbox(app, "https://example.com", req); err != nil {
		t.Errorf("unknown activity type should be silently ignored, got error: %v", err)
	}
}

// ---- delivery queue tests ----

func TestQueueDeliveryToFollowers_NoFollowers(t *testing.T) {
	app := newTestApp(t)
	defer app.Cleanup()

	activity := Activity{
		Context: "https://www.w3.org/ns/activitystreams",
		Type:    "Create",
		ID:      "https://example.com/ap/activities/test",
		Actor:   "https://example.com/ap/actor",
	}

	if err := QueueDeliveryToFollowers(app, activity); err != nil {
		t.Fatalf("QueueDeliveryToFollowers with no followers: %v", err)
	}

	queue, err := app.FindRecordsByFilter("ap_delivery_queue", "id != ''", "", 0, 0)
	if err != nil {
		t.Fatalf("find queue: %v", err)
	}
	if len(queue) != 0 {
		t.Errorf("expected empty queue with no followers, got %d", len(queue))
	}
}

func TestQueueDeliveryToFollowers_CreatesQueueEntry(t *testing.T) {
	app := newTestApp(t)
	defer app.Cleanup()

	// Add a follower
	col, _ := app.FindCollectionByNameOrId("ap_followers")
	follower := core.NewRecord(col)
	follower.Set("actor_url", "https://remote.example/users/alice")
	follower.Set("inbox_url", "https://remote.example/users/alice/inbox")
	_ = app.Save(follower)

	activity := Activity{
		Context: "https://www.w3.org/ns/activitystreams",
		Type:    "Create",
		ID:      "https://example.com/ap/activities/test",
		Actor:   "https://example.com/ap/actor",
	}

	if err := QueueDeliveryToFollowers(app, activity); err != nil {
		t.Fatalf("QueueDeliveryToFollowers: %v", err)
	}

	queue, err := app.FindRecordsByFilter("ap_delivery_queue", "id != ''", "", 0, 0)
	if err != nil {
		t.Fatalf("find queue: %v", err)
	}
	if len(queue) != 1 {
		t.Fatalf("expected 1 queue entry, got %d", len(queue))
	}
	if queue[0].GetString("inbox_url") != "https://remote.example/users/alice/inbox" {
		t.Errorf("queue inbox_url = %q", queue[0].GetString("inbox_url"))
	}
}

func TestQueueDeliveryToFollowers_DeduplicatesSharedInbox(t *testing.T) {
	app := newTestApp(t)
	defer app.Cleanup()

	col, _ := app.FindCollectionByNameOrId("ap_followers")

	// Two followers sharing the same shared inbox (e.g. same Mastodon instance)
	for i, actor := range []string{"alice", "bob"} {
		follower := core.NewRecord(col)
		follower.Set("actor_url", fmt.Sprintf("https://mastodon.example/users/%s-%d", actor, i))
		follower.Set("inbox_url", fmt.Sprintf("https://mastodon.example/users/%s/inbox", actor))
		follower.Set("shared_inbox_url", "https://mastodon.example/inbox")
		_ = app.Save(follower)
	}

	activity := Activity{
		Context: "https://www.w3.org/ns/activitystreams",
		Type:    "Create",
		ID:      "https://example.com/ap/activities/test",
		Actor:   "https://example.com/ap/actor",
	}

	if err := QueueDeliveryToFollowers(app, activity); err != nil {
		t.Fatalf("QueueDeliveryToFollowers: %v", err)
	}

	queue, err := app.FindRecordsByFilter("ap_delivery_queue", "id != ''", "", 0, 0)
	if err != nil {
		t.Fatalf("find queue: %v", err)
	}
	// Should be 1 entry (shared inbox deduplicated), not 2
	if len(queue) != 1 {
		t.Errorf("expected 1 queue entry (shared inbox deduped), got %d", len(queue))
	}
	if queue[0].GetString("inbox_url") != "https://mastodon.example/inbox" {
		t.Errorf("should use shared inbox URL, got %q", queue[0].GetString("inbox_url"))
	}
}
