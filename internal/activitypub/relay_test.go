package activitypub

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

func addRelayFollower(t *testing.T, app *tests.TestApp, actor, inbox, shared string) {
	t.Helper()
	col, err := app.FindCollectionByNameOrId("ap_relay_followers")
	if err != nil {
		t.Fatalf("find ap_relay_followers: %v", err)
	}
	r := core.NewRecord(col)
	r.Set("actor_url", actor)
	r.Set("inbox_url", inbox)
	r.Set("shared_inbox_url", shared)
	if err := app.Save(r); err != nil {
		t.Fatalf("save relay follower: %v", err)
	}
}

func queuedActivities(t *testing.T, app core.App) []map[string]any {
	t.Helper()
	recs, err := app.FindRecordsByFilter("ap_delivery_queue", "id != ''", "next_retry", 0, 0)
	if err != nil {
		t.Fatalf("query queue: %v", err)
	}
	out := make([]map[string]any, 0, len(recs))
	for _, r := range recs {
		var m map[string]any
		if err := json.Unmarshal([]byte(r.GetString("activity")), &m); err != nil {
			t.Fatalf("queued activity JSON: %v", err)
		}
		m["_inbox"] = r.GetString("inbox_url")
		out = append(out, m)
	}
	return out
}

func relayInboxRequest(body string) *http.Request {
	return httptest.NewRequest("POST", "/ap/relay/inbox", strings.NewReader(body))
}

func TestRelayActorJSON_IsApplicationWithOwnInbox(t *testing.T) {
	app := newTestApp(t)
	defer app.Cleanup()
	createSettingsWithKeys(t, app)

	b, err := RelayActorJSON(app, testBase)
	if err != nil {
		t.Fatalf("RelayActorJSON: %v", err)
	}
	var a map[string]any
	if err := json.Unmarshal(b, &a); err != nil {
		t.Fatal(err)
	}
	if a["type"] != "Application" || a["id"] != testBase+"/ap/relay" {
		t.Errorf("type/id = %v / %v", a["type"], a["id"])
	}
	if a["preferredUsername"] != "relay" || a["inbox"] != testBase+"/ap/relay/inbox" {
		t.Errorf("username/inbox = %v / %v", a["preferredUsername"], a["inbox"])
	}
	key := a["publicKey"].(map[string]any)
	if key["id"] != testBase+"/ap/relay#main-key" || key["owner"] != testBase+"/ap/relay" {
		t.Errorf("publicKey = %v", key)
	}
}

func TestRelayInbox_RejectsBadSignature(t *testing.T) {
	app := newTestApp(t)
	defer app.Cleanup()
	createSettingsWithKeys(t, app)
	orig := signatureVerifier
	signatureVerifier = func(core.App, string, *http.Request, []byte) error { return errors.New("bad sig") }
	defer func() { signatureVerifier = orig }()

	err := HandleRelayInbox(app, testBase, relayInboxRequest(`{"type":"Follow","actor":"https://remote.example/a"}`))
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("err = %v, want ErrUnauthorized", err)
	}
}

func TestRelayInbox_FollowIsPersistedAndAcceptedFromRelay(t *testing.T) {
	app := newTestApp(t)
	defer app.Cleanup()
	createSettingsWithKeys(t, app)
	inboxSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(202) }))
	defer inboxSrv.Close()
	actorSrv := mockActorServer(t, inboxSrv.URL)
	defer actorSrv.Close()

	body := fmt.Sprintf(`{"type":"Follow","id":"https://remote.example/f/1","actor":%q,"object":%q}`, actorSrv.URL, testBase+"/ap/relay")
	if err := HandleRelayInbox(app, testBase, relayInboxRequest(body)); err != nil {
		t.Fatalf("HandleRelayInbox: %v", err)
	}
	time.Sleep(200 * time.Millisecond)

	f, err := app.FindFirstRecordByFilter("ap_relay_followers", "actor_url = {:a}", map[string]any{"a": actorSrv.URL})
	if err != nil {
		t.Fatalf("relay follower not saved: %v", err)
	}
	if f.GetString("inbox_url") != inboxSrv.URL {
		t.Errorf("inbox_url = %q", f.GetString("inbox_url"))
	}
	if n, _ := app.CountRecords("ap_followers"); n != 0 {
		t.Errorf("relay follow leaked into ap_followers (%d)", n)
	}
	q := queuedActivities(t, app)
	if len(q) != 1 || q[0]["type"] != "Accept" || q[0]["actor"] != testBase+"/ap/relay" {
		t.Fatalf("queue = %v, want one Accept from the relay actor", q)
	}
}

