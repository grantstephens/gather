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
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

func slugPlace(t *testing.T, app *tests.TestApp, name, city, slug, status string) string {
	t.Helper()
	p := createPlace(t, app, name, "1 High St", city, 56.7, -3.7, status)
	p.Set("slug", slug)
	if err := app.Save(p); err != nil {
		t.Fatal(err)
	}
	return p.Id
}

func actorDoc(t *testing.T, b []byte) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	return m
}

func TestSubjectActorJSON_PlaceIsAGroupWithItsOwnKey(t *testing.T) {
	app := newTestApp(t)
	defer app.Cleanup()
	createSettingsWithKeys(t, app)
	id := slugPlace(t, app, "Pitlochry Festival Theatre", "Pitlochry", "pft", "approved")

	b, err := SubjectActorJSON(app, testBase, "place", id)
	if err != nil {
		t.Fatalf("SubjectActorJSON: %v", err)
	}
	a := actorDoc(t, b)
	wantID := testBase + "/ap/actor/place/" + id
	if a["type"] != "Group" || a["id"] != wantID || a["name"] != "Pitlochry Festival Theatre" {
		t.Errorf("type/id/name = %v / %v / %v", a["type"], a["id"], a["name"])
	}
	if a["preferredUsername"] != "pft" || a["inbox"] != wantID+"/inbox" {
		t.Errorf("username/inbox = %v / %v", a["preferredUsername"], a["inbox"])
	}
	key := a["publicKey"].(map[string]any)
	if key["id"] != wantID+"#main-key" || key["owner"] != wantID {
		t.Errorf("publicKey = %v", key)
	}
}

func TestSubjectActorJSON_TownIsAGroupNamedAfterTheCity(t *testing.T) {
	app := newTestApp(t)
	defer app.Cleanup()
	createSettingsWithKeys(t, app)
	slugPlace(t, app, "Theatre", "Pitlochry", "pft", "approved")

	b, err := SubjectActorJSON(app, testBase, "town", "pitlochry")
	if err != nil {
		t.Fatal(err)
	}
	a := actorDoc(t, b)
	if a["type"] != "Group" || a["name"] != "Pitlochry" || a["preferredUsername"] != "town-pitlochry" {
		t.Errorf("actor = %v", a)
	}
}

func TestSubjectActorJSON_KeyIsGeneratedOnceAndDiffersFromInstanceKey(t *testing.T) {
	app := newTestApp(t)
	defer app.Cleanup()
	settings := createSettingsWithKeys(t, app)
	id := slugPlace(t, app, "Theatre", "Pitlochry", "pft", "approved")

	pem := func() string {
		b, err := SubjectActorJSON(app, testBase, "place", id)
		if err != nil {
			t.Fatal(err)
		}
		return actorDoc(t, b)["publicKey"].(map[string]any)["publicKeyPem"].(string)
	}
	first, second := pem(), pem()
	if first != second {
		t.Error("key changed between requests")
	}
	if first == settings.GetString("ap_public_key") {
		t.Error("place reused the instance key")
	}
	if n, _ := app.CountRecords("ap_actors"); n != 1 {
		t.Errorf("ap_actors = %d, want 1", n)
	}
}

func TestSubjectActorJSON_ConcurrentFirstHitsCreateOneKey(t *testing.T) {
	app := newTestApp(t)
	defer app.Cleanup()
	createSettingsWithKeys(t, app)
	id := slugPlace(t, app, "Theatre", "Pitlochry", "pft", "approved")

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := SubjectActorJSON(app, testBase, "place", id); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if n, _ := app.CountRecords("ap_actors"); n != 1 {
		t.Errorf("ap_actors = %d, want 1", n)
	}
}

func TestSubjectActorJSON_PendingOrMissingIsNotFound(t *testing.T) {
	app := newTestApp(t)
	defer app.Cleanup()
	createSettingsWithKeys(t, app)
	pending := slugPlace(t, app, "Shady Bar", "Dunkeld", "shady", "pending")

	cases := [][2]string{{"place", pending}, {"place", "nope"}, {"town", "dunkeld"}, {"town", "nowhere"}, {"tag", "x"}}
	for _, c := range cases {
		if _, err := SubjectActorJSON(app, testBase, c[0], c[1]); !errors.Is(err, ErrSubjectNotFound) {
			t.Errorf("%v: err = %v, want ErrSubjectNotFound", c, err)
		}
	}
	if n, _ := app.CountRecords("ap_actors"); n != 0 {
		t.Errorf("keys minted for unresolvable subjects: %d", n)
	}
}

func subjectFollow(t *testing.T, app *tests.TestApp, kind, ref, follower string) error {
	t.Helper()
	body := fmt.Sprintf(`{"type":"Follow","id":"https://remote.example/f/%d","actor":%q,"object":%q}`,
		time.Now().UnixNano(), follower, subjectActorID(testBase, kind, ref))
	return HandleSubjectInbox(app, testBase, kind, ref, relayInboxRequest(body))
}

