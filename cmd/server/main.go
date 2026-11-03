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
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/uppertoe/vps-app-kit/egress"
	"github.com/uppertoe/vps-app-kit/mail"
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
	site := web.Site{Title: m.Title, Organisation: b.Organisation, ContactEmail: b.ContactEmail,
		HealthUnattested: app.HealthUnattested(m), DataClass: string(m.DataClass)}

	if *routes || *caddy {
		// The route list needs no database or settings.
		srv, err := web.New(web.Deps{Config: config.Config{PublicURL: "https://generate.invalid"}, Site: site})
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

	srv, err := web.New(web.Deps{
		Config: cfg,
		Store:  st,
		Log:    log,
		HTTP:   egress.New(m.Hosts()),
		Mail:   mailer(cfg, log),
		Site:   site,
	})
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

// mailer picks the email back end from EMAIL_BACKEND.
func mailer(cfg config.Config, log *slog.Logger) mail.Sender {
	switch cfg.EmailBackend {
	case "smtp":
		return mail.SMTP{Host: cfg.SMTPHost, Port: 587, Username: cfg.SMTPUser, Password: cfg.SMTPPass}
	case "log":
		return mail.Log{Logger: log}
	}
	return noMail{}
}

// noMail refuses to send, so a feature that sends email fails visibly when
// EMAIL_BACKEND is off rather than dropping the message.
type noMail struct{}

func (noMail) Send(context.Context, mail.Message) error {
	return errors.New("email is off (EMAIL_BACKEND=off)")
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
