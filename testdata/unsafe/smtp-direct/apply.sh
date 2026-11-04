cat > internal/web/mailer.go <<'GO'
package web

import "net/smtp"

func send() error { return smtp.SendMail("x:25", nil, "", nil, nil) }
GO
