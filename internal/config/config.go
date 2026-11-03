// Package config reads the app's settings from the environment. The app
// refuses to start if a required setting is missing, or still holds the
// CHANGE_ME placeholder copied from .env.example.
package config

import (
	"errors"
	"fmt"
	"net/mail"
	"net/url"
	"os"
	"strings"
)

// Config is every setting the app reads.
type Config struct {
	// ListenAddr is where the container listens. Caddy is the only client.
	ListenAddr string
	// PublicURL is the app's address as browsers see it. Writes from any
	// other origin are refused, and links in email are built from it.
	PublicURL string
	// AuthURL is the gateway's address, for the sign-out link.
	AuthURL string
	// SQLitePath is the live database; SnapshotPath is the copy the
	// backup reads.
	SQLitePath   string
	SnapshotPath string

	// EmailBackend is "off", "log" or "smtp".
	EmailBackend string
	EmailFrom    mail.Address
	SMTPHost     string
	SMTPUser     string
	SMTPPass     string

	// MembersURL and MembersToken reach the gateway's members endpoint, for
	// an app that lists groups under members in app.yaml.
	MembersURL   string
	MembersToken string
}

const placeholder = "CHANGE_ME"

// Load reads the environment.
func Load() (Config, error) {
	c := Config{
		ListenAddr:   env("LISTEN_ADDR", ":8080"),
		PublicURL:    strings.TrimRight(env("PUBLIC_URL", ""), "/"),
		AuthURL:      strings.TrimRight(env("AUTH_URL", ""), "/"),
		SQLitePath:   env("SQLITE_PATH", "/data/app.db"),
		SnapshotPath: env("SNAPSHOT_PATH", "/data/snapshot/app.db"),
		EmailBackend: env("EMAIL_BACKEND", "off"),
		SMTPHost:     env("SMTP_HOST", ""),
		SMTPUser:     env("SMTP_USER", ""),
		SMTPPass:     env("SMTP_PASS", ""),
		MembersURL:   env("GATEWAY_MEMBERS_URL", ""),
		MembersToken: env("GATEWAY_MEMBERS_TOKEN", ""),
	}
	var errs []error
	for _, k := range []string{"PUBLIC_URL", "AUTH_URL", "SMTP_HOST", "SMTP_USER", "SMTP_PASS", "EMAIL_FROM", "GATEWAY_MEMBERS_URL", "GATEWAY_MEMBERS_TOKEN"} {
		if strings.Contains(os.Getenv(k), placeholder) {
			errs = append(errs, fmt.Errorf("%s still holds the %s placeholder from .env.example", k, placeholder))
		}
	}
	u, err := url.Parse(c.PublicURL)
	local := err == nil && isLocalHost(u.Hostname())
	switch {
	case c.PublicURL == "" || err != nil || u.Host == "" || u.User != nil || (u.Path != "" && u.Path != "/") || u.RawQuery != "" || u.Fragment != "":
		errs = append(errs, errors.New("PUBLIC_URL must be the app's address alone, such as https://starter.example.org, with no path, query or credentials"))
	case u.Scheme != "https" && !(u.Scheme == "http" && local):
		errs = append(errs, errors.New("PUBLIC_URL must use https; plain http is allowed only for 127.0.0.1 or localhost while developing"))
	case c.AuthURL == "":
		// The gateway lives at auth.<the server's domain>.
		host := u.Hostname()
		if i := strings.Index(host, "."); i > 0 {
			c.AuthURL = u.Scheme + "://auth." + host[i+1:]
		}
	}
	switch c.EmailBackend {
	case "log":
		// The log back end prints whole messages, so on a server they would
		// reach the shipped logs.
		if !local {
			errs = append(errs, errors.New("EMAIL_BACKEND=log prints every message to the log and is allowed only while developing; use off or smtp"))
		}
	case "off":
	case "smtp":
		if c.SMTPHost == "" || c.SMTPUser == "" || c.SMTPPass == "" {
			errs = append(errs, errors.New("EMAIL_BACKEND=smtp needs SMTP_HOST, SMTP_USER and SMTP_PASS"))
		}
	default:
		errs = append(errs, fmt.Errorf("EMAIL_BACKEND must be off, log or smtp, not %q", c.EmailBackend))
	}
	if c.EmailBackend != "off" {
		a, err := mail.ParseAddress(env("EMAIL_FROM", ""))
		if err != nil {
			errs = append(errs, errors.New("EMAIL_FROM must be an address, such as Starter app <starter@example.org>"))
		} else {
			c.EmailFrom = *a
		}
	}
	return c, errors.Join(errs...)
}

func env(k, def string) string {
	if v, ok := os.LookupEnv(k); ok && strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	return def
}

// isLocalHost reports whether the app is running on the developer's machine.
func isLocalHost(h string) bool {
	return h == "localhost" || h == "127.0.0.1" || h == "::1"
}