func TestRelayInbox_UndoFollowRemovesFollower(t *testing.T) {
	app := newTestApp(t)
	defer app.Cleanup()
	createSettingsWithKeys(t, app)
	addRelayFollower(t, app, "https://remote.example/a", "https://remote.example/a/inbox", "")

	body := `{"type":"Undo","actor":"https://remote.example/a","object":{"type":"Follow","actor":"https://remote.example/a"}}`
	if err := HandleRelayInbox(app, testBase, relayInboxRequest(body)); err != nil {
		t.Fatal(err)
	}
	time.Sleep(200 * time.Millisecond)

	if n, _ := app.CountRecords("ap_relay_followers"); n != 0 {
		t.Errorf("relay followers = %d, want 0", n)
	}
}

func TestQueueRelayEvent_CreateSendsEventThenAnnouncePerInbox(t *testing.T) {
	app := newTestApp(t)
	defer app.Cleanup()
	createSettingsWithKeys(t, app)
	setTimezone(t, app, "Europe/London")
	// two followers sharing one shared inbox, one with its own
	addRelayFollower(t, app, "https://a.example/a", "https://a.example/a/inbox", "https://a.example/inbox")
	addRelayFollower(t, app, "https://a.example/b", "https://a.example/b/inbox", "https://a.example/inbox")
	addRelayFollower(t, app, "https://b.example/c", "https://b.example/c/inbox", "")
	ev := createPublishedEvent(t, app, "Summer Gig", time.Now().Add(48*time.Hour))

	if err := QueueRelayEvent(app, testBase, ev, "Create"); err != nil {
		t.Fatalf("QueueRelayEvent: %v", err)
	}

	got := map[string][]string{} // inbox -> activity types in queue order
	for _, q := range queuedActivities(t, app) {
		inbox := q["_inbox"].(string)
		got[inbox] = append(got[inbox], q["type"].(string))
	}
	want := map[string][]string{
		"https://a.example/inbox":   {"Create", "Announce"},
		"https://b.example/c/inbox": {"Create", "Announce"},
	}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("queued = %v, want %v", got, want)
	}
}

func TestQueueRelayEvent_ActorsAndObjects(t *testing.T) {
	app := newTestApp(t)
	defer app.Cleanup()
	createSettingsWithKeys(t, app)
	setTimezone(t, app, "Europe/London")
	addRelayFollower(t, app, "https://a.example/a", "https://a.example/a/inbox", "")
	ev := createPublishedEvent(t, app, "Summer Gig", time.Now().Add(48*time.Hour))
	evID := testBase + "/ap/events/" + ev.Id

	if err := QueueRelayEvent(app, testBase, ev, "Create"); err != nil {
		t.Fatal(err)
	}
	q := queuedActivities(t, app)
	if len(q) != 2 {
		t.Fatalf("queue len = %d", len(q))
	}
	create, announce := q[0], q[1]
	if create["actor"] != testBase+"/ap/actor" {
		t.Errorf("Create actor = %v, want the instance actor (it owns the event)", create["actor"])
	}
	if obj, _ := create["object"].(map[string]any); obj["type"] != "Event" || obj["id"] != evID {
		t.Errorf("Create object = %v", create["object"])
	}
	if announce["actor"] != testBase+"/ap/relay" || announce["object"] != evID {
		t.Errorf("Announce = actor %v object %v, want relay announcing %s", announce["actor"], announce["object"], evID)
	}
}

