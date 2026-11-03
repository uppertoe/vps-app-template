# Conventions

## How a feature is put together

A feature usually touches four places, in this order.

1. **The store.** Add a file under `internal/store/` for the new concern,
   with a migration if the schema changes. Functions take a `context.Context`
   and return plain structs. Write a test in `store_test.go` style, against a
   temporary database.
2. **The route.** Register it in `Routes` in `internal/web/server.go` with
   its access level, then run `make generate`.
3. **The handler.** Add it to `internal/web/handlers.go`, or to a new file
   named after the area of the site. A handler reads the identity with
   `gateway.From`, calls the store, and renders a page with `s.render` or a
   partial with `s.renderPartial`.
4. **The template.** A page is `templates/<name>.html` defining `content`.
   A piece that htmx replaces is a partial, `templates/_<name>.html`, defining
   a named template that both the page and the handler use.

## htmx and plain forms

Every form works without JavaScript: give it `method` and `action` as well as
`hx-post`. The handler answers an htmx request (`HX-Request: true`) with the
partial, and a plain post with a redirect, as `addNote` does.

## Times and text

- Store times as RFC 3339 strings in UTC. Show them in Melbourne time with
  the `date` template function.
- Store text as the person typed it, trimmed. Escaping happens in the
  template, never before saving.

## Tests

- Store tests use a temporary SQLite file.
- Web tests build the server with a temporary store and send requests with
  the `Remote-*` headers set, as `web_test.go` does. Test each route as
  someone who should be admitted and as someone who should not.
- `e2e/browser_check.py` opens every page in a real browser and fails on
  anything the CSP blocks. Add each new page to its `PAGES` list.

## Layout

Pages are a single column, readable at phone width. The masthead menu folds
behind a Menu button below 640 pixels. Check a new page at 390 pixels wide;
the browser check fails on any page that scrolls sideways.
