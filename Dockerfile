# syntax=docker/dockerfile:1

# Build: pure Go (modernc SQLite), static, stripped. Never add "-tags dev":
# that build contains the sign-in stand-in, and CI refuses an image with it.
FROM golang:1.27.3-alpine@sha256:d200d2dd76b03a99b52651fd1325aff7dcd35422c35c01bac56571b0507f3deb AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/app ./cmd/server
# The data folder, owned by the nonroot uid, so an empty named volume
# mounted there inherits the ownership.
RUN mkdir -p /data/snapshot && chown -R 65532:65532 /data && chmod -R 700 /data

# Run: distroless, no shell, nonroot (65532). The binary probes itself with
# -healthcheck, so the image needs no curl.
FROM gcr.io/distroless/static-debian12:nonroot@sha256:afa5c872c891853ca7fcf1f12c3edb23f7eeef36189728842dd51042ff57f7ab
COPY --from=build /out/app /app
COPY --from=build --chown=65532:65532 /data /data
COPY --from=build /usr/local/go/lib/time/zoneinfo.zip /zoneinfo.zip
ENV ZONEINFO=/zoneinfo.zip TZ=Australia/Melbourne LISTEN_ADDR=:8080 \
    SQLITE_PATH=/data/app.db SNAPSHOT_PATH=/data/snapshot/app.db
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/app"]
