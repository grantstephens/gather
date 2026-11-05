package newsletter

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"gather/internal/api"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

func factory(t testing.TB) *tests.TestApp { return api.NewTestApp(t) }

// prep registers the routes and optionally mutates the app before the request.
func prep(mutate func(t testing.TB, app *tests.TestApp)) func(testing.TB, *tests.TestApp, *core.ServeEvent) {
	return func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
		Register(e, testBase)
		if mutate != nil {
			mutate(t, app)
		}
	}
}

func enabled(t testing.TB, app *tests.TestApp) { enable(t, app) }

func jsonBody(s string) *strings.Reader { return strings.NewReader(s) }

func TestRoutesConfig(t *testing.T) {
	(&tests.ApiScenario{
		Name: "disabled without mail", Method: http.MethodGet, URL: "/api/newsletter/config",
		TestAppFactory: factory, ExpectedStatus: 200,
		ExpectedContent: []string{`"enabled":false`, `"privacy_url":null`},
		BeforeTestFunc: prep(func(t testing.TB, app *tests.TestApp) {
			setNewsletterSettings(t, app, true, nil)
			setMail(app, false, "")
		}),
	}).Test(t)

	(&tests.ApiScenario{
		Name: "enabled with privacy page", Method: http.MethodGet, URL: "/api/newsletter/config",
		TestAppFactory: factory, ExpectedStatus: 200,
		ExpectedContent: []string{`"enabled":true`, `"privacy_url":"/privacy"`},
		BeforeTestFunc: prep(func(t testing.TB, app *tests.TestApp) {
			enable(t, app)
			pages, _ := app.FindCollectionByNameOrId("pages")
			p := core.NewRecord(pages)
			p.Set("title", "Privacy")
			p.Set("slug", "privacy")
			if err := app.Save(p); err != nil {
				t.Fatal(err)
			}
		}),
	}).Test(t)
}

func TestRoutesSubscribe(t *testing.T) {
	(&tests.ApiScenario{
		Name: "invalid email", Method: http.MethodPost, URL: "/api/newsletter/subscribe",
		Body: jsonBody(`{"email":"nope"}`), TestAppFactory: factory, ExpectedStatus: 400,
		ExpectedContent: []string{"valid email"}, BeforeTestFunc: prep(enabled),
	}).Test(t)

	(&tests.ApiScenario{
		Name: "unavailable when disabled", Method: http.MethodPost, URL: "/api/newsletter/subscribe",
		Body: jsonBody(`{"email":"a@example.com"}`), TestAppFactory: factory, ExpectedStatus: 503,
		ExpectedContent: []string{"isn't available"},
		BeforeTestFunc: prep(func(t testing.TB, app *tests.TestApp) {
			setNewsletterSettings(t, app, true, nil)
			setMail(app, false, "")
		}),
	}).Test(t)

	(&tests.ApiScenario{
		Name: "creates a pending subscriber and sends the confirmation", Method: http.MethodPost, URL: "/api/newsletter/subscribe",
		Body: jsonBody(`{"email":"New@Example.com","website":""}`), TestAppFactory: factory, ExpectedStatus: 200,
		ExpectedContent: []string{"Check your inbox"}, Delay: 200 * time.Millisecond,
		BeforeTestFunc: prep(enabled),
		AfterTestFunc: func(t testing.TB, app *tests.TestApp, res *http.Response) {
			recs := allSubscribers(t, app)
			if len(recs) != 1 || recs[0].GetString("email") != "new@example.com" || recs[0].GetString("status") != "pending" {
				t.Errorf("want one pending lowercased subscriber, got %d", len(recs))
			}
			if n := app.TestMailer.TotalSend(); n != 1 {
				t.Errorf("confirmation mails = %d, want 1", n)
			}
		},
	}).Test(t)

	(&tests.ApiScenario{
		Name: "honeypot answers success but does nothing", Method: http.MethodPost, URL: "/api/newsletter/subscribe",
		Body: jsonBody(`{"email":"bot@example.com","website":"http://spam"}`), TestAppFactory: factory, ExpectedStatus: 200,
		ExpectedContent: []string{"Check your inbox"}, Delay: 100 * time.Millisecond,
		BeforeTestFunc: prep(enabled),
		AfterTestFunc: func(t testing.TB, app *tests.TestApp, res *http.Response) {
			if n := len(allSubscribers(t, app)); n != 0 {
				t.Errorf("honeypot created %d records", n)
			}
			if app.TestMailer.TotalSend() != 0 {
				t.Error("honeypot must not send mail")
			}
		},
	}).Test(t)

	(&tests.ApiScenario{
		Name: "already confirmed address gets the same answer", Method: http.MethodPost, URL: "/api/newsletter/subscribe",
		Body: jsonBody(`{"email":"done@example.com"}`), TestAppFactory: factory, ExpectedStatus: 200,
		ExpectedContent: []string{"Check your inbox"}, Delay: 100 * time.Millisecond,
		BeforeTestFunc: prep(func(t testing.TB, app *tests.TestApp) {
			enable(t, app)
			seedSubscriber(t, app, "done@example.com", "confirmed", time.Now())
		}),
		AfterTestFunc: func(t testing.TB, app *tests.TestApp, res *http.Response) {
			if app.TestMailer.TotalSend() != 0 {
				t.Error("no mail for an already confirmed address")
			}
		},
	}).Test(t)
}

