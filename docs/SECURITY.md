# Security model

The app's security rests on a few invariants. This page states each one and
names the test that proves it, so that a reviewer can check the evidence
rather than the intent. Most of the enforcement lives in
[vps-app-kit](https://github.com/uppertoe/vps-app-kit), which every app
shares, and in the platform's sign-in gateway and Caddy.

| Invariant | How it is enforced | Proven by |
|---|---|---|
| Only the gateway can say who a person is | Caddy deletes any `Remote-*` header a client sends and sets the gateway's; the app publishes no port, so Caddy is the only way in | `kitgate` sends forged headers to every gated route and requires the gateway to refuse; the container probe sends repeated headers; `kitlint all` refuses `ports:`, `network_mode`, extra and external networks, host-path and docker.sock mounts, `cap_add`, `userns_mode` and any security option that turns a protection off; the server's compose audit checks the same on the running containers |
| Every route declares who may reach it | A route cannot be registered without an access level; the Caddy snippet is generated from the same list | The kit's router tests; `kitlint all` fails on a hand-edited or stale snippet |
| Caddy's gate is never weaker than the app's | Each level maps to exactly one Caddy guard | `kitgate` signs in as five kinds of person and requires every refusal to come from the gateway itself, not only from the app |
| A public page never sees an identity | The router deletes identity headers on public and webhook routes, and Caddy strips them with `strip_public` | The kit's router tests; `kitgate` |
| Writes come only from the app's own pages | The router refuses a state-changing request whose origin is not `PUBLIC_URL` | The kit's secure tests (sibling subdomain, `Origin: null`, lookalike hosts); `kitgate` cross-site writes |
| No script runs that the app did not ship | A CSP with a per-response nonce and no `unsafe-eval` or `unsafe-inline`; no inline handlers or styles | `e2e/browser_check.py` fails on any CSP violation; `kitlint` refuses inline handlers, `style=`, `hx-on`, `x-html` and escape-bypassing types |
| The app talks only to declared services | `s.http` reaches only `app.yaml`'s hosts, over HTTPS, never an internal address; every third-party library is declared in `app.yaml`; the server's firewall limits outbound ports, not hosts | The kit's egress tests; `kitlint` refuses the usual ways of making another client or connection, an undeclared library, and a replaced or outdated kit. It checks ordinary code, and is not a sandbox: code written to evade it can |
| Group membership stays in the gateway | The app reads it from each request, or from the gateway's members endpoint at the moment of use | `kitgate` reads a declared group through the app and requires an undeclared one to be refused |
| The dev sign-in stand-in never ships | It is compiled only with the `dev` tag | CI greps the image for it and requires `/dev/identity` to answer 404 |
| A request cannot exhaust memory | The kit's router caps every request body at 64 KiB (256 KiB for a webhook) before anything parses it; the server's read and header timeouts bound slow clients | The kit's router and webhook tests; `internal/web` `TestOversizedBodyRefused` |
| Production settings are strict | `PUBLIC_URL` must be HTTPS except on 127.0.0.1 or localhost; `EMAIL_BACKEND=log` is refused outside local development; placeholder values are refused | `internal/config` tests |

## What the app must get right itself

The kit cannot know which records belong to whom. The starter's notes are
deliberately shared with everyone who can sign in, and the page says so. A
record that belongs to one person must store its owner from the gateway's
identity and name the owner in every query, as `docs/RECIPES.md` shows, with
a test that one person cannot reach another's records. On a shared server
the gateway may admit people from more than one organisation, so a page for
one organisation's staff uses `access.Domain`, not `access.SignedIn`.

## Health information

`data_class: health` puts a prototype banner on every page and a warning on
every CI run until `compliance/health-attestation.md` records that the
server meets the compliance pack's safeguards. The file is a record that the
owner reviews through CODEOWNERS; its existence is not, by itself, evidence
of compliance.

## Backups

The app writes an hourly `VACUUM INTO` snapshot, and the platform's restic
backup copies the snapshot off the server and checks the repository weekly.
A failed snapshot is logged at error level. The scaffold's
`docs/09-recovery.md` describes the restore drill; run it before the app
holds anything that matters.
