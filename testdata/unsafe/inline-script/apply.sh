perl -0pi -e 's{\{\{end\}\}\s*\z}{<script>alert(1)</script>\n{{end}}\n}' internal/web/templates/home.html
