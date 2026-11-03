// Package web holds the app's pages. Routes are registered in one place,
// Routes below, each with its access level; the Caddy snippet in deploy/ is
// generated from the same list (make generate).
package web

import (
	"embed"
	"io/fs"
	"log/slog"
	"net/http"

	"github.com/uppertoe/vps-app-kit/access"
	"github.com/uppertoe/vps-app-kit/assets"
	"github.com/uppertoe/vps-app-kit/mail"
	"github.com/uppertoe/vps-app-kit/router"

	"github.com/uppertoe/vps-app-template/brand"
	"github.com/uppertoe/vps-app-template/internal/config"
	"github.com/uppertoe/vps-app-template/internal/store"
)

//go:embed static all:templates
var embedded embed.FS

// AdminGroup is the gateway group whose members see the admin pages. An
// administrator adds people to it on the gateway's Groups page.
const AdminGroup = "starterapp_admin"

// Deps are what the server is built from. main.go fills every field.
type Deps struct {
	Config config.Config
	Store  *store.Store
	Log    *slog.Logger
	// HTTP is the only client for outside services. It reaches only the
	// hosts listed under outbound in app.yaml (see docs/RECIPES.md).
	HTTP *http.Client
	// Mail sends email: through SES, to the log, or nowhere, as
	// EMAIL_BACKEND says.
	Mail mail.Sender
	Site Site
}

// Server holds what the handlers share.
type Server struct {
	cfg    config.Config
	store  *store.Store
	log    *slog.Logger
	http   *http.Client
	mail   mail.Sender
	static *assets.Set
	brand  *assets.Set
	pages  *pages
	site   Site
}

// Site is what every page shows about the app.
type Site struct {
	Title            string
	Organisation     string
	ContactEmail     string
	HealthUnattested bool
	AuthURL          string
	AdminGroup       string
	// DataClass is app.yaml's data_class: none, personal or health.
	DataClass string
}

// New builds the server. Store may be nil when only the route list is
// needed (make generate), since no handler runs then.
func New(d Deps) (*Server, error) {
	staticFS, _ := fs.Sub(embedded, "static")
	static, err := assets.New(staticFS, "/static/")
	if err != nil {
		return nil, err
	}
	brandSet, err := assets.New(brand.Files(), "/brand/")
	if err != nil {
		return nil, err
	}
	if d.Log == nil {
		d.Log = slog.Default()
	}
	d.Site.AdminGroup = AdminGroup
	s := &Server{cfg: d.Config, store: d.Store, log: d.Log, http: d.HTTP, mail: d.Mail,
		static: static, brand: brandSet, site: d.Site}
	s.pages, err = parsePages(s)
	if err != nil {
		return nil, err
	}
	return s, nil
}

// Routes registers every route. Read it as the app's access policy: each
// line says who may reach a path. Add a route here and run make generate.
func (s *Server) Routes() *router.Router {
	rt := router.New(router.Options{PublicOrigin: s.cfg.PublicURL, Deny: s.deny})

	rt.Handle("GET /static/", access.Public("stylesheets, scripts and fonts"), s.static.Handler())
	rt.Handle("GET /brand/", access.Public("logo and icons"), s.brand.Handler())
	rt.HandleFunc("GET /{$}", access.Public("the landing page"), s.home)
	rt.HandleFunc("GET /privacy", access.Public("the privacy policy"), s.privacy)

	rt.HandleFunc("GET /notes", access.SignedIn(), s.notes)
	rt.HandleFunc("POST /notes", access.SignedIn(), s.addNote)

	rt.HandleFunc("GET /admin", access.Group(AdminGroup), s.admin)

	return rt
}

// Handler is Routes as an http.Handler.
func (s *Server) Handler() http.Handler { return s.Routes() }
