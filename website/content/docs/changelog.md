---
title: Changelog
description: Curated milestone changelog for bunpy v0.1.x through v0.10.x -- what changed, what was removed, and why.
weight: 90
---

This page covers milestone releases from v0.1.x to v0.10.x. Patch-level bug fixes are omitted; for the full commit-by-commit history see the [GitHub releases page](https://github.com/tamnd/bunpy/releases).

## v0.10.x (2026-03)

The headline of v0.10 is a clean break: `bunpy.lock` is removed. `uv.lock` is the sole lockfile format going forward.

- **v0.10.29** -- `bunpy.lock` removed entirely. Projects that still have a `bunpy.lock` in their repo get an error with a migration hint on first `bunpy install`. Run `bunpy install --migrate-lock` to convert and delete the file in one step.
- Binary releases now publish on every tag push, not just on manual workflow dispatch. The CDN URL `https://tamnd.github.io/bunpy/install.sh` always points at the latest tag.
- Package manager lock performance: `bunpy pm lock` now resolves a 47-package tree in ~85 ms on a warm cache and ~1.4 s on a cold cache. The speedup comes from a parallel resolver written in Go with HTTP/2 multiplexing, lock seeding, and eager prefetch of transitive dependencies.
- `bunpy upgrade` gained `--version` flag so CI jobs can pin an exact release without touching the install script.
- goipy updated to CPython 3.14.0a7 semantics; 214 of 263 stdlib modules pass the full test suite.

## v0.9.x (2026-01)

v0.9 was the migration release: both lockfile formats coexisted so teams could upgrade incrementally.

- `uv.lock` became the default output for `bunpy install` and `bunpy add`. New projects never see `bunpy.lock`.
- `bunpy install --use-bunpy-lock` let teams keep the old format while testing the new resolver.
- `bunpy pm migrate` command converted an existing `bunpy.lock` to `uv.lock` and printed a diff of the resolved versions.
- Performance pass on the interpreter start path: cold-start time for a `print("hello")` script dropped from ~38 ms to ~9 ms on an M-series Mac. The improvement came from lazy loading stdlib modules and skipping the AST-to-bytecode step for scripts that had a matching `.pyc` in the wheel cache.
- `BUNPY_MAX_PROCS` environment variable added to cap the goroutine pool used for parallel imports.
- `bunpy check` linter gained 14 new rules covering f-string syntax errors and unreachable code after `return`.

## v0.8.x (2025-11)

v0.8 reversed the workspace experiment from v0.7 and tightened the single-project model instead.

- Workspace support removed. The `[tool.bunpy.workspace]` table in `pyproject.toml` is now a hard error with a message pointing at the v0.7 removal notes.
- `bunpy install` is now strictly scoped to the directory containing `pyproject.toml`. There is no recursive project discovery.
- `bunpy add --dev` introduced development-only dependencies that are excluded from `bunpy build` output.
- `bunpy remove` command added.
- `bunpy pm why <package>` prints the dependency chain that brought a package into the lockfile.
- `bunpy.serve` gained TLS support via `ssl_cert` and `ssl_key` kwargs. Self-signed cert generation with `bunpy serve --tls-auto` for local development.
- Internal refactor: the package manager, bundler, test runner, and HTTP server all moved into the same binary with no subprocess calls between components.

## v0.7.x (2025-09)

v0.7 was the workspace preview. It shipped, got real-world feedback, and was pulled back in v0.8 because the model added complexity without clear wins for the typical single-project case.

- `[tool.bunpy.workspace]` table in `pyproject.toml` allowed declaring multiple sub-packages in a monorepo under one root.
- `bunpy install` in the root resolved all workspace members together and produced a single shared `bunpy.lock`.
- Cross-member imports worked via a synthetic path entry injected at startup.
- Known limitation acknowledged in docs: circular member dependencies were silently ignored rather than errored.
- `bunpy run --member <name>` let you run a script from a specific workspace member.

## v0.6.x (2025-07)

v0.6 added the `fetch` global, making HTTP requests available in any script without imports.

- `fetch(url, options={})` injected into every script's global scope. Mirrors the WHATWG Fetch API closely enough that most JavaScript Fetch tutorials translate directly to bunpy.
- `fetch` supports `method`, `headers`, `body`, and `signal` (via `AbortController`).
- Response object has `.text()`, `.json()`, `.bytes()`, `.status`, `.headers`, and `.ok`.
- `AbortController` and `AbortSignal` added as globals alongside `fetch`.
- `bunpy.fetch` module added for cases where explicit imports are preferred over globals: `from bunpy.fetch import fetch`.
- `User-Agent` header set to `bunpy/<version>` by default and can be overridden per request.
- Streaming responses not yet supported (body is buffered); noted in docs as a v0.8 target (it slipped to v0.9).

## v0.5.x (2025-05)

v0.5 introduced `bunpy.serve`, a minimal HTTP server API modelled on `Bun.serve`.

- `from bunpy.serve import serve` starts a blocking HTTP listener on a given port.
- Handler function receives a `Request` object with `.method`, `.path`, `.query`, `.headers`, and `.body`.
- Return a dict with `status` and `body`, or return a `Response` object directly.
- Async handlers supported: `async def handler(req): ...` runs on the event loop.
- `serve(handler, port=3000, hostname="0.0.0.0")` signature.
- `bunpy serve` CLI alias: `bunpy serve server.py` starts the server and restarts it on file change (watch mode).
- Graceful shutdown on SIGINT and SIGTERM.
- No middleware, no router -- by design. Routing is left to userland libraries.

## v0.4.x (2025-03)

v0.4 shipped the test runner.

- `bunpy test` discovers test files matching `tests/test_*.py` or `test_*.py` by default. Pattern overrideable with `--pattern`.
- `from bunpy.test import test, expect` decorator-based API: decorate a function with `@test("description")` to register it.
- `expect(value)` chainable matchers: `.to_be()`, `.to_equal()`, `.to_raise()`, `.to_be_none()`, `.to_be_truthy()`, `.not_()`.
- Tests run in parallel by default across a goroutine pool sized to the number of CPU cores.
- `--serial` flag for sequential execution (useful when tests share state).
- Exit code 1 on any failure, 0 on full pass -- CI-friendly by default.
- Coverage reporting added in v0.4.8: `bunpy test --coverage` prints a per-file line-coverage table.
- Watch mode: `bunpy test --watch` re-runs affected test files on save.
- JUnit XML output via `--reporter junit > results.xml` for CI systems that consume JUnit.

## v0.3.x (2025-01)

v0.3 added the bundler.

- `bunpy build` packages a project into a `.pyz` (Python zip application) archive.
- The `.pyz` contains all source files, all installed dependencies, and a `__main__.py` entry point.
- The archive is runnable with `python3 myapp.pyz` on any machine with CPython 3.12+ installed, or with `bunpy run myapp.pyz` without any system Python.
- `bunpy build --minify` strips docstrings and blank lines to reduce archive size.
- `bunpy build --target <path>` controls output location (default: `<project-name>.pyz` in the current directory).
- The bundler does not embed the goipy runtime. A v0.5 goal of a fully self-contained binary (`bunpy build --standalone`) was deferred after it grew the archive size to over 50 MB in testing.

## v0.2.x (2024-11)

v0.2 introduced the package manager prototype.

- `bunpy add <package>` resolves and installs packages from PyPI.
- `bunpy install` installs all dependencies listed in `pyproject.toml`.
- First lockfile format: `bunpy.lock` (TOML-based, bunpy-specific). This format was later replaced by `uv.lock` in v0.9 and removed in v0.10.
- `bunpy pm list` shows installed packages and versions.
- `bunpy pm why` placeholder added (full implementation came in v0.8).
- Wheel cache at `~/.cache/bunpy/wheels/` shared across projects.
- Pure-Python wheels only in v0.2. Extension wheels (packages with compiled C extensions) came in v0.3.4.

## v0.1.x (2024-09)

The initial release. One goal: run a Python file.

- `bunpy <file.py>` executes a Python script using the embedded goipy interpreter.
- goipy runs CPython 3.14 semantics. No system Python required.
- Standard library modules available at first release: the subset of CPython stdlib that goipy had implemented at the time (roughly 140 modules, mostly `builtins`, `os`, `sys`, `json`, `re`, `math`, `datetime`, and `collections`).
- `bunpy --version` prints version and platform.
- `bunpy init <name>` scaffolds a `pyproject.toml`.
- No package manager, no bundler, no test runner -- those came in subsequent releases.
- Single static binary for macOS arm64 and Linux x86_64. Windows and Linux arm64 added in v0.1.8.
- Binary size: approximately 28 MB compressed.
