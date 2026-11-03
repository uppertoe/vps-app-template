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

// Server holds what the handlers share.
type Server struct {
	cfg    config.Config
	store  *store.Store
	log    *slog.Logger
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
}

// New builds the server. st may be nil when only the route list is needed
// (make generate), since no handler runs then.
func New(cfg config.Config, st *store.Store, log *slog.Logger, site Site) (*Server, error) {
	staticFS, _ := fs.Sub(embedded, "static")
	static, err := assets.New(staticFS, "/static/")
	if err != nil {
		return nil, err
	}
	brandSet, err := assets.New(brand.Files(), "/brand/")
	if err != nil {
		return nil, err
	}
	site.AdminGroup = AdminGroup
	s := &Server{cfg: cfg, store: st, log: log, static: static, brand: brandSet, site: site}
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

	rt.HandleFunc("GET /notes", access.SignedIn(), s.notes)
	rt.HandleFunc("POST /notes", access.SignedIn(), s.addNote)

	rt.HandleFunc("GET /admin", access.Group(AdminGroup), s.admin)

	return rt
}

// Handler is Routes as an http.Handler.
func (s *Server) Handler() http.Handler { return s.Routes() }
