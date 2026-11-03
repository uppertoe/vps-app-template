# Go app template: design note

Approved 2026-11-03, after an independent security review. Nothing described
here has been built yet.

## Purpose

The template gives anyone, including someone building with an LLM, a new web
app that already meets the platform's rules: it signs people in through the
gateway, passes the container and image checks, publishes its design docs and
deploys into a server repo without changes. Safety comes from the structure of
the code and from CI, not from the author reading guidance.

The template is taken from the fellowship enquiries app, which is the most
complete of the Go apps, and improves on it where we would now do things
differently. It is for new apps only; moving the existing apps across is a
separate project.

## The three pieces

| Piece | Repository (public) | Holds | Who edits it |
|---|---|---|---|
| Template | `uppertoe/vps-app-template` | The skeleton an app starts from and then owns | The app's author, once copied |
| Kit | `uppertoe/vps-app-kit` | A small Go module for the parts that must stay identical across apps | Only us, with semantic versioning |
| Reusable workflow | `vps-app-kit/.github/workflows/app-ci.yml` | The checks every app runs | Only us |

An app refers to the kit with one line in `go.mod` and to the workflow with one
`uses:` line. Renovate keeps both current. Both repositories must be public so
that apps in other people's organisations can fetch them; neither holds
anything sensitive. Public repositories also run GitHub Actions without using
the organisation's minutes, which matters for the template's own test suite.

### Release and consumption

An app pins the reusable workflow to a full commit SHA, never to a tag, and
pins the kit by version in `go.mod`. Renovate proposes updates to both after
a minimum release age. A person merges updates to the workflow and minor or
major kit releases, and only kit patch releases merge on their own.

The kit's `main` branch is protected and needs review. Release tags are
protected, and the repository carries CODEOWNERS, `SECURITY.md` and private
vulnerability reporting. A release that changes a security default says so
in its notes, because an API check cannot see a change in behaviour.

The reusable workflow declares `permissions: {}` and grants each job only
what it needs. It never runs untrusted code under `pull_request_target` or
`workflow_run`, and every action it uses is pinned by SHA. Images are built
and pushed in a separate job that runs only on a push to the default branch,
so a pull request never holds `packages: write`. Each image carries OCI
source and revision labels, a build provenance attestation and an SBOM.

## The template skeleton

```
cmd/server/main.go          wiring only: config, store, routes, start
internal/config/            env → typed config; refuses to start if a required value is missing
internal/store/             SQLite access, one file per concern
internal/store/migrations/  numbered SQL files, never edited once merged
internal/web/               handlers, one file per area of the site
internal/web/templates/     html/template pages and partials (_*.html)
internal/web/static/        app.css, js/app.js (Alpine components), vendored js/
brand/                      the branding slots (see below)
app.yaml                    the app's manifest: name, access, data class, connections
deploy/apps/<name>/         docker-compose.yml, <name>.caddy, .env.example, backup.env.example
design-docs/                _index.md and pages that meet the docs-site contract
docs/                       CONVENTIONS.md, COPY.md (writing voice for UI copy)
CLAUDE.md, AGENTS.md        the rules, written for an assistant to follow
scripts/init.sh             renames everything in one pass and asks the setup questions
Dockerfile, Makefile, renovate.json, .github/workflows/ci.yml
```

`scripts/init.sh` asks for the app's name, its hostname, who may sign in, the
brand preset and the data class, then renames the module path, the service,
the Caddy snippet, the env file and the design-docs front matter. It finishes
by running the tests, so a new app starts green. The script writes no
secrets, changes nothing outside the repository and deletes itself once it
has run.

`make run` starts the app on 127.0.0.1 with a stand-in for the gateway, as
the fellows app does with `/dev/identity`. The stand-in is compiled only
under a `dev` build tag. The Dockerfile builds without that tag, so the
stand-in's code is absent from every image, and CI confirms that the built
image answers 404 on `/dev/*` and contains none of its symbols. Any
`DEV_AUTH` value in a compose file or `.env.example` fails CI.

### Front-end libraries

htmx and Alpine are vendored under `static/js/`, with `static/js/VENDOR.json`
recording each file's exact version, source URL and SHA-256. CI checks each
file against its recorded hash, and a Renovate custom manager proposes
updates. Because a pull request can change a file and its hash together, CI
also downloads each changed file again from its recorded URL, which must be
on an allowlisted registry, and compares it with the registry's published
integrity value. Go dependencies are not vendored, because `go.sum` already
pins them.

