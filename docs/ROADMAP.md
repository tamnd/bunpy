# Roadmap

This file mirrors the per-version ladder. The full roadmap with
reserved spec-doc numbers lives in
[`notes/Spec/1100/1170_bunpy_roadmap.md`](https://github.com/tamnd/notes/blob/main/Spec/1100/1170_bunpy_roadmap.md).

One version per PR. PR is a single squashed commit on `main`. CI is
green on linux, macOS, and Windows before merge. Each PR adds at
least one fixture under `tests/fixtures/` whose end-to-end harness
goes from non-zero to zero in the same PR.

## v0.0.x — repo skeleton + minimum runtime

| Version | Title                                | Status     |
|---------|--------------------------------------|------------|
| v0.0.1  | repo bootstrap                       | shipped    |
| v0.0.2  | `bunpy <file.py>` shells to goipy    | next       |
| v0.0.3  | `bunpy run` subcommand               | planned    |
| v0.0.4  | embedded stdlib smoke                | planned    |
| v0.0.5  | binary metadata baked at build time  | planned    |
| v0.0.6  | release tooling actually publishes   | planned    |
| v0.0.7  | embedded help + man pages            | planned    |
| v0.0.8  | `bunpy repl`                         | planned    |

## v0.1.x — package manager (twelve rungs)

`bunpy add`, `bunpy install`, lockfile, PubGrub resolver, PyPI
client, wheel installer, `bunpy update / outdated / remove / link
/ patch / why`.

## v0.2.x — workspaces + audit + publish + create + bunpyx

## v0.3.x — built-in API surface, part 1

`bunpy.serve`, `bunpy.file/write/read`, `bunpy.sql` (sqlite +
postgres + mysql), `bunpy.redis`, `bunpy.s3`, `bunpy.shell` /
`bunpy.dollar`, `bunpy.spawn`, `bunpy.glob`, `bunpy.cron`,
fetch/URL/Request/Response globals, `bunpy.WebSocket`,
`bunpy.password`, `bunpy.gzip / base64`.

## v0.4.x — built-in API surface, part 2

`bunpy.dns / semver / deep_equals`, `bunpy.cookie / CSRF /
escape_html`, `bunpy.HTMLRewriter`, `bunpy.YAML`, `bunpy.dlopen`
(FFI), `bunpy.Worker`, `bunpy.Terminal`, `bunpy.WebView`, timer
globals, `bunpy.URLPattern`, `bunpy.set_system_time`, UUID v7.

## v0.5.x — test runner (eight rungs)

Discovery, `bunpy.expect` matchers, mock / spy_on, snapshots,
`--parallel`, `--isolate`, `--shard / --changed`, coverage.

## v0.6.x — bundler + `--compile`

`.pyz` output, tree-shaking, `--target`, `--compile` (single Go
binary), `--target browser` (WASM), bundler plugins, build-time
macros, bytecode caching.

## v0.7.x — runtime polish

`--hot`, `--watch`, Go-native asyncio policy, `.env` loader,
sidecar CPython for C-extensions, `bunpy fmt`, `bunpy check`,
Markdown in terminal.

## v0.8.x — Node compatibility shim

`bunpy.node.fs / path / os / http / https / net / tls / crypto /
worker_threads / stream / zlib`.

## v0.9.x — performance, reproducibility, docs

Startup-time budget, reproducible `--compile`, corpus benchmarks
vs pip / uv / poetry / pdm, install-from-source path, full docs
site.

## v1.0.0 — stability commitment

Public surface frozen. Module path `/v1`. Reproducible builds
gate. `docs/COVERAGE.md` has every Bun feature accounted for as
implemented / deviated / skipped.