func TestRoutesConfirmAndUnsubscribe(t *testing.T) {
	(&tests.ApiScenario{
		Name: "confirm with a valid token", Method: http.MethodPost, URL: "/api/newsletter/confirm?token=ct-p@example.com",
		TestAppFactory: factory, ExpectedStatus: 200, ExpectedContent: []string{`"confirmed"`},
		BeforeTestFunc: prep(func(t testing.TB, app *tests.TestApp) {
			seedSubscriber(t, app, "p@example.com", "pending", time.Now())
		}),
		AfterTestFunc: func(t testing.TB, app *tests.TestApp, res *http.Response) {
			if r := allSubscribers(t, app)[0]; r.GetString("status") != "confirmed" {
				t.Errorf("status = %q", r.GetString("status"))
			}
		},
	}).Test(t)

	(&tests.ApiScenario{
		Name: "confirm with a bad token", Method: http.MethodPost, URL: "/api/newsletter/confirm?token=nope",
		TestAppFactory: factory, ExpectedStatus: 400, ExpectedContent: []string{`"invalid"`}, BeforeTestFunc: prep(nil),
	}).Test(t)

	(&tests.ApiScenario{
		Name: "confirm must not be a GET (link scanners)", Method: http.MethodGet, URL: "/api/newsletter/confirm?token=ct-p@example.com",
		TestAppFactory: factory, ExpectedStatus: 404, NotExpectedContent: []string{"confirmed"},
		BeforeTestFunc: prep(func(t testing.TB, app *tests.TestApp) {
			seedSubscriber(t, app, "p@example.com", "pending", time.Now())
		}),
		AfterTestFunc: func(t testing.TB, app *tests.TestApp, res *http.Response) {
			if r := allSubscribers(t, app)[0]; r.GetString("status") != "pending" {
				t.Errorf("a GET must not confirm anyone, status = %q", r.GetString("status"))
			}
		},
	}).Test(t)

	(&tests.ApiScenario{
		Name: "unsubscribe deletes (form-encoded one-click body)", Method: http.MethodPost, URL: "/api/newsletter/unsubscribe?token=ut-u@example.com",
		Body: strings.NewReader("List-Unsubscribe=One-Click"), Headers: map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
		TestAppFactory: factory, ExpectedStatus: 200, ExpectedContent: []string{`"ok"`},
		BeforeTestFunc: prep(func(t testing.TB, app *tests.TestApp) {
			seedSubscriber(t, app, "u@example.com", "confirmed", time.Now())
		}),
		AfterTestFunc: func(t testing.TB, app *tests.TestApp, res *http.Response) {
			if n := len(allSubscribers(t, app)); n != 0 {
				t.Errorf("subscriber should be deleted, %d left", n)
			}
		},
	}).Test(t)

	(&tests.ApiScenario{
		Name: "unsubscribe with an unknown token still answers ok", Method: http.MethodPost, URL: "/api/newsletter/unsubscribe?token=nope",
		TestAppFactory: factory, ExpectedStatus: 200, ExpectedContent: []string{`"ok"`}, BeforeTestFunc: prep(nil),
	}).Test(t)
}