htmx is configured with `allowEval` and `allowScriptTags` off,
`selfRequestsOnly` on and `historyCacheSize` set to 0, so page HTML never
reaches the browser's local storage. Alpine components are registered only
through `Alpine.data` in `app.js`. A lint step fails on `hx-on*`, `x-html` and
inline event handlers.

## The kit

The kit stays small. Each package exists because at least two of the current
apps already do the same thing by hand.

| Package | Provides | Drawn from |
|---|---|---|
| `gateway` | Reads `Remote-Email`, `Remote-User` and `Remote-Groups` into an `Identity` on gated routes only | fellows `identity.go`, every gated app |
| `router` | Route registration in which every route declares its access level; refuses access by default | new |
| `secure` | Security headers, CSP with per-response nonce, same-origin check on writes, body limit, `no-store` on signed-in pages | fellows `security.go` |
| `assets` | `asset()` content-hash URLs and immutable caching | fellows, caffeine, narrator |
| `sqlitex` | Open with WAL, foreign keys and busy timeout; embedded migrations; `Snapshot` by `VACUUM INTO` on its own connection; integrity check | pstonn `store.go`, the neverdrugs self-heal |
| `serve` | Graceful shutdown, JSON logging to stdout, `-healthcheck` flag, `/healthz` | every Go app |
| `mail` | One sending interface with SMTP and log back ends | fellows, pstonn |
| `egress` | An HTTP client that reaches only the hosts declared in `app.yaml`, with timeouts and redacted logging | new |

The `router` package is where most of the safety lies. Each access level
maps to exactly one Caddy guard, and there are no others.

| Access level | Caddy guard | Admits |
|---|---|---|
| `Public` | `strip_public` | Anyone; the app receives no identity |
| `Webhook` | `strip_public` | A request with a valid signature; no identity |
| `SignedIn` | `protected` | Any signed-in user, but not a guest |
| `SignedInOrGuest` | `protected_open` | Any signed-in user and guests from open registration |
| `Domain(d)` | `protected_domains` | Users with an address at the domain |
| `Group(g)` | `protected_groups` | Members of the group, including a guest an admin has placed in it and a break-glass card for it |
| `DomainOrGroup(d, g)` | `protected_access` | Either of the above |

The app enforces the same level itself from the `Remote-*` headers, so either
layer alone keeps a path closed. On a `Public` or `Webhook` route the kit does
not read those headers at all. The role is read only from the first element
of `Remote-Groups`, groups are compared exactly in lower case, and a request
carrying more than one value for any `Remote-*` header is refused.

The kit generates the app's `.caddy` snippet from the route list, and CI
fails if the committed snippet differs from a fresh one. The snippet ends in
a catch-all that answers 404, so a path the app did not register never
reaches the container.

## Branding slots

Branding lives in `brand/`, and the rest of the app reads it from there.

| Slot | File | Used for |
|---|---|---|
| Names | `brand/brand.yaml`: app name, short name, organisation, contact address | Page titles, footer, email sender name, design-docs front matter |
| Logo | `brand/logo.svg` | Header |
| Icon | `brand/icon.svg`, `icon-32.png`, `icon-180.png` | Favicon and home-screen icon |
| Colours and type | `brand/brand.css`: custom properties for colours and font families | Every page, in light and dark |
| Fonts | `brand/fonts/*.woff2` (optional) | Self-hosted, since the CSP allows fonts only from the app |

The template ships a neutral brand only, with every slot filled, so an app
that is never branded still builds and deploys. Every brand file goes through
`asset()`, so a changed logo reaches users at once. A file left out of
`brand/` falls back to the neutral one.

## Connections

`app.yaml` declares every connection the app makes. The manifest drives three
things: the `egress` client refuses any host not listed, `init.sh` and CI
generate the matching lines in `.env.example`, and CI writes a design-docs page
listing what reaches the app and what leaves the server. That page answers the
question a privacy review always asks.

