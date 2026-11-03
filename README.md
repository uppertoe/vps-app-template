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

1. On GitHub, choose **Use this template** and create your repository.
2. Clone it, then run `scripts/init.sh` from the repository root. It asks
   for the app's name, title, address and data class, renames everything,
   and runs the tests. Commit the result.
3. Run the app locally with `make run`, then open
   `http://127.0.0.1:8080/dev/identity` to choose who you are signed in as.
   The stand-in for the gateway exists only in this local build.

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

## Everyday commands

| Command | Does |
|---|---|
| `make run` | Runs the app locally with the dev sign-in stand-in |
| `make test` | Vets and tests the code |
| `make lint` | Runs the platform's checks on the code and templates |
| `make generate` | Regenerates the Caddy snippet and the connections page |
| `make image` | Builds the container image |

## Deploying

CI builds and publishes an image tagged with the commit on every push to
`main`. In the server repo, copy `deploy/apps/<name>/` to `apps/<name>/`, pin
the image by digest, add the compose file to the `include:` list, re-render
the Caddy routes and deploy, as the scaffold's `docs/04-server-repo.md`
describes. Point the backup at the snapshot folder with
`backup.env.example`.

## Data class

Set `data_class` in `app.yaml` honestly. A `personal` app holds information
about identifiable people and needs a privacy page. A `health` app holds
identifiable health information. Until its safeguards are recorded in
`compliance/`, every page carries a banner stating that it is a prototype
and is not for real patient data, and CI warns on every run. Nothing is
blocked, so prototypes can be built and shown.
