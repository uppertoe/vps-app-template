perl -0pi -e 's{^data_class: none$}{data_class: personal}m' app.yaml
perl -0pi -e 's{\trt.HandleFunc\("GET /privacy"[^\n]*\n}{}' internal/web/server.go
go run ./cmd/server -caddy > deploy/apps/demo-app/demo-app.caddy
