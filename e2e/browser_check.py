"""Open every page in a real browser and fail on anything the CSP blocks.

Unit tests cannot show that htmx and Alpine work under the app's Content
Security Policy; a browser can. This script signs in through the dev
stand-in, opens each page at desktop and phone width, and fails on any CSP
violation, console error or failed request. It then adds a note through the
htmx form and checks the Alpine counter.

    make run            # in one terminal
    python3 e2e/browser_check.py http://localhost:8080

Needs: pip install playwright && playwright install chromium
"""

import sys

from playwright.sync_api import sync_playwright

BASE = (sys.argv[1] if len(sys.argv) > 1 else "http://localhost:8080").rstrip("/")
PAGES = [
    ("/", None),
    ("/notes", "staff@example.org|user"),
    ("/admin", "staff@example.org|user,starterapp_admin"),
]
WIDTHS = [1280, 390]


def sign_in(page, identity):
    email, groups = identity.split("|")
    page.goto(BASE + "/dev/identity")
    page.fill("input[name=email]", email)
    page.fill("input[name=groups]", groups)
    page.click("form button")


def main():
    problems = []
    with sync_playwright() as p:
        browser = p.chromium.launch()
        for width in WIDTHS:
            for path, identity in PAGES:
                ctx = browser.new_context(viewport={"width": width, "height": 900})
                page = ctx.new_page()
                if identity:
                    sign_in(page, identity)
                    page.wait_for_load_state("networkidle")
                seen = []
                page.on("console", lambda m: seen.append(f"console {m.type}: {m.text}") if m.type in ("error", "warning") else None)
                page.on("pageerror", lambda e: seen.append(f"page error: {e}"))
                page.on("requestfailed", lambda r: seen.append(f"request failed: {r.url} ({r.failure})"))
                resp = page.goto(BASE + path)
                page.wait_for_load_state("networkidle")
                if resp is None or resp.status != 200:
                    seen.append(f"status {resp.status if resp else 'none'}")
                overflow = page.evaluate("document.documentElement.scrollWidth > window.innerWidth")
                if overflow:
                    seen.append("the page scrolls sideways")
                problems += [f"{path} at {width}px: {s}" for s in seen]
                ctx.close()

        # The htmx form and the Alpine counter, at phone width.
        ctx = browser.new_context(viewport={"width": 390, "height": 900})
        page = ctx.new_page()
        errors = []
        page.on("console", lambda m: errors.append(m.text) if m.type == "error" else None)
        sign_in(page, "staff@example.org|user")
        page.wait_for_load_state("networkidle")
        page.goto(BASE + "/notes")
        page.fill("#note-body", "Hello")
        counter = page.inner_text(".hint")
        if counter != "1995 characters left":
            problems.append(f"Alpine counter reads {counter!r}")
        page.click("form button[type=submit]")
        page.wait_for_selector("#note-list li:has-text('Hello')")
        if page.input_value("#note-body") != "":
            problems.append("the form was not cleared after saving")
        if page.url != BASE + "/notes":
            problems.append(f"htmx did not handle the form; the page moved to {page.url}")
        page.click(".menu-button")
        if not page.is_visible(".menu a[href='/notes']"):
            problems.append("the phone menu does not open")
        problems += [f"/notes form: console error: {e}" for e in errors]
        browser.close()

    for line in problems:
        print("FAIL", line)
    if problems:
        sys.exit(1)
    print(f"ok: {len(PAGES) * len(WIDTHS)} page views, the htmx form, the counter and the menu")


if __name__ == "__main__":
    main()
