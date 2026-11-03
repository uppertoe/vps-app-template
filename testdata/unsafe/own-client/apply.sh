cat > internal/web/fetch.go <<'GO'
package web

import "net/http"

var client = &http.Client{}
GO