| Connection | Direction | How the template handles it | Version |
|---|---|---|---|
| People in a browser | In | Through Caddy and the gateway; the app reads only the three `Remote-*` headers through `gateway` | 1 |
| Public pages | In | Declared `Public`; the `.caddy` snippet uses `strip_public`, which removes the auth cookie and any `Remote-*` header a client sends | 1 |
| Guests from open registration | In | Declared `SignedInOrGuest`; every other level refuses guests | 1 |
| Webhooks from outside services | In | Declared `Webhook`, outside the gateway, with the checks set out below | 1 |
| Sending email | Out | SES over SMTP on port 587 through `mail`; credentials from `aws-ses-setup.py` | 1 |
| Outside HTTPS APIs (such as Anthropic or RxNav) | Out | Through `egress` only, hosts listed in `app.yaml`; the design-docs page names what data each one receives | 1 |
| Staff alerts | Out | ntfy at its public address, declared like any other host | 1 |
| Backups | Platform | The app writes an hourly `VACUUM INTO` snapshot to `/data/snapshot/`, and `deploy/…/backup.env.example` points restic at that folder rather than at the live database | 1 |
| Logs | Platform | JSON to stdout, collected by journald and the log export | 1 |
| Receiving email | In | SES to S3, polled by the app, as fellows does | Later |
| File storage in S3 | Out | An IAM user scoped to one prefix, made by a setup script | Later |
| Signing in to a third party on the user's behalf (OAuth) | Out | Tokens sealed at rest, as pstonn's `secretbox` does | Later |
| Calls between apps on one server | Between | Not supported: each app has its own network by design | Not planned |

A `Webhook` route accepts POST only. The kit caps the body before reading it,
computes the signature over the raw bytes and compares it in constant time
before anything parses the body. The signed payload must carry a timestamp
within five minutes, and event IDs are kept in SQLite for a period, so a
replayed delivery is refused. Each webhook has its own secret of at least
32 bytes, the app refuses to start without it, and two secrets are accepted
while one is being rotated. Every failure gets the same response, is rate
limited and is logged without the signature. The kit names a verifier for
each provider's scheme, such as GitHub's, rather than offering a generic
HMAC check. A `Webhook` route is the only kind exempt from the same-origin
check.

AWS stays optional. Each AWS connection gets its own IAM user with the least
access that does the job, made by a setup script in the server repo, as the
backup and SES scripts already work. Secrets go in the app's `.env` on the
server and never in `app.yaml`.

The container egress firewall limits outbound traffic to ports 53, 80, 443
and 587, and the firewall is the boundary actually enforced. The manifest is
a reviewed declaration and a guard on calls made through the kit, and the
generated docs page says so. The `egress` client matches hosts exactly,
connects over HTTPS on port 443 only, checks every redirect against the
list, ignores proxy settings in the environment and refuses to connect to
private, link-local or metadata addresses. A static analyser in CI fails on
any HTTP client, default transport or direct dial outside the kit, including
an SDK constructed without the kit's client.

The generated connections page names each host and what data it receives,
but never an environment variable's name, since names such as `*_KEY` break
the docs-site contract.

Logging uses an allowlist of named fields. Headers, bodies and query strings
are never logged, and a signed-in user's email address is treated as personal
data in the log. The app refuses to start on a placeholder value copied from
`.env.example`, and gitleaks runs in CI on the template and on every app.

## Data

SQLite is the default store. Migrations are numbered files embedded in the
binary, and CI fails if a migration already on `main` has been changed. Times
are stored as RFC 3339 UTC strings.

`app.yaml` records the data class as `none`, `personal` or `health`. A
`personal` app must have a privacy page and retention periods in code, as
fellows has. A `health` app is subject to the compliance tripwire: AU hosting, TOTP
and encryption at rest. Prototypes must remain buildable, so the tripwire
warns rather than refuses. CI annotates every run of a `health` app until a
file in the repo records that those steps are in place on the target server,
and the app shows the same warning in its startup log. The server repo's CI
gives the same warning when it includes such an app. `app.yaml` and the
attestation file are listed in CODEOWNERS, and the attestation names the
server and the date. Until the attestation exists, every page of the app
carries a banner stating that the prototype is not for real patient data.

Snapshots in `/data/snapshot/` are written at mode 600. A snapshot keeps
deleted rows until the next one is taken, and the backups keep them for the
backup retention period, so a `personal` or `health` app's privacy page
states that period, as the fellows privacy page does.

## Testing the template

The tests have two jobs: to show that a new app is correct, and to show that
the checks catch an unsafe one. The second matters as much as the first,
because a check that never fails proves nothing.

1. **The kit's own tests.** Unit and fuzz tests on header parsing, the
   same-origin check, the router's access rules and the CSP. `apidiff` stops
   a release that breaks the API without a major version.
2. **Instantiate and test.** CI runs `init.sh` with sample answers into a
   clean directory, then runs vet, the tests, `staticcheck`, `gosec` and
   `govulncheck`. It also checks that no placeholder string survived the
   rename and that the commands in the README work as written.
