# vps-app-template

A starting point for a small web app on the vps-scaffold platform: Go, htmx
and Alpine, SQLite, sign-in through the platform's gateway, design docs, and
the platform's checks already passing. The shared parts come from
[vps-app-kit](https://github.com/uppertoe/vps-app-kit), which Renovate keeps
up to date.

The template is meant to be safe to build on with an AI assistant. The kit
makes the risky things hard to write, and CI fails on the rest; `AGENTS.md`
sets out the rules in the form an assistant follows.

## Starting an app

You need Git, Go 1.26 or later (Go downloads the exact version the app
names), `make`, and a GitHub account. Docker is needed only to build the
image locally, and Python with Playwright only to run the browser check.

1. On GitHub, choose **Use this template**, then **Create a new
   repository**. A private repository suits most apps. Create it in the
   `anaes-data-lab` organisation if you can (ask the platform owner to add
   you): the platform's update bot then keeps the app's shared parts
   current, as "Keeping the app up to date" below describes. A repository
   under your own account works too, but you set up those updates yourself.
2. Clone your repository and run `scripts/init.sh` from its root. The script
   asks for the app's name, title, address and data class, and for your name
   and contact address, renames everything, and runs the tests. Commit and push the result; CI runs on
   every push.
3. Run the app with `make run`, then open
   `http://127.0.0.1:8080/dev/identity` and choose who you are signed in as.
   The role comes first (`user` or `admin`), then any groups, such as
   `user,<name>_admin`. This stand-in for the sign-in service exists only in
   the local build.
4. Build the app. If you work with an AI assistant such as Claude Code, ask
   it to read `AGENTS.md` first; the rules there are the ones CI enforces.
   `docs/RECIPES.md` has worked examples of the usual changes: a new page,
   a database change, an outside service, email, a webhook and a scheduled
   job.
5. Before each push, run `make test`, `make lint` and `make generate`, and
   commit what changed.
6. When the app is ready to go on the server, follow `docs/DEPLOYING.md`.

## What you get

| Path | Holds |
|---|---|
| `app.yaml` | The app's name, address, data class and every outside connection |
| `internal/web/server.go` | Every route, each with its access level |
| `internal/web/templates/`, `static/` | Pages, styles and Alpine components |
| `internal/store/` | SQLite access and numbered migrations |
| `brand/` | Logo, icons, colours and fonts; a neutral brand fills every slot |
| `deploy/apps/<name>/` | The folder to copy into the server repo; the `.caddy` file is generated |
| `design-docs/` | Pages for the design docs site; `connections.md` is generated |
| `compliance/` | Where a health app records that its safeguards are in place |
| `docs/SECURITY.md` | The app's security invariants and the tests that prove each |

## Keeping the app up to date

The shared parts of the app come from `vps-app-kit`, pinned to a version in
`go.mod` and in `.github/workflows/ci.yml`. Renovate proposes each new
version as a pull request. Patch releases merge on their own once CI passes;
minor and major releases, and the CI workflow, wait for you to merge them.
If such a pull request fails because a generated file changed, run
`make generate`, commit, and push to its branch.

- **In `anaes-data-lab`:** ask the platform owner to add the repository to
  the update bot. Nothing else is needed.
- **Under your own account:** install the Renovate GitHub app
  (github.com/apps/renovate) on the repository. The repository's
  `renovate.json` already holds the right settings.

Files copied from the template when the repository was created, such as
`AGENTS.md`, the docs and the starter code, are never updated: later
improvements to the template do not reach an existing app.

## Everyday commands

| Command | Does |
|---|---|
| `make run` | Runs the app locally with the dev sign-in stand-in |
| `make test` | Vets and tests the code |
| `make lint` | Runs the platform's checks on the code and templates |
| `make generate` | Regenerates the Caddy snippet and the connections page |
| `make image` | Builds the container image |

## Deploying

CI publishes the image on every push to `main`. The platform owner puts it
on the server; `docs/DEPLOYING.md` sets out what you send them and what they
do with it.

## Data class

Set `data_class` in `app.yaml` honestly. A `personal` app holds information
about identifiable people and needs a privacy page. A `health` app holds
identifiable health information. Until its safeguards are recorded in
`compliance/`, every page carries a banner stating that it is a prototype
and is not for real patient data, and CI warns on every run. Nothing is
blocked, so prototypes can be built and shown.
