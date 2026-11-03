# Recipes

Worked examples of the changes an app most often needs. Each one follows the
rules in `AGENTS.md`, and each one compiles as written. After any of them,
run `make test`, `make lint` and `make generate`, and commit what changed.

## A page for signed-in people

Register the route with its access level in `Routes`
(`internal/web/server.go`):

```go
rt.HandleFunc("GET /rota", access.SignedIn(), s.rota)
```

Write the handler in a file named after the area of the site, such as
`internal/web/rota.go`:

```go
package web

import (
	"net/http"

	"github.com/uppertoe/vps-app-kit/gateway"
)

func (s *Server) rota(w http.ResponseWriter, r *http.Request) {
	id, _ := gateway.From(r.Context()) // who is signed in
	s.render(w, r, http.StatusOK, "rota.html", "Rota", map[string]any{"Name": id.Display()})
}
```

Then add `internal/web/templates/rota.html`:

```html
{{define "content"}}
<h1>Rota</h1>
<p>You are signed in as {{.Data.Name}}.</p>
{{end}}
```

Run `make generate`: the new path appears in the Caddy snippet with the same
guard. Add the page to `PAGES` in `e2e/browser_check.py`, and add a test in
`internal/web/web_test.go` that requests it as someone who should be admitted
and as someone who should not.

## A page for one group, or for one domain

```go
rt.HandleFunc("GET /reports", access.Group("starterapp_reports"), s.reports)
rt.HandleFunc("GET /staff", access.Domain("rch.org.au"), s.staff)
rt.HandleFunc("GET /either", access.DomainOrGroup([]string{"rch.org.au"}, []string{"starterapp_reports"}), s.either)
```

An administrator adds people to a group on the sign-in service's Groups page.
Name a new group after the app, as `starterapp_reports` is, so it cannot be
confused with another app's group. To show something only to a group's
members within a page, test it in the template with
`{{if .Identity.InGroup "starterapp_reports"}}`.

## Pages under one path share one access level

Caddy guards by path prefix. A route with a wildcard, such as
`/cases/{id}`, guards everything under `/cases/`, so a coordinators-only
action cannot live at `/cases/{id}/select` while staff read
`/cases/{id}`; `make generate` stops with an error naming the two routes.
Give actions with a narrower level their own prefix:

```go
rt.HandleFunc("GET /cases/{id}", access.SignedIn(), s.showCase)
rt.HandleFunc("POST /selection/{id}", access.Group("starterapp_coordinators"), s.setSelection)
```

## A change to the database

Add a new numbered file; never edit one that has been merged.
`internal/store/migrations/0002_shifts.sql`:

```sql
CREATE TABLE shifts (
    id         INTEGER PRIMARY KEY,
    starts_on  TEXT NOT NULL, -- YYYY-MM-DD
    staff_name TEXT NOT NULL
);
CREATE INDEX shifts_starts_on ON shifts(starts_on);
```

Then a file for the concern, `internal/store/shifts.go`. Every value is a
parameter:

```go
package store

import "context"

// Shift is one rostered shift.
type Shift struct {
	ID        int64
	StartsOn  string
	StaffName string
}

// ShiftsFrom lists the shifts on or after a date, earliest first.
func (s *Store) ShiftsFrom(ctx context.Context, day string) ([]Shift, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, starts_on, staff_name FROM shifts WHERE starts_on >= ? ORDER BY starts_on`, day)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Shift
	for rows.Next() {
		var sh Shift
		if err := rows.Scan(&sh.ID, &sh.StartsOn, &sh.StaffName); err != nil {
			return nil, err
		}
		out = append(out, sh)
	}
	return out, rows.Err()
}
```

The store holds one connection, so read every row into a slice, as above,
before issuing another query.

## Calling an outside service

Declare the service in `app.yaml`, with what it is for and what it receives:

```yaml
outbound:
  - host: api.example.org
    purpose: Looks up drug interactions for a medication list
    data: The medication names, without any patient details
```

Then call it through the server's client, `s.http`, which refuses any host
not in `app.yaml`:

```go
package web

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type interaction struct {
	Drugs   []string `json:"drugs"`
	Summary string   `json:"summary"`
}

func (s *Server) interactions(ctx context.Context, names []string) ([]interaction, error) {
	// A slow service must not hold up the page.
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	u := "https://api.example.org/v1/interactions?" + url.Values{"drug": names}.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("interactions: status %d", resp.StatusCode)
	}
	var out []interaction
	err = json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&out)
	return out, err
}
```

An API key goes in the server's `.env`, never in code: add
`EXAMPLE_API_KEY=CHANGE_ME` to `deploy/apps/starterapp/.env.example`, read
it in `internal/config/config.go`, and send it as a header on the request.
Run `make generate` to update the connections page in the design docs.

If a library wants an `*http.Client`, give it `s.http`. A library that makes
its own client is refused by `make lint`.

To test code that calls an outside service, hide the call behind a function
or interface field on the server, set from `Deps`, and give tests a fake.
Tests never reach the real service, and the egress client refuses the
loopback address a test server would use.

## Sending email

Declare it in `app.yaml`:

```yaml
email:
  smtp_host: email-smtp.ap-southeast-2.amazonaws.com
  purpose: Tells the rota coordinator when a swap is requested
  data: The names on the swap and a link to it