3. **Behind the real gateway.** CI builds the image, runs it with the §5
   compose block, and puts it behind Caddy with the real guard snippets and
   the real gateway image, pinned by digest. A mock is not enough, because
   the mock in the server repo's `ci-forward-auth-test.sh` cannot evaluate
   `X-Auth-Policy` and so cannot prove guest or group outcomes. Every
   declared route is requested as each of anonymous, guest, user, group
   member, user at another domain and break-glass, and the result must
   match the route's access level. Undeclared paths and path variants
   (case, trailing slash, encoded dots, doubled slashes) are probed the same
   way. The suite also checks that forged `Remote-*` headers never reach a
   public or webhook route, that cross-site writes are refused and that the
   container runs non-root with a read-only root filesystem and no
   published port.
4. **In a browser.** Playwright opens every page at desktop and phone width
   and fails on any CSP violation in the console, which catches inline
   scripts and Alpine expressions that need `eval`. An axe pass checks
   accessibility.
5. **Into a server repo.** CI copies the deploy bundle into a checkout of
   `server-instance-template`, renders the Caddy routes and runs that repo's
   own checks, including the image-pin and compose checks.
6. **Against the docs site.** CI runs the design-docs contract check on the
   generated `design-docs/`.
7. **Known-bad apps.** A `testdata/unsafe/` folder holds small variants that
   each break one rule, and CI requires every one to fail with its expected
   message. The variants cover:
   - the CSP: an inline script and a `style=` attribute;
   - the page: `hx-on` or `x-html`, and a write from a
     sibling subdomain or with `Origin: null`;
   - identity: a route with no access level, forged `Remote-*` headers on a
     public or webhook path, a guest on a `SignedIn` route, a user outside
     the group on a group route, a public catch-all, and a Caddy matcher or
     path variant that disagrees with the router;
   - webhooks: unsigned, badly signed, stale, replayed and oversized
     deliveries;
   - egress: a host missing from `app.yaml`, `http.DefaultClient`, an SDK
     built without the kit's client, a redirect to an undeclared host and a
     hostname that resolves to a private address;
   - data and secrets: SQL built from strings, a secret written to the log,
     a secret-shaped value in `.env.example` and an edited migration;
   - the build: an unpinned image, a root user, a published port, an image
     built with the dev tag or `DEV_AUTH`, a vendored file that matches its
     manifest but not upstream, and a workflow with write-all permissions,
     `pull_request_target` or an unpinned `uses:`.

   A `health` app without an attestation is the one variant that must pass,
   with the warning and the banner present.
8. **Weekly drift run.** Once a week CI makes a fresh app from the template
   and runs the whole suite against the latest scaffold and gateway, so a
   platform change that breaks the template is found before an author finds
   it.

Apps made from the template run a lighter version through the reusable
workflow: steps 2 and 3 for their own code, the vendored-file hashes, the
route comparison and the migration check. The heavier browser and drift runs
stay in the template's own public repository.

## Decisions

The owner decided the following on 2026-11-03.

| Question | Decision |
|---|---|
| Names and home | `uppertoe/vps-app-template` and `uppertoe/vps-app-kit`, both public |
| Alpine build | The CSP build, with every component defined in `app.js`, so the CSP carries no `'unsafe-eval'` |
| Connections | Declared in `app.yaml` and enforced at run time |
| Health data | A warning, not a refusal, so that prototypes can be built |
| Image tags | Commit-SHA tags only, with no `latest` |
| Branding | The neutral brand only; an unbranded app is never blocked |

### After the security review

An independent security review of this plan was carried out on 2026-11-03,
and its findings are folded into the sections above. The owner decided the
four that changed the design on the same day.

| Question | Decision |
|---|---|
| The Caddy snippet | Generated from the route list, with a drift check like the one for `.generated/` |
| Renovate on the kit and the workflow | Automerge off for the reusable workflow and for minor and major kit releases; patch releases merge on their own |
| Inline styles | No `'unsafe-inline'` in `style-src`; templates set positions and colours through classes or CSS custom properties set from `app.js` |
| A banner on `health` prototypes | Shown on every page until the attestation exists, stating that the prototype is not for real patient data; it blocks nothing |

## Not in the first version

Postgres, job queues beyond an in-process scheduler, receiving email, S3 file
storage, third-party OAuth and moving the existing apps. Each can follow once
an app needs it, as a kit package and a template recipe.