func TestQueueRelayEvent_UpdateIsEventOnly(t *testing.T) {
	app := newTestApp(t)
	defer app.Cleanup()
	createSettingsWithKeys(t, app)
	setTimezone(t, app, "Europe/London")
	addRelayFollower(t, app, "https://a.example/a", "https://a.example/a/inbox", "")
	ev := createPublishedEvent(t, app, "Summer Gig", time.Now().Add(48*time.Hour))

	if err := QueueRelayEvent(app, testBase, ev, "Update"); err != nil {
		t.Fatal(err)
	}
	q := queuedActivities(t, app)
	if len(q) != 1 || q[0]["type"] != "Update" {
		t.Fatalf("queue = %v, want a single Update (already announced)", q)
	}
}

func TestQueueRelayEvent_DeleteCarriesOnlyTheID(t *testing.T) {
	app := newTestApp(t)
	defer app.Cleanup()
	createSettingsWithKeys(t, app)
	setTimezone(t, app, "Europe/London")
	addRelayFollower(t, app, "https://a.example/a", "https://a.example/a/inbox", "")
	ev := createPublishedEvent(t, app, "Secret Title", time.Now().Add(48*time.Hour))

	if err := QueueRelayEvent(app, testBase, ev, "Delete"); err != nil {
		t.Fatal(err)
	}
	q := queuedActivities(t, app)
	if len(q) != 1 || q[0]["type"] != "Delete" || q[0]["object"] != testBase+"/ap/events/"+ev.Id {
		t.Fatalf("queue = %v", q)
	}
}

func TestQueueRelayEvent_NoFollowersQueuesNothing(t *testing.T) {
	app := newTestApp(t)
	defer app.Cleanup()
	createSettingsWithKeys(t, app)
	setTimezone(t, app, "Europe/London")
	ev := createPublishedEvent(t, app, "Summer Gig", time.Now().Add(48*time.Hour))

	if err := QueueRelayEvent(app, testBase, ev, "Create"); err != nil {
		t.Fatal(err)
	}
	if n, _ := app.CountRecords("ap_delivery_queue"); n != 0 {
		t.Errorf("queue = %d, want 0", n)
	}
}

func TestFetchPublicKey_AcceptsPKCS1KeysLikeMobilizonPublishes(t *testing.T) {
	app := newTestApp(t)
	defer app.Cleanup()
	createSettingsWithKeys(t, app)

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	pkcs1 := string(pem.EncodeToMemory(&pem.Block{Type: "RSA PUBLIC KEY", Bytes: x509.MarshalPKCS1PublicKey(&priv.PublicKey)}))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"publicKey": map[string]any{"publicKeyPem": pkcs1}})
	}))
	defer srv.Close()

	got, err := fetchPublicKey(app, testBase, srv.URL+"/relay#main-key")
	if err != nil {
		t.Fatalf("fetchPublicKey: %v", err)
	}
	if got.N.Cmp(priv.N) != 0 {
		t.Error("parsed key differs from the published one")
	}
}

func TestVerifyHTTPSignature_HostHeaderComesFromRequestHost(t *testing.T) {
	app := newTestApp(t)
	defer app.Cleanup()
	settings := createSettingsWithKeys(t, app)
	priv, err := ParsePrivateKey(settings.GetString("ap_private_key"))
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"publicKey": map[string]any{"publicKeyPem": settings.GetString("ap_public_key")}})
	}))
	defer srv.Close()

	body := []byte(`{"type":"Follow"}`)
	req := httptest.NewRequest("POST", "https://example.com/ap/relay/inbox", strings.NewReader(string(body)))
	if err := signRequest(req, priv, srv.URL+"#main-key", body); err != nil {
		t.Fatal(err)
	}
	// net/http never puts Host in r.Header on the server side; only r.Host carries it.
	req.Header.Del("Host")

	if err := verifyHTTPSignature(app, testBase, req, body); err != nil {
		t.Fatalf("verifyHTTPSignature: %v", err)
	}
}

func TestWebfinger_ResolvesRelay(t *testing.T) {
	b, err := HandleWebfinger("acct:relay@example.com", testBase)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), testBase+"/ap/relay") {
		t.Errorf("webfinger = %s", b)
	}
}
