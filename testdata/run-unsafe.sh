#!/usr/bin/env bash
# run-unsafe.sh — prove that the platform's checks catch each unsafe change.
#
# It initialises a clean copy of the template, commits it (so the migration
# check has a base), confirms kitlint passes, then applies each variant in
# testdata/unsafe/<case>/apply.sh to a fresh copy. Every variant must make
# "kitlint all" fail with the text in its expect file, except a variant with
# an "outcome" file reading "pass", which must pass and print that text.
#
# The rules enforced at run time rather than in the repository (forged and
# repeated identity headers, guests on signed-in routes, cross-site writes,
# webhook signatures and replays, egress redirects and private addresses)
# are proven by the kit's own tests and by the container probe in CI.
#
#   testdata/run-unsafe.sh            uses the kit version in go.mod
#   KITLINT=/path/to/kitlint testdata/run-unsafe.sh
set -euo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

if [[ -z "${KITLINT:-}" ]]; then
  kit="$(cd "$root" && go list -m -f '{{.Version}}' github.com/uppertoe/vps-app-kit)"
  GOBIN="$work/bin" go install "github.com/uppertoe/vps-app-kit/cmd/kitlint@$kit"
  KITLINT="$work/bin/kitlint"
fi

base="$work/base"
rsync -a --exclude .git --exclude data --exclude testdata "$root/" "$base/"
(cd "$base" && bash scripts/init.sh --name demo-app --title "Demo app" --subdomain demo \
  --module github.com/example/demo-app --owner example --repo demo-app --data-class none \
  --contact "Demo Person" --contact-email demo@example.org --organisation "Example Hospital" --yes >/dev/null)
(cd "$base" && git init -q -b main && git add -A && git -c user.name=ci -c user.email=ci@example.org commit -q -m base)

if ! (cd "$base" && "$KITLINT" all -base HEAD . >"$work/baseline.txt" 2>&1); then
  echo "FAIL the clean app does not pass kitlint:"; cat "$work/baseline.txt"; exit 1
fi
echo "ok   the clean app passes"

failed=0
for dir in "$root"/testdata/unsafe/*/; do
  name="$(basename "$dir")"
  expect="$(cat "$dir/expect")"
  outcome="fail"
  [[ -f "$dir/outcome" ]] && outcome="$(cat "$dir/outcome")"
  copy="$work/$name"
  cp -R "$base" "$copy"
  (cd "$copy" && bash "$dir/apply.sh")
  set +e
  (cd "$copy" && "$KITLINT" all -base HEAD . >"$work/$name.txt" 2>&1)
  rc=$?
  set -e
  if [[ "$outcome" == "pass" && $rc -ne 0 ]]; then
    echo "FAIL $name: should pass, but kitlint failed"; failed=1
  elif [[ "$outcome" == "fail" && $rc -eq 0 ]]; then
    echo "FAIL $name: kitlint passed an unsafe change"; failed=1
  elif ! grep -qF -- "$expect" "$work/$name.txt"; then
    echo "FAIL $name: expected \"$expect\""; failed=1
  else
    echo "ok   $name"
    continue
  fi
  sed 's/^/       /' "$work/$name.txt"
  rm -rf "$copy"
done
exit $failed
