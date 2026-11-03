cat >> internal/web/render.go <<'GO'

func unsafeBody(s string) any { return template.HTML(s) }
GO
