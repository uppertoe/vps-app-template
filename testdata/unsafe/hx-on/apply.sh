perl -0pi -e 's{hx-swap="outerHTML"}{hx-swap="outerHTML" hx-on:click="alert(1)"}' internal/web/templates/notes.html