func TestRoutesPixel(t *testing.T) {
	(&tests.ApiScenario{
		Name: "pixel route serves a gif", Method: http.MethodGet, URL: "/newsletter/open/aaaaaaaaaaaaaaa.gif",
		TestAppFactory: factory, ExpectedStatus: 200, ExpectedContent: []string{"GIF89a"}, BeforeTestFunc: prep(nil),
	}).Test(t)
}

func TestRoutesAdmin(t *testing.T) {
	(&tests.ApiScenario{
		Name: "admin endpoint rejects guests", Method: http.MethodGet, URL: "/api/newsletter/admin",
		TestAppFactory: factory, ExpectedStatus: 401, ExpectedContent: []string{"Authentication required"}, BeforeTestFunc: prep(nil),
	}).Test(t)

	plain := map[string]string{}
	(&tests.ApiScenario{
		Name: "admin endpoint rejects plain users", Method: http.MethodGet, URL: "/api/newsletter/admin",
		Headers: plain, TestAppFactory: factory, ExpectedStatus: 403, ExpectedContent: []string{"required"},
		BeforeTestFunc: prep(func(t testing.TB, app *tests.TestApp) {
			plain["Authorization"] = api.CreateTestUser(t, app, "user@example.com", "password12345", "user").Token
		}),
	}).Test(t)

	adminHdr := map[string]string{}
	(&tests.ApiScenario{
		Name: "admin endpoint for admins", Method: http.MethodGet, URL: "/api/newsletter/admin",
		Headers: adminHdr, TestAppFactory: factory, ExpectedStatus: 200,
		ExpectedContent: []string{`"confirmed":1`, `"pending":2`, `"mail_configured":true`, `"toggle_on":true`, `"timezone"`},
		BeforeTestFunc: prep(func(t testing.TB, app *tests.TestApp) {
			enable(t, app)
			seedSubscriber(t, app, "a@example.com", "confirmed", time.Now())
			seedSubscriber(t, app, "b@example.com", "pending", time.Now())
			seedSubscriber(t, app, "c@example.com", "pending", time.Now())
			adminHdr["Authorization"] = api.CreateTestUser(t, app, "admin@example.com", "password12345", "admin").Token
		}),
	}).Test(t)

	testHdr := map[string]string{}
	(&tests.ApiScenario{
		Name: "send test goes to the caller only", Method: http.MethodPost, URL: "/api/newsletter/test",
		Headers: testHdr, TestAppFactory: factory, ExpectedStatus: 200, ExpectedContent: []string{`"sent"`},
		BeforeTestFunc: prep(func(t testing.TB, app *tests.TestApp) {
			enable(t, app)
			ev := mkEvent(t, app, time.Now().AddDate(0, 0, 2), time.Time{}, "published", "")
			mkPick(t, app, pickSpec{slug: "p", start: "2026-11-06", events: []string{ev}})
			seedSubscriber(t, app, "sub@example.com", "confirmed", time.Now())
			testHdr["Authorization"] = api.CreateTestUser(t, app, "admin2@example.com", "password12345", "admin").Token
		}),
		AfterTestFunc: func(t testing.TB, app *tests.TestApp, res *http.Response) {
			if app.TestMailer.TotalSend() != 1 || app.TestMailer.LastMessage().To[0].Address != "admin2@example.com" {
				t.Errorf("test mail must go only to the caller; sent %d", app.TestMailer.TotalSend())
			}
		},
	}).Test(t)
}

func TestRegisterCronAddsJobs(t *testing.T) {
	app := api.NewTestApp(t)
	defer app.Cleanup()
	RegisterCron(app, testBase)
	have := map[string]bool{}
	for _, j := range app.Cron().Jobs() {
		have[j.Id()] = true
	}
	for _, id := range []string{"newsletter-tick", "newsletter-purge"} {
		if !have[id] {
			t.Errorf("cron job %q not registered", id)
		}
	}
}
