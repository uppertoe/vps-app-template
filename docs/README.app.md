# Starter app

Replace this paragraph with one sentence on what the app helps people do.

The app was started from
[vps-app-template](https://github.com/uppertoe/vps-app-template) and runs on
the vps-scaffold platform behind its sign-in service. The shared parts come
from [vps-app-kit](https://github.com/uppertoe/vps-app-kit), which Renovate
keeps up to date.

## Working on the app

You need Git, Go 1.26 or later (Go downloads the exact version the app
names) and `make`. Docker is needed only to build the image locally, and
Python with Playwright only to run the browser check.

- Run the app with `make run`, then open
  `http://127.0.0.1:8080/dev/identity` and choose who you are signed in as.
  The role comes first (`user` or `admin`), then any groups, such as
  `user,starterapp_admin`. This stand-in for the sign-in service exists only
  in the local build.
- If you work with an AI assistant such as Claude Code, ask it to read
  `AGENTS.md` first; the rules there are the ones CI enforces.
  `docs/RECIPES.md` has worked examples of the usual changes.
- Before each push, run `make test`, `make lint` and `make generate`, and
  commit what changed. CI runs the same checks, and more, on every push.

## Commands

| Command | Does |
|---|---|
| `make run` | Runs the app locally with the dev sign-in stand-in |
| `make test` | Vets and tests the code |
| `make lint` | Runs the platform's checks |
| `make generate` | Regenerates the Caddy snippet and the connections page |
| `make image` | Builds the container image |

## Deploying

CI publishes the image on every push to `main`. The platform owner puts it
on the server; `docs/DEPLOYING.md` sets out what you send them and what they
do with it.
