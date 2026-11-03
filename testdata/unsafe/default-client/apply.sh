cat > internal/web/fetch.go <<'GO'
package web

import "net/http"

func fetch() { _, _ = http.Get("https://api.example.org") }
GO
