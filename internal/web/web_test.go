package web

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/uppertoe/vps-app-kit/gateway"

	"github.com/uppertoe/vps-app-template/internal/config"
	"github.com/uppertoe/vps-app-template/internal/store"
)

const origin = "https://starter.example.org"

func newServer(t *testing.T, site Site) http.Handler {
	t.Helper()
	st, err := store.Open(context.Background(), filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	if site.Title == "" {
		site = Site{Title: "Starter app", Organisation: "Org", ContactEmail: "help@example.org", AuthURL: "https://auth.example.org", DataClass: "none"}
	}
	srv, err := New(Deps{Config: config.Config{PublicURL: origin}, Store: st, Log: slog.New(slog.NewTextHandler(os.Stderr, nil)), Site: site})
	if err != nil {
		t.Fatal(err)
	}
	return srv.Handler()
}

func get(h http.Handler, path, email, groups string) *httptest.ResponseRecorder {
	r := httptest.NewRequest("GET", path, nil)
	if email != "" {
		r.Header.Set(gateway.HeaderEmail, email)
		r.Header.Set(gateway.HeaderGroups, groups)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func TestPagesRender(t *testing.T) {
	h := newServer(t, Site{})
	for _, c := range []struct {
		path, email, groups string
		status              int
		contains            string
	}{
		{"/", "", "", 200, "Sign in to see the shared notes"},
		{"/privacy", "", "", 200, "holds no personal information"},
		{"/notes", "ann@example.org", "user", 200, "No notes have been saved yet."},
		{"/notes", "", "", 401, "Sign-in is required"},
		{"/admin", "ann@example.org", "user", 403, "You do not have access to this page"},
		{"/admin", "ann@example.org", "user,starterapp_admin", 200, "0 notes have been saved."},
	} {
		w := get(h, c.path, c.email, c.groups)
		if w.Code != c.status || !strings.Contains(w.Body.String(), c.contains) {
			t.Errorf("%s as %q: %d, want %d containing %q", c.path, c.groups, w.Code, c.status, c.contains)
		}
	}
}

func TestAdminLinkOnlyForAdmins(t *testing.T) {
	h := newServer(t, Site{})
	if strings.Contains(get(h, "/notes", "a@example.org", "user").Body.String(), `href="/admin"`) {
		t.Error("a user who is not in the group sees the admin link")
	}
	if !strings.Contains(get(h, "/notes", "a@example.org", "user,starterapp_admin").Body.String(), `href="/admin"`) {
		t.Error("a group member does not see the admin link")
	}
}

func TestAddNote(t *testing.T) {
	h := newServer(t, Site{})
	post := func(body string, htmx bool) *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", "/notes", strings.NewReader(url.Values{"body": {body}}.Encode()))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		r.Header.Set("Origin", origin)
		r.Header.Set(gateway.HeaderEmail, "ann@example.org")
		r.Header.Set(gateway.HeaderGroups, "user")
		if htmx {
			r.Header.Set("HX-Request", "true")
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	if w := post("<script>alert(1)</script>", true); w.Code != 200 || strings.Contains(w.Body.String(), "<script>alert") {
		t.Fatalf("htmx add: %d %s", w.Code, w.Body.String())
	}
	if w := post("plain form", false); w.Code != 303 {
		t.Fatalf("form add: %d", w.Code)
	}
	if w := post("", true); !strings.Contains(w.Body.String(), "between 1 and 2,000 characters") {
		t.Fatalf("empty note: %s", w.Body.String())
	}
	// Without JavaScript, a refused note shows the page again with the
	// error and the typed text, never a redirect that hides the refusal.
	long := strings.Repeat("x", 2001)
	w := post(long, false)
	if w.Code != http.StatusUnprocessableEntity || !strings.Contains(w.Body.String(), "between 1 and 2,000 characters") || !strings.Contains(w.Body.String(), long) {
		t.Fatalf("refused form post: %d", w.Code)
	}
}

func TestHealthBanner(t *testing.T) {
	banner := "It is not approved to hold real patient information"
	if strings.Contains(get(newServer(t, Site{}), "/", "", "").Body.String(), banner) {
		t.Fatal("banner shown on an app that is not a health app")
	}
	h := newServer(t, Site{Title: "x", ContactEmail: "h@example.org", HealthUnattested: true})
	for _, p := range []string{"/", "/notes"} {
		if !strings.Contains(get(h, p, "a@example.org", "user").Body.String(), banner) {
			t.Errorf("%s lacks the banner", p)
		}
	}
}

// Every URL a page refers to must be served, with the immutable cache.
func TestAssetURLsResolve(t *testing.T) {
	h := newServer(t, Site{})
	page := get(h, "/", "", "").Body.String()
	urls := regexp.MustCompile(`(?:href|src)="(/(?:static|brand)/[^"]+)"`).FindAllStringSubmatch(page, -1)
	if len(urls) < 6 {
		t.Fatalf("found only %d asset URLs", len(urls))
	}
	for _, m := range urls {
		u := strings.ReplaceAll(m[1], "&amp;", "&")
		w := get(h, u, "", "")
		body, _ := io.ReadAll(w.Body)
		if w.Code != 200 || len(body) == 0 || !strings.Contains(w.Header().Get("Cache-Control"), "immutable") {
			t.Errorf("%s: %d, %d bytes, %q", u, w.Code, len(body), w.Header().Get("Cache-Control"))
		}
	}
}
