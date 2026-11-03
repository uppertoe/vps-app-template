#!/usr/bin/env bash
# init.sh — turn a fresh copy of vps-app-template into your app.
#
# Run it once, from the repository root, straight after "Use this template":
#
#   scripts/init.sh
#
# It asks for the app's name, title, address and data class, renames
# everything in one pass, removes the files that belong only to the
# template, regenerates the Caddy snippet and connections page, and runs the
# tests. It writes no secrets and changes nothing outside this folder.
#
# For CI and scripted use, every answer can be given as a flag:
#   scripts/init.sh --name rota --title "Rota" --subdomain rota \
#     --module github.com/acme/rota --owner acme --repo rota --data-class none --yes
set -euo pipefail

cd "$(dirname "$0")/.."

if [[ ! -f app.yaml ]] || ! grep -q '^name: starterapp$' app.yaml; then
  echo "init.sh: this copy has already been initialised (app.yaml no longer names starterapp)." >&2
  exit 1
fi

name="" title="" subdomain="" module="" owner="" repo="" data_class="" yes=0
while [[ $# -gt 0 ]]; do
  case "$1" in
    --name) name="$2"; shift 2 ;;
    --title) title="$2"; shift 2 ;;
    --subdomain) subdomain="$2"; shift 2 ;;
    --module) module="$2"; shift 2 ;;
    --owner) owner="$2"; shift 2 ;;
    --repo) repo="$2"; shift 2 ;;
    --data-class) data_class="$2"; shift 2 ;;
    --yes) yes=1; shift ;;
    *) echo "init.sh: unknown option $1" >&2; exit 2 ;;
  esac
done

# Defaults from the git remote, when there is one.
remote="$(git remote get-url origin 2>/dev/null || true)"
if [[ "$remote" =~ github\.com[:/]([^/]+)/([^/.]+)(\.git)?$ ]]; then
  owner="${owner:-${BASH_REMATCH[1]}}"
  repo="${repo:-${BASH_REMATCH[2]}}"
fi

# Lower case, without bash 4 (macOS ships bash 3.2).
lc() { printf '%s' "$1" | tr '[:upper:]' '[:lower:]'; }

ask() { # ask VAR "question" "default"
  local var="$1" q="$2" def="${3:-}" ans
  if [[ -n "${!var}" ]]; then return; fi
  if [[ $yes -eq 1 ]]; then printf -v "$var" '%s' "$def"; return; fi
  read -r -p "$q${def:+ [$def]}: " ans
  printf -v "$var" '%s' "${ans:-$def}"
}

ask repo "GitHub repository name" ""
ask owner "GitHub owner (user or organisation)" ""
ask name "App name, used for the folder and service in the server repo (lower case, digits, hyphens)" "$(lc "$repo")"
ask title "The app's name as people see it" ""
ask subdomain "Subdomain the app is served at" "$name"
ask module "Go module path" "github.com/$owner/$repo"
ask data_class "Most sensitive information the app will hold: none, personal or health" "none"

fail() { echo "init.sh: $*" >&2; exit 1; }
[[ "$name" =~ ^[a-z][a-z0-9-]*$ ]] || fail "the app name must be lower-case letters, digits and hyphens, starting with a letter"
[[ "$subdomain" =~ ^[a-z][a-z0-9-]*$ ]] || fail "the subdomain must be one lower-case label"
[[ -n "$title" && "$title" != *$'\n'* && "$title" != *'"'* ]] || fail "the title is required and must be one line without quotes"
[[ "$module" =~ ^[a-z0-9.-]+(/[A-Za-z0-9._-]+)+$ ]] || fail "the module path must look like github.com/owner/repo"
[[ "$owner" =~ ^[A-Za-z0-9-]+$ && "$repo" =~ ^[A-Za-z0-9._-]+$ ]] || fail "the GitHub owner and repository are required"
case "$data_class" in none|personal|health) ;; *) fail "the data class must be none, personal or health" ;; esac
group="${name//-/_}_admin"

if [[ $yes -ne 1 ]]; then
  echo
  echo "The app will be set up as follows."
  echo "  Name:        $name"
  echo "  Title:       $title"
  echo "  Address:     $subdomain.<the server's domain>"
  echo "  Module:      $module"
  echo "  Image:       ghcr.io/$(lc "$owner")/$(lc "$repo")"
  echo "  Admin group: $group"
  echo "  Data class:  $data_class"
  read -r -p "Continue? [y/N] " ok
  [[ "$ok" =~ ^[Yy] ]] || fail "nothing has been changed"
fi

# Files that belong to the template, not to an app.
rm -rf docs/design-note.md testdata .github/workflows/template-selftest.yml

# Every text file, excluding vendored code and images.
files=()
while IFS= read -r -d '' f; do files+=("$f"); done < <(
  find . -type f \
    -not -path './.git/*' -not -path './internal/web/static/js/vendor/*' \
    -not -name '*.png' -not -name '*.woff2' -not -name 'init.sh' -print0)

export NAME="$name" TITLE="$title" SUBDOMAIN="$subdomain" MODULE="$module" \
  OWNER="$owner" IMAGE="ghcr.io/$(lc "$owner")/$(lc "$repo")" GROUP="$group" DATA_CLASS="$data_class"
perl -0pi -e '
  s{github\.com/uppertoe/vps-app-template}{$ENV{MODULE}}g;
  s{starterapp_admin}{$ENV{GROUP}}g;
  s{starterapp}{$ENV{NAME}}g;
  s{Starter app}{$ENV{TITLE}}g;
  s{^subdomain: starter$}{subdomain: $ENV{SUBDOMAIN}}m;
  s{^data_class: none$}{data_class: $ENV{DATA_CLASS}}m;
  s{ghcr\.io/OWNER/REPO}{$ENV{IMAGE}}g;
' "${files[@]}"
# The template's owner is replaced by yours, in CODEOWNERS only.
perl -pi -e 's{\@uppertoe\b}{\@$ENV{OWNER}}g' .github/CODEOWNERS

mv deploy/apps/starterapp "deploy/apps/$name"
mv "deploy/apps/$name/starterapp.caddy" "deploy/apps/$name/$name.caddy"

echo "Regenerating the Caddy snippet and the connections page."
go run ./cmd/server -caddy > "deploy/apps/$name/$name.caddy"
go run ./cmd/server -connections > design-docs/connections.md
echo "Running the tests."
go vet ./...
go test ./...

rm -- "$0"
echo
echo "The app is set up. Commit the result, then read AGENTS.md before adding features."
echo "Run it locally with: make run"