func TestSubjectInbox_FollowIsStoredAgainstTheSubjectAndAcceptedByIt(t *testing.T) {
	app := newTestApp(t)
	defer app.Cleanup()
	createSettingsWithKeys(t, app)
	id := slugPlace(t, app, "Theatre", "Pitlochry", "pft", "approved")
	inboxSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(202) }))
	defer inboxSrv.Close()
	actorSrv := mockActorServer(t, inboxSrv.URL)
	defer actorSrv.Close()

	if err := subjectFollow(t, app, "place", id, actorSrv.URL); err != nil {
		t.Fatal(err)
	}
	time.Sleep(200 * time.Millisecond)

	f, err := app.FindFirstRecordByFilter("ap_subject_followers", "subject = {:s}", map[string]any{"s": "place:" + id})
	if err != nil || f.GetString("actor_url") != actorSrv.URL || f.GetString("inbox_url") != inboxSrv.URL {
		t.Fatalf("follower = %v / %v", f, err)
	}
	if n, _ := app.CountRecords("ap_followers"); n != 0 {
		t.Errorf("subject follow leaked into ap_followers")
	}
	q := queuedActivities(t, app)
	if len(q) != 1 || q[0]["type"] != "Accept" || q[0]["actor"] != subjectActorID(testBase, "place", id) {
		t.Fatalf("queue = %v, want one Accept from the place actor", q)
	}
}

func TestSubjectInbox_DuplicateFollowKeepsOneRow(t *testing.T) {
	app := newTestApp(t)
	defer app.Cleanup()
	createSettingsWithKeys(t, app)
	id := slugPlace(t, app, "Theatre", "Pitlochry", "pft", "approved")
	inboxSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(202) }))
	defer inboxSrv.Close()
	actorSrv := mockActorServer(t, inboxSrv.URL)
	defer actorSrv.Close()

	for i := 0; i < 3; i++ {
		if err := subjectFollow(t, app, "place", id, actorSrv.URL); err != nil {
			t.Fatal(err)
		}
	}
	time.Sleep(400 * time.Millisecond)

	if n, _ := app.CountRecords("ap_subject_followers"); n != 1 {
		t.Errorf("followers = %d, want 1", n)
	}
}

func TestSubjectInbox_UnknownSubjectIsNotFound(t *testing.T) {
	app := newTestApp(t)
	defer app.Cleanup()
	createSettingsWithKeys(t, app)

	err := subjectFollow(t, app, "place", "nope", "https://remote.example/a")
	if !errors.Is(err, ErrSubjectNotFound) {
		t.Errorf("err = %v, want ErrSubjectNotFound", err)
	}
}

func TestSubjectInbox_UndoOnlyRemovesThatSubjectsFollow(t *testing.T) {
	app := newTestApp(t)
	defer app.Cleanup()
	createSettingsWithKeys(t, app)
	a := slugPlace(t, app, "Theatre", "Pitlochry", "pft", "approved")
	b := slugPlace(t, app, "Cinema", "Pitlochry", "cinema", "approved")
	col, _ := app.FindCollectionByNameOrId("ap_subject_followers")
	for _, s := range []string{"place:" + a, "place:" + b} {
		r := core.NewRecord(col)
		r.Set("subject", s)
		r.Set("actor_url", "https://remote.example/a")
		r.Set("inbox_url", "https://remote.example/a/inbox")
		if err := app.Save(r); err != nil {
			t.Fatal(err)
		}
	}

	body := `{"type":"Undo","actor":"https://remote.example/a","object":{"type":"Follow","actor":"https://remote.example/a"}}`
	if err := HandleSubjectInbox(app, testBase, "place", a, relayInboxRequest(body)); err != nil {
		t.Fatal(err)
	}
	time.Sleep(200 * time.Millisecond)

	left, _ := app.FindRecordsByFilter("ap_subject_followers", "id != ''", "", 0, 0)
	if len(left) != 1 || left[0].GetString("subject") != "place:"+b {
		t.Errorf("remaining followers = %v", left)
	}
}

func TestSubjectInbox_RejectsBadSignature(t *testing.T) {
	app := newTestApp(t)
	defer app.Cleanup()
	createSettingsWithKeys(t, app)
	id := slugPlace(t, app, "Theatre", "Pitlochry", "pft", "approved")
	orig := signatureVerifier
	signatureVerifier = func(core.App, string, *http.Request, []byte) error { return errors.New("bad sig") }
	defer func() { signatureVerifier = orig }()

	if err := subjectFollow(t, app, "place", id, "https://remote.example/a"); !errors.Is(err, ErrUnauthorized) {
		t.Errorf("err = %v, want ErrUnauthorized", err)
	}
}

