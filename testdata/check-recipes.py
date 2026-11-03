"""Prove that every code block in docs/RECIPES.md compiles.

It copies the template to a temporary folder, applies every recipe at once
(files, routes, the migration, the schedule), stubs any handler a route line
names but no recipe defines, then vets and tests the result and checks that
the routes still generate. Run from the repository root:

    python3 testdata/check-recipes.py
"""

import os
import re
import shutil
import subprocess
import sys
import tempfile

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))


def run(cmd, cwd, env=None):
    r = subprocess.run(cmd, cwd=cwd, env=env, capture_output=True, text=True)
    if r.returncode != 0:
        print(f"FAIL {' '.join(cmd)}\n{r.stdout}\n{r.stderr}")
        sys.exit(1)
    return r.stdout


def main():
    work = tempfile.mkdtemp(prefix="recipes-")
    app = os.path.join(work, "app")
    shutil.copytree(ROOT, app, ignore=shutil.ignore_patterns(".git", "data", "testdata"))
    md = open(os.path.join(app, "docs", "RECIPES.md")).read()
    blocks = re.findall(r"```(\w+)\n(.*?)```", md, re.S)
    go = [b for lang, b in blocks if lang == "go"]
    web = os.path.join(app, "internal", "web")

    def write(rel, body, mode="w"):
        with open(os.path.join(app, rel), mode) as f:
            f.write(body)

    n_files = 0
    for b in go:
        if b.startswith("package "):
            pkg = b.split()[1]
            n_files += 1
            write(os.path.join("internal", pkg, f"recipe{n_files}.go"), b)
        elif b.lstrip().startswith("// ") and "func (s *Store)" in b:
            write(os.path.join("internal", "store", "notes.go"), "\n" + b, "a")
    n_sql = 1
    for lang, b in blocks:
        if lang == "sql":
            n_sql += 1
            write(os.path.join("internal", "store", "migrations", f"{n_sql:04d}_recipe.sql"), b)
        if lang == "html":
            write(os.path.join("internal", "web", "templates", "rota.html"), b)

    srv_path = os.path.join(web, "server.go")
    srv = open(srv_path).read()
    route_lines = [l.strip() for b in go for l in b.splitlines() if l.strip().startswith("rt.")]
    srv = srv.replace("\treturn rt\n}", "\t" + "\n\t".join(route_lines) + "\n\n\treturn rt\n}")
    if "github *webhook.Endpoint" not in srv and any("s.github" in l for l in route_lines):
        srv = re.sub(r"(type Server struct \{.*?)\n\}", r"\1\n\tgithub *webhook.Endpoint\n}", srv, count=1, flags=re.S)
        srv = srv.replace("import (\n", 'import (\n\t"github.com/uppertoe/vps-app-kit/webhook"\n', 1)
        build = [b for b in go if b.lstrip().startswith("if d.Store != nil")][0]
        srv = srv.replace("\treturn s, nil\n}", build + "\treturn s, nil\n}", 1)
        srv = srv.replace("import (\n", 'import (\n\t"context"\n', 1)
    existing = "".join(open(os.path.join(web, f)).read() for f in os.listdir(web) if f.endswith(".go"))
    existing += srv
    for name in sorted(set(re.findall(r"\bs\.(\w+)\)?\s*$", "\n".join(route_lines), re.M))):
        is_field = re.search(r"type Server struct \{[^}]*\n\t" + name + r"\s", srv) is not None
        if f"func (s *Server) {name}(" not in existing and not is_field:
            srv += f"\nfunc (s *Server) {name}(w http.ResponseWriter, r *http.Request) {{}}\n"
    open(srv_path, "w").write(srv)

    main_path = os.path.join(app, "cmd", "server", "main.go")
    mn = open(main_path).read()
    sched = [b for b in go if b.lstrip().startswith("go func()")]
    if sched:
        anchor = "\tgo snapshotLoop(snapCtx, log, st.Path(), cfg.SnapshotPath)\n"
        if anchor not in mn:
            print("FAIL main.go no longer has the snapshot loop the schedule recipe refers to")
            sys.exit(1)
        mn = mn.replace(anchor, anchor + sched[0])
        open(main_path, "w").write(mn)

    env = dict(os.environ, GITHUB_WEBHOOK_SECRET="x" * 40)
    run(["gofmt", "-w", "."], app)
    run(["go", "vet", "./..."], app, env)
    run(["go", "test", "./..."], app, env)
    routes = run(["go", "run", "./cmd/server", "-routes"], app, env)
    run(["go", "run", "./cmd/server", "-caddy"], app, env)
    shutil.rmtree(work)
    print(f"ok: {len(go)} Go blocks from docs/RECIPES.md compile, test and generate; {routes.count(chr(10))} routes")


if __name__ == "__main__":
    main()
