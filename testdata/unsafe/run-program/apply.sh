cat > internal/web/run.go <<'GO'
package web

import "os/exec"

func run() { _ = exec.Command("sh", "-c", "id").Run() }
GO