func TestProcessDeliveryQueue_SignsWithTheSendingSubjectsKey(t *testing.T) {
	app := newTestApp(t)
	defer app.Cleanup()
	createSettingsWithKeys(t, app)
	id := slugPlace(t, app, "Theatre", "Pitlochry", "pft", "approved")
	actorID := subjectActorID(testBase, "place", id)
	b, err := SubjectActorJSON(app, testBase, "place", id)
	if err != nil {
		t.Fatal(err)
	}
	pubPEM := actorDoc(t, b)["publicKey"].(map[string]any)["publicKeyPem"].(string)

	var got *http.Request
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r
		w.WriteHeader(202)
	}))
	defer srv.Close()
	if err := QueueDelivery(app, Activity{Context: "x", Type: "Announce", ID: actorID + "/a/1", Actor: actorID, Object: "o"}, srv.URL+"/inbox"); err != nil {
		t.Fatal(err)
	}

	time.Sleep(1100 * time.Millisecond) // the queue compares next_retry at second resolution
	ProcessDeliveryQueue(app)

	if got == nil {
		t.Fatal("nothing delivered")
	}
	sig := got.Header.Get("Signature")
	if !strings.Contains(sig, `keyId="`+actorID+`#main-key"`) {
		t.Fatalf("Signature = %q", sig)
	}
	// the signature must verify against the place's published key, not the instance key
	block, _ := pem.Decode([]byte(pubPEM))
	pk, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	signed := fmt.Sprintf("(request-target): post /inbox\nhost: %s\ndate: %s\ndigest: %s", got.Host, got.Header.Get("Date"), got.Header.Get("Digest"))
	raw, _ := base64.StdEncoding.DecodeString(parseSignatureHeader(sig)["signature"])
	h := sha256.Sum256([]byte(signed))
	if err := rsa.VerifyPKCS1v15(pk.(*rsa.PublicKey), crypto.SHA256, h[:], raw); err != nil {
		t.Errorf("signature does not verify with the place's public key: %v", err)
	}
}

func TestSubjectCollection_FollowersCountAndOutboxIsAValidEmptyCollection(t *testing.T) {
	app := newTestApp(t)
	defer app.Cleanup()
	createSettingsWithKeys(t, app)
	id := slugPlace(t, app, "Theatre", "Pitlochry", "pft", "approved")
	addSubjectFollower(t, app, "place:"+id, "https://a.example/u", "https://a.example/u/inbox", "")
	addSubjectFollower(t, app, "town:pitlochry", "https://b.example/u", "https://b.example/u/inbox", "")

	b, err := SubjectCollection(app, testBase, "place", id, "followers")
	if err != nil {
		t.Fatal(err)
	}
	c := actorDoc(t, b)
	if c["type"] != "OrderedCollection" || c["totalItems"] != float64(1) || c["id"] != subjectActorID(testBase, "place", id)+"/followers" {
		t.Errorf("followers = %v", c)
	}
	b, err = SubjectCollection(app, testBase, "place", id, "outbox")
	if err != nil {
		t.Fatal(err)
	}
	if c := actorDoc(t, b); c["type"] != "OrderedCollection" || c["totalItems"] != float64(0) {
		t.Errorf("outbox = %v", c)
	}
	if _, err := SubjectCollection(app, testBase, "place", "nope", "followers"); !errors.Is(err, ErrSubjectNotFound) {
		t.Errorf("err = %v", err)
	}
}

func TestWebfingerFor_ResolvesPlaceAndTownButNotPending(t *testing.T) {
	app := newTestApp(t)
	defer app.Cleanup()
	createSettingsWithKeys(t, app)
	id := slugPlace(t, app, "Theatre", "Pitlochry", "pft", "approved")
	slugPlace(t, app, "Shady Bar", "Dunkeld", "shady", "pending")

	b, err := HandleWebfingerFor(app, "acct:pft@example.com", testBase)
	if err != nil || !strings.Contains(string(b), testBase+"/ap/actor/place/"+id) {
		t.Errorf("place: %s / %v", b, err)
	}
	b, err = HandleWebfingerFor(app, "acct:town-pitlochry@example.com", testBase)
	if err != nil || !strings.Contains(string(b), testBase+"/ap/actor/town/pitlochry") {
		t.Errorf("town: %s / %v", b, err)
	}
	for _, name := range []string{"shady", "town-dunkeld", "ghost"} {
		if _, err := HandleWebfingerFor(app, "acct:"+name+"@example.com", testBase); err == nil {
			t.Errorf("%s resolved, want not found", name)
		}
	}
	// instance + relay still work
	for _, name := range []string{"events", "relay"} {
		if _, err := HandleWebfingerFor(app, "acct:"+name+"@example.com", testBase); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
}
