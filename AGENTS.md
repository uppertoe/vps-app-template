# Rules for working in this app

This app runs on a shared, hardened server behind a sign-in gateway. Other
apps, and other people's information, live on the same server, so a few rules
are not negotiable. CI enforces each of them; this file explains them so that
you write code that passes the first time.

Read `docs/CONVENTIONS.md` before adding a feature, and `docs/COPY.md` before
writing anything a person will read. `docs/RECIPES.md` has worked examples of
the usual changes; follow the nearest one.

## Who may reach a page

- Every route is registered in `internal/web/server.go`, in `Routes`, with an
  access level from `github.com/uppertoe/vps-app-kit/access`. A route without
  a level does not compile.
- Choose the narrowest level that works: `Group("starterapp_admin")` for
  administration, `SignedIn()` for anyone signed in, `Public("reason")` only
  for pages anyone in the world may read. The sign-in service on a shared
  server may admit people from more than one organisation, so if a page is
  for one organisation's staff only, use `Domain("<their domain>")` rather
  than `SignedIn()`, and ask the owner which domain.
- Name new groups after the app with underscores, such as
  `starterapp_coordinators`.
- Group membership is set in the sign-in service. Never store a list of a
  group's members in the app, its database or its settings.
- One path prefix has one access level: a route such as `/cases/{id}` guards
  everything under `/cases/`. Put an action with a narrower level under its
  own prefix (see `docs/RECIPES.md`).
- After changing routes, run `make generate`. It rewrites
  `deploy/apps/starterapp/starterapp.caddy` from the route list. Never edit
  that file by hand; CI fails if it differs from a fresh generation.
- Read who the person is with `gateway.From(r.Context())`. Never read the
  `Remote-*` headers or any cookie directly, and never trust a name, email or
  role sent in a form or query string.
- Never add routes under `/dev/`. That path is reserved for the local sign-in
  stand-in, which exists only in `make run`.

## The page

- Templates use `html/template`, which escapes every value. Never use
  `template.HTML`, `safeHTML` or anything else that skips escaping.
- No inline JavaScript: no `<script>` blocks, no `onclick=` or other `on*`
  attributes, no `hx-on`, no `javascript:` links. Behaviour lives in
  `internal/web/static/js/app.js` as `Alpine.data` components, and templates
  refer to them by name: `x-data="counter"`.
- Alpine attributes may name a property or call a method of the component,
  nothing more: `x-text="remaining"`, `x-on:click="toggle"`. Expressions such
  as `x-on:click="open = !open"` do not run under the CSP; add a method.
- Never use `x-html`. Use `x-text`.
- No inline styles: no `style=` attributes and no `<style>` blocks. Add a
  class to `internal/web/static/app.css`. Colours and fonts come from
  `brand/brand.css`.
- Never load anything from another site: no CDN scripts, fonts or images.
  Third-party JavaScript is vendored under `static/js/vendor/` and recorded,
  with its hash, in `static/js/VENDOR.json`.
- Link to static files with `{{asset "app.css"}}` and to brand files with
  `{{brand "logo.svg"}}`, so the URL changes when the file does.

## Data

- Decide who may read each kind of record before writing it. A record that
  belongs to one person stores its owner, taken from `gateway.From`, and
  every query that reads, changes or deletes it names the owner in its
  `WHERE` clause; `docs/RECIPES.md` shows the pattern. A record shared by
  everyone signed in, like the starter's notes, says so on the page. Test
  that one person cannot reach another's records.
- All data lives in SQLite through `internal/store`. Every query passes its
  values as parameters (`?`). Never build SQL by joining strings.
- A schema change is a new file, `internal/store/migrations/0002_<what>.sql`.
  Never edit a migration that has been merged; the app refuses to start if
  one has changed.
- If the app will hold information about identifiable people, including the
  names or email addresses of staff, set `data_class` in `app.yaml` to
  `personal`, or to `health` for health information, and tell the owner.
  Free text that people type about patients, such as a case summary, can
  identify a patient even without a name: if it could, the class is
  `health`. Such
  an app must fill in `design-docs/privacy.md` and the `/privacy` page
  (`templates/privacy.html`) with what it holds, who sees it, where it is
  kept and for how long; `make lint` fails without them. A health app shows a
  prototype banner until the safeguards in `compliance/README.md` are
  recorded.

## Talking to other services

- Every outside service must be listed in `app.yaml` under `outbound`, with
  its purpose and the information it receives. Run `make generate` to update
  the design docs.
- Call it only through `s.http`, the server's client, which reaches only the
  hosts in `app.yaml`. Never use `http.Get` and its siblings,
  `http.DefaultClient`, an `http.Client` value or `new(http.Client)`,
  `http.Transport`, `net.Dial`, `httputil` proxies or `net/smtp`: `make lint`
  fails on each. Hold only the `*http.Client` you are given.
- Every third-party library goes under `libraries` in `app.yaml` with what it
  is for, and `make lint` fails on one that is missing. Before adding one,
  check whether the standard library or the kit already does the job. If a
  library makes its own network connections, give it `s.http`; if it cannot
  take one, ask the owner before using it.
- Send email only through `s.mail`, and declare it under `email` in
  `app.yaml`. A service that sends the app deliveries uses an
  `access.Webhook` route and the kit's `webhook` package. `docs/RECIPES.md`
  shows each.
- Secrets live in `.env` on the server, never in code, `app.yaml`, the design
  docs or a commit. Add the setting's name to
  `deploy/apps/starterapp/.env.example` with the value `CHANGE_ME`.
- Never run other programs (`os/exec`, `os.StartProcess`) or load code at run
  time (`plugin`).

## Logging

- Log with the `*slog.Logger` the server holds, using named fields. Never log
  a request's headers, body, query string or form, a secret, or a person's
  email address.

## Before you finish

Run `make test`, `make lint` and `make generate`, and commit what
`make generate` changed. If the change affects what the app does, who uses
it or what it holds, update `design-docs/`.

## Files you must not edit

- `deploy/apps/starterapp/starterapp.caddy` and `design-docs/connections.md`,
  which are generated.
- `internal/web/static/js/vendor/`, which is changed only with its
  `VENDOR.json` entry.
- `Dockerfile`'s build flags: never add `-tags dev`.
- `.github/workflows/`, unless the owner asks.
