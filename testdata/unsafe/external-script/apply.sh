perl -0pi -e 's{</head>}{<script src="https://cdn.example.org/x.js"></script>\n</head>}' internal/web/templates/layout.html
