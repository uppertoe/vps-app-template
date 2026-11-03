package web

import (
	"bytes"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/uppertoe/vps-app-kit/gateway"
	"github.com/uppertoe/vps-app-kit/secure"
)

// pages holds one parsed template set per page. Files starting with "_" are
// partials: parsed into every page, and renderable on their own for htmx.
type pages struct {
	byName map[string]*template.Template
}

func parsePages(s *Server) (*pages, error) {
	funcs := template.FuncMap{
		"asset": s.static.URL,
		"brand": s.brand.URL,
		"date":  func(t time.Time) string { return t.In(melbourne).Format("2 January 2006, 3:04 pm") },
	}
	all, err := fs.Glob(embedded, "templates/*.html")
	if err != nil {
		return nil, err
	}
	var partials []string
	for _, f := range all {
		if strings.HasPrefix(path.Base(f), "_") || path.Base(f) == "layout.html" {
			partials = append(partials, f)
		}
	}
	p := &pages{byName: map[string]*template.Template{}}
	for _, f := range all {
		base := path.Base(f)
		if strings.HasPrefix(base, "_") || base == "layout.html" {
			continue
		}
		t, err := template.New(base).Funcs(funcs).ParseFS(embedded, append([]string{f}, partials...)...)
		if err != nil {
			return nil, fmt.Errorf("template %s: %w", base, err)
		}
		p.byName[base] = t
	}
	partialSet, err := template.New("partials").Funcs(funcs).ParseFS(embedded, partials...)
	if err != nil {
		return nil, err
	}
	p.byName["_partials"] = partialSet
	return p, nil
}

var melbourne = func() *time.Location {
	l, err := time.LoadLocation("Australia/Melbourne")
	if err != nil {
		return time.UTC
	}
	return l
}()

// view is what every page receives.
type view struct {
	Site     Site
	Identity gateway.Identity
	SignedIn bool
	Nonce    string
	Title    string
	Data     any
}

func (s *Server) newView(r *http.Request, title string, data any) view {
	id, ok := gateway.From(r.Context())
	return view{Site: s.site, Identity: id, SignedIn: ok, Nonce: secure.Nonce(r.Context()), Title: title, Data: data}
}

// render writes a whole page through layout.html.
func (s *Server) render(w http.ResponseWriter, r *http.Request, status int, page, title string, data any) {
	t, ok := s.pages.byName[page]
	if !ok {
		s.fail(w, r, fmt.Errorf("no page %s", page))
		return
	}
	var buf bytes.Buffer
	if err := t.ExecuteTemplate(&buf, "layout", s.newView(r, title, data)); err != nil {
		s.fail(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = buf.WriteTo(w)
}

// renderPartial writes one partial, for an htmx response.
func (s *Server) renderPartial(w http.ResponseWriter, r *http.Request, name string, data any) {
	var buf bytes.Buffer
	if err := s.pages.byName["_partials"].ExecuteTemplate(&buf, name, s.newView(r, "", data)); err != nil {
		s.fail(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = buf.WriteTo(w)
}

// fail logs the error and shows a plain error page. The error itself is
// never shown to the reader.
func (s *Server) fail(w http.ResponseWriter, r *http.Request, err error) {
	if s.log != nil {
		s.log.Error("handler failed", "route", r.Pattern, "err", err)
	}
	http.Error(w, "Something went wrong on our side. Please try again in a moment.", http.StatusInternalServerError)
}

type deniedView struct {
	Heading string
	Body    string
}

// deny is the router's refusal page.
func (s *Server) deny(w http.ResponseWriter, r *http.Request, status int) {
	v := deniedView{
		Heading: "You do not have access to this page",
		Body:    "If you need access, you can contact " + s.site.ContactEmail + ".",
	}
	if status == http.StatusUnauthorized {
		v = deniedView{
			Heading: "Sign-in is required",
			Body:    "This page will be available only after you have signed in. Please return to the home page and sign in from there.",
		}
	}
	s.render(w, r, status, "error.html", v.Heading, v)
}
