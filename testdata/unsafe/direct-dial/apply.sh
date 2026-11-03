cat > internal/web/fetch.go <<'GO'
package web

import "net"

func dial() { _, _ = net.Dial("tcp", "10.0.0.1:5432") }
GO