```

Then send through `s.mail`:

```go
package web

import (
	"context"
	netmail "net/mail"

	"github.com/uppertoe/vps-app-kit/mail"
)

func (s *Server) tellCoordinator(ctx context.Context, to, link string) error {
	return s.mail.Send(ctx, mail.Message{
		From:    s.cfg.EmailFrom,
		To:      []netmail.Address{{Address: to}},
		Subject: "A swap has been requested",
		Text:    "A swap has been requested on the rota. You can review it here:\n\n" + link,
	})
}
```

Locally, `make run` prints each email to the log. On the server, set
`EMAIL_BACKEND=smtp` and the SMTP settings in `.env`; the platform owner
mints the SES credentials. Build links from `s.cfg.PublicURL`, never from
the request.

## Emailing the members of a group

The app cannot list a gateway group's members. To email a group of people,
such as coordinators, keep their addresses in a setting, `COORDINATOR_EMAILS`
in `.env`, read as a comma-separated list in `internal/config/config.go`.
Say in the design docs that the list must be kept in step with the group by
hand.

## Receiving a webhook

Declare the sender in `app.yaml`:

```yaml
webhooks:
  - path: /hooks/github
    from: GitHub
    purpose: Rebuilds the handbook when its source changes
```

The handler needs the database and the secret, which do not exist while
`make generate` lists the routes, so build it in `New` and register whatever
was built. In `internal/web/hooks.go`:

```go
package web

import (
	"context"
	"errors"
	"net/http"
	"os"

	"github.com/uppertoe/vps-app-kit/webhook"
)

// newGitHubHook builds the verified handler. The secret is at least 32
// characters, set in the server's .env and in GitHub's webhook settings.
func (s *Server) newGitHubHook(ctx context.Context) (http.Handler, error) {
	secret := os.Getenv("GITHUB_WEBHOOK_SECRET")
	if len(secret) < webhook.MinSecretBytes {
		return nil, errors.New("GITHUB_WEBHOOK_SECRET must be at least 32 characters")
	}
	replay, err := s.store.WebhookReplay(ctx)
	if err != nil {
		return nil, err
	}
	return webhook.Handler(webhook.NewGitHub([]byte(secret)), replay, webhook.Options{Log: s.log},
		func(w http.ResponseWriter, r *http.Request, body []byte) {
			// body is the verified delivery. Act on it, then answer 2xx.
			w.WriteHeader(http.StatusAccepted)
		}), nil
}

// orNotFound serves h, or 404 when h was not built, as when make generate
// lists the routes without a database or settings.
func orNotFound(h http.Handler) http.Handler {
	if h == nil {
		return http.NotFoundHandler()
	}
	return h
}
```

Add a field `github http.Handler` to `Server`, build it at the end of `New`,
before `return s, nil`, so the app refuses to start without the secret:

```go
	if d.Store != nil {
		if s.github, err = s.newGitHubHook(context.Background()); err != nil {
			return nil, err
		}
	}
```

and register it in `Routes`:

```go
rt.Handle("POST /hooks/github", access.Webhook("GitHub pushes"), orNotFound(s.github))
```

A delivery that is unsigned, badly signed, stale, replayed or too large never
reaches the function. Add `GITHUB_WEBHOOK_SECRET=CHANGE_ME` to
`.env.example`.

## Something that runs on a schedule

Add the work to the store, `internal/store/notes.go` for this example:

```go
// DeleteNotesBefore deletes notes saved before the cutoff and says how many.
func (s *Store) DeleteNotesBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	res, err := s.db.ExecContext(ctx, `DELETE FROM notes WHERE created_at < ?`,
		cutoff.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
```

Then start it from `main` in `cmd/server/main.go`, after the snapshot loop
starts, with the same context so it stops when the app stops:

```go
	go func() {
		t := time.NewTicker(24 * time.Hour)
		defer t.Stop()
		for {
			select {
			case <-snapCtx.Done():
				return
			case <-t.C:
				n, err := st.DeleteNotesBefore(snapCtx, time.Now().AddDate(0, -6, 0))
				if err != nil {
					log.Error("delete old notes", "err", err)
					continue
				}
				log.Info("deleted old notes", "count", n)
			}
		}
	}()
```

Keep scheduled work short and safe to repeat: the app may restart at any
time, and an interrupted run should do no harm when it runs again.

## Who is signed in

In a handler, `gateway.From(r.Context())` returns the identity on any route
that is not public. Its fields are `Email`, `Name`, `Role` and `Groups`, with
`InGroup`, `AtDomain`, `IsAdmin` and `Display()` to ask about them. In a
template, the same identity is `.Identity`, and `.SignedIn` says whether
there is one. Never read a name, an email address or a role from a form or a
query string.
