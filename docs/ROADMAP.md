# Roadmap

This file is the per-version ladder.

One version per PR. PR is a single squashed commit on `main`. CI
is green on linux, macOS, and Windows before merge. Each PR adds
at least one fixture under `tests/fixtures/` whose end-to-end
harness goes from non-zero to zero in the same PR.

## v0.0.x: repo skeleton plus minimum runtime

| Version | Title                                | Status     |
|---------|--------------------------------------|------------|
| v0.0.1  | repo bootstrap                       | shipped    |
| v0.0.2  | `bunpy <file.py>` shells to goipy    | shipped    |
| v0.0.3  | `bunpy run` subcommand               | shipped    |
| v0.0.4  | embedded stdlib smoke                | shipped    |
| v0.0.5  | binary metadata baked at build time  | shipped    |
| v0.0.6  | release tooling actually publishes   | shipped    |
| v0.0.7  | embedded help and man pages          | shipped    |
| v0.0.8  | `bunpy repl`                         | shipped    |

## v0.1.x: package manager (twelve rungs)

| Version | Title                                | Status     |
|---------|--------------------------------------|------------|
| v0.1.0  | pyproject.toml reader (`bunpy pm config`) | shipped |
| v0.1.1  | PyPI client (`bunpy pm info`)        | shipped    |
| v0.1.2  | wheel installer (PEP 427)            | shipped    |
| v0.1.3  | `bunpy add` (single package, naive)  | shipped    |
| v0.1.4  | `bunpy.lock` writer plus reader      | shipped    |
| v0.1.5  | resolver, platform wheels, markers   | shipped    |
| v0.1.6  | dep lanes (`-D`, `-O`, `-P`)         | shipped    |
| v0.1.7  | `bunpy update` and `bunpy outdated`  | shipped    |
| v0.1.8  | `bunpy remove`                       | shipped    |
| v0.1.9  | `bunpy link` and `bunpy unlink`      | shipped    |
| v0.1.10 | `bunpy patch` and `--commit`         | shipped    |
| v0.1.11 | `bunpy why`                          | shipped    |

## v0.2.x: workspaces, audit, publish, create, bunpyx

| Version | Title                                | Status     |
|---------|--------------------------------------|------------|
| v0.2.0  | workspaces                           | shipped    |
| v0.2.1  | audit                                | shipped    |
| v0.2.2  | publish                              | shipped    |
| v0.2.3  | create                               | planned    |
| v0.2.4  | bunpyx                               | planned    |

## v0.3.x: built-in API surface, part 1

`bunpy.serve`, `bunpy.file/write/read`, `bunpy.sql` (sqlite,
postgres, mysql), `bunpy.redis`, `bunpy.s3`, `bunpy.shell`,
`bunpy.dollar`, `bunpy.spawn`, `bunpy.glob`, `bunpy.cron`, the
fetch/URL/Request/Response globals, `bunpy.WebSocket`,
`bunpy.password`, `bunpy.gzip`, `bunpy.base64`.

## v0.4.x: built-in API surface, part 2

`bunpy.dns`, `bunpy.semver`, `bunpy.deep_equals`, `bunpy.cookie`,
`bunpy.CSRF`, `bunpy.escape_html`, `bunpy.HTMLRewriter`,
`bunpy.YAML`, `bunpy.dlopen` (FFI), `bunpy.Worker`,
`bunpy.Terminal`, `bunpy.WebView`, the timer globals,
`bunpy.URLPattern`, `bunpy.set_system_time`, UUID v7.

## v0.5.x: test runner (eight rungs)

Discovery, `bunpy.expect` matchers, mock and `spy_on`,
snapshots, `--parallel`, `--isolate`, `--shard` and `--changed`,
coverage.

## v0.6.x: bundler and `--compile`

`.pyz` output, tree-shaking, `--target`, `--compile` (single Go
binary), `--target browser` (WASM), bundler plugins, build-time
macros, bytecode caching.

## v0.7.x: runtime polish

`--hot`, `--watch`, Go-native asyncio policy, `.env` loader,
sidecar CPython for C-extensions, `bunpy fmt`, `bunpy check`,
Markdown in terminal.

## v0.8.x: Node compatibility shim

`bunpy.node.fs`, `path`, `os`, `http`, `https`, `net`, `tls`,
`crypto`, `worker_threads`, `stream`, `zlib`.

## v0.9.x: performance, reproducibility, docs

Startup-time budget, reproducible `--compile`, corpus benchmarks
against pip, uv, poetry, pdm, install-from-source path, full
docs site.

## v1.0.0: stability commitment

Public surface frozen. Module path `/v1`. Reproducible builds
gate. `docs/COVERAGE.md` accounts for every Bun feature as
implemented, deviated, or skipped.
