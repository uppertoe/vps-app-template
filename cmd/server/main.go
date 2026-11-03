// Command server runs the app.
//
//	server               serve on LISTEN_ADDR
//	server -healthcheck  probe the running server (the container healthcheck)
//	server -routes       print every route and its access level
//	server -caddy        print the Caddy snippet generated from the routes
//	server -connections  print design-docs/connections.md from app.yaml
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/uppertoe/vps-app-kit/router"
	"github.com/uppertoe/vps-app-kit/serve"
	"github.com/uppertoe/vps-app-kit/sqlitex"

	app "github.com/uppertoe/vps-app-template"
	"github.com/uppertoe/vps-app-template/brand"
	"github.com/uppertoe/vps-app-template/internal/config"
	"github.com/uppertoe/vps-app-template/internal/store"
	"github.com/uppertoe/vps-app-template/internal/web"
)

func main() {
	healthcheck := flag.Bool("healthcheck", false, "probe the running server and exit")
	routes := flag.Bool("routes", false, "print the routes and exit")
	caddy := flag.Bool("caddy", false, "print the generated Caddy snippet and exit")
	connections := flag.Bool("connections", false, "print the connections page and exit")
	flag.Parse()

	if *healthcheck {
		serve.Healthcheck(envOr("LISTEN_ADDR", ":8080"))
	}
	m, err := app.Manifest()
	if err != nil {
		fatal(err)
	}
	if *connections {
		fmt.Print(m.ConnectionsPage())
		return
	}
	b, err := brand.Load()
	if err != nil {
		fatal(err)
	}
	site := web.Site{Title: m.Title, Organisation: b.Organisation, ContactEmail: b.ContactEmail, HealthUnattested: app.HealthUnattested(m)}

	if *routes || *caddy {
		// The route list needs no database or settings.
		srv, err := web.New(config.Config{PublicURL: "https://generate.invalid"}, nil, nil, site)
		if err != nil {
			fatal(err)
		}
		rt := srv.Routes()
		if *routes {
			fmt.Print(rt.Table())
			return
		}
		out, err := rt.Caddy(router.Site{Name: m.Name, Host: m.SiteHost(), Port: m.Port})
		if err != nil {
			fatal(err)
		}
		fmt.Print(out)
		return
	}

	log := serve.Logger()
	cfg, err := config.Load()
	if err != nil {
		fatal(err)
	}
	site.AuthURL = cfg.AuthURL
	if site.HealthUnattested {
		log.Warn("this app holds health information and no attestation is recorded; every page shows the prototype banner", "attestation", m.HealthAttestation)
	}
	ctx := context.Background()
	st, err := store.Open(ctx, cfg.SQLitePath)
	if err != nil {
		fatal(err)
	}
	defer st.Close()

	snapCtx, stopSnapshots := context.WithCancel(ctx)
	defer stopSnapshots()
	go snapshotLoop(snapCtx, log, st.Path(), cfg.SnapshotPath)

	srv, err := web.New(cfg, st, log, site)
	if err != nil {
		fatal(err)
	}
	if err := serve.Run(log, cfg.ListenAddr, srv.Handler()); err != nil {
		fatal(err)
	}
}

// snapshotLoop writes the hourly copy the backup reads.
func snapshotLoop(ctx context.Context, log *slog.Logger, dbPath, dest string) {
	sqlitex.SnapshotEvery(ctx, dbPath, dest, time.Hour, func(err error) {
		log.Error("snapshot failed", "err", err)
	})
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "starterapp:", err)
	os.Exit(1)
}
