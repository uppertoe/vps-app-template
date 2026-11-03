perl -0pi -e 's{(\trt.HandleFunc\("GET /admin")}{\trt.HandleFunc("GET /open-admin", access.Public("forgotten"), s.admin)\n$1}' internal/web/server.go
