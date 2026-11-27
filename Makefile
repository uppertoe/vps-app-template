.PHONY: run test lint generate check-generated build image vendor-check

KIT := github.com/uppertoe/vps-app-kit
# The kit's version from go.mod, so the checker matches the library.
KIT_VERSION := $(shell go list -m -f '{{.Version}}' $(KIT))

# Run locally with the dev stand-in for the gateway: open
# http://127.0.0.1:$(PORT)/dev/identity to choose who you are.
PORT ?= 8080
run:
	DEV_DB=$${DEV_DB:-./data/dev.db}; mkdir -p ./data; \
	LISTEN_ADDR=127.0.0.1:$(PORT) PUBLIC_URL=http://127.0.0.1:$(PORT) SQLITE_PATH=$$DEV_DB \
	SNAPSHOT_PATH=./data/snapshot/dev.db EMAIL_BACKEND=log EMAIL_FROM='Dev <dev@example.org>' \
	go run -tags dev ./cmd/server

test:
	go vet ./...
	go test ./...

# The same checks CI runs, so that a push which passes here passes there:
# formatting, staticcheck, known vulnerabilities, and the platform's checks
# on the code and templates. The two tool versions match the kit's CI
# workflow (app-ci.yml); change them together. Each tool is built with the
# Go the app is checked with, as CI does, or it misreads the standard library.
STATICCHECK := honnef.co/go/tools/cmd/staticcheck@v0.8.2
GOVULNCHECK := golang.org/x/vuln/cmd/govulncheck@v1.9.0
lint:
	@test -z "$$(gofmt -l .)" || { gofmt -l .; echo "Run gofmt -w ." >&2; exit 1; }
	GOTOOLCHAIN=$$(go env GOVERSION) go run $(STATICCHECK) ./...
	GOTOOLCHAIN=$$(go env GOVERSION) go run $(GOVULNCHECK) ./...
	go run $(KIT)/cmd/kitlint@$(KIT_VERSION) all .

# Regenerate the files built from the code and the manifest. Commit the
# result; CI fails if these files differ from a fresh generation.
generate:
	go run ./cmd/server -caddy > deploy/apps/starterapp/starterapp.caddy
	go run ./cmd/server -connections > design-docs/connections.md

check-generated: generate
	git diff --exit-code -- deploy design-docs

build:
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o bin/starterapp ./cmd/server

image:
	docker build -t starterapp .
