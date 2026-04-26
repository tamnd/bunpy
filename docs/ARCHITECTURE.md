# Architecture

bunpy is one binary that does what Bun does for JavaScript, applied
to Python:

- **Runtime** — runs `.py` files via the Pure-Go Python toolchain
  (`gopapy` parser → `gocopy` compiler → `goipy` bytecode VM).
- **Package manager** — installs from PyPI, resolves with PubGrub
  over PEP 691 metadata, locks to a stable text-readable
  `bunpy.lock`.
- **Bundler** — bundles a project to a `.pyz` zipapp or, with
  `--compile`, to a single static Go binary embedding the
  bytecode.
- **Test runner** — discovers pytest- and unittest-shaped tests,
  exposes a Bun-flavoured matcher API, runs in parallel and
  isolated per-file.

The full design is in
[`notes/Spec/1100/1169_bunpy.md`](https://github.com/tamnd/notes/blob/main/Spec/1100/1169_bunpy.md);
this page is the in-repo landing summary.

## Pipeline

```
.py source
  │
  ├── gopapy.ParseFile    →  ast.Module          (gopapy/v2)
  ├── gocopy.Compile      →  bytes (.pyc bytes)  (gocopy/v1)
  ├── goipy.Eval          →  Python objects      (goipy/v1)
  │
  ├── bunpy/runtime       →  registers bunpy.* and Web-platform globals
  ├── bunpy/pkg           →  PyPI + resolver + lockfile + workspaces
  ├── bunpy/build         →  bundler + --compile single-binary emitter
  └── bunpy/test          →  discovery + parallel + isolate + coverage
```

## Module layout

```
cmd/bunpy/        — CLI entry: subcommand router + per-command files
runtime/          — embeds goipy.VM; module loader; hot reload; env
api/              — bunpy.* Python-side API written in Go
pkg/              — package manager (resolver, wheel install, lock)
build/            — bundler + --compile + plugins + macros
test/             — bunpy test runner (discovery + parallel + isolate)
repl/             — native REPL on goipy
server/           — bunpy run --hot dev server + Markdown rendering
internal/         — manifest, marker, cache, lockio, pyclink, platform
tests/            — fixtures, corpus, end-to-end run.sh
docs/             — these pages
changelog/        — per-version release notes; rolled into CHANGELOG.md
scripts/          — build-changelog.sh, feature-coverage.sh
.github/workflows/— ci.yml, build.yml, release.yml
```

The detailed pipeline (HTTP server, SQL, Redis, S3, shell, cron,
fetch, WebSocket, FFI, Worker, ...) is documented per-feature in
the spec.
