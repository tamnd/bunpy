# Changelog

All notable changes to bunpy are recorded here. The format follows
[Keep a Changelog 1.1](https://keepachangelog.com/en/1.1.0/). Once
bunpy reaches 1.0 the project will follow
[Semantic Versioning](https://semver.org/spec/v2.0.0.html); until
then, expect minor version bumps to sometimes include breaking
changes.

## [Unreleased]

## [0.0.3] - 2026-04-26

`bunpy run <file.py>` is the explicit form of `bunpy <file.py>`.
Both routes converge on the same `runtime.Run` call so behaviour
is identical. Pyproject script-name dispatch (`bunpy run
<script>`) waits for the package manager in v0.1.x.

### Added

- `bunpy run <file.py> [args...]` subcommand in
  `cmd/bunpy/main.go`.
- `bunpy run --help` prints the run-scoped usage.
- `bunpy run` with no args writes a usage line to stderr and exits
  non-zero.
- `bunpy run -` reserved for stdin scripts; today it errors with
  "stdin scripts not yet wired".
- `tests/run.sh` runs every fixture twice: positional dispatch and
  through `bunpy run`. Each fixture counts as two assertions, so
  the count doubles.
- Five new go tests: `TestRunSubcommand`, `TestRunSubcommandNoArgs`,
  `TestRunSubcommandHelp`, `TestRunSubcommandStdinReserved`,
  `TestRunSubcommandRejectsNonPyArg`.

### Changed

- `cmd/bunpy/main.go` extracts the file-running path into
  `runFile` so positional and `run` paths share one entry point.
- `docs/ROADMAP.md` marks v0.0.3 shipped, v0.0.4 next.
- `docs/CLI.md` reflects the v0.0.3 surface.
- `docs/COVERAGE.md` flips the `bun run` row to `done`.
- The version banner in `bunpy --help` lists the four wired
  capabilities.

## [0.0.2] - 2026-04-26

`bunpy <file.py>` now runs. It is a thin pipe: gocopy compiles
the source to a CodeObject, the marshal stream hops into goipy,
and goipy's VM runs it. No shell, no subprocess, all in-process.

The set of programs that work is bounded by what gocopy compiles
today (empty modules, `pass`, top-level docstring, single literal
assignment). That set grows as gocopy ships its own rungs; bunpy
inherits the upgrades automatically.

### Added

- `runtime/run.go` exporting `runtime.Run(filename, source, args,
  stdout, stderr)`. Compiles via gocopy, marshals across the
  module boundary, runs on goipy. Honours `SystemExit` and
  formats uncaught Python exceptions through `vm.FormatException`.
- `cmd/bunpy/main.go` dispatch for positional `.py` files. Help,
  version, and a typo path still work.
- `tests/fixtures/v002/` with `empty.py`, `pass.py`,
  `docstring.py`, `assign.py` and matching `expected_*.txt`.
- `tests/run.sh` walks `tests/fixtures/v00*` and asserts each
  fixture's exit code and stdout. Runs on bash, MSYS, and
  git-bash.
- `scripts/sync-deps.sh` clones gopapy, gocopy, and goipy as
  siblings of the bunpy checkout at pinned commits. CI calls it
  before any go command.
- `go.work` listing the four local modules. The bunpy module
  itself stays free of `require` lines for sibling repos so the
  workspace is the single source of pinning.

### Changed

- `cmd/bunpy/main.go` returns `(code int, err error)` from `run`
  so `os.Exit` reflects Python's exit code on `SystemExit`.
- `.github/workflows/{ci,build,release}.yml` run
  `scripts/sync-deps.sh` before `go vet`, `go build`, and `go
  test`.
- `docs/ROADMAP.md` marks v0.0.2 shipped, v0.0.3 next.
- `docs/COVERAGE.md` flips the `bunpy <file>` row to `done`.

### Notes

The `/v1` major-version path on gocopy and gopapy is currently
incompatible with their `v0.x` tags. Until those repos either
drop the suffix or cut a `v1.0.0` tag, bunpy resolves them
through the workspace at pinned commits. The pinning lives in
`scripts/sync-deps.sh`.

## [0.0.1] - 2026-04-26

Bootstrap. The repo is the skeleton: README, LICENSE, the CLI
entry point with `bunpy --version` and `bunpy --help`, the CI
workflow set (lint, test, cross-platform build, tag-driven
release), the changelog tooling, and the docs landing pages.

There is no runtime yet. `bunpy <file.py>` is reserved for
v0.0.2, which wires goipy in. The ladder is in
[`docs/ROADMAP.md`](docs/ROADMAP.md).

### Added

- `cmd/bunpy/main.go` with `bunpy version`, `--version`,
  `bunpy help`, `--help`, and a subcommand router shaped for the
  rungs ahead.
- `LICENSE` (MIT, dated 2026-04-26).
- `README.md` with the Bun-to-bunpy CLI map and the Python API
  shape.
- `.github/workflows/ci.yml` runs `go vet`, `go build`, and
  `go test ./...` on linux, macOS, and Windows.
- `.github/workflows/build.yml` is a cross-compile sanity
  matrix (linux/darwin/windows over amd64/arm64).
- `.github/workflows/release.yml` does the tag-driven build,
  produces archives plus SHA-256 checksums, and creates a
  GitHub release whose body is `changelog/${tag}.md`.
- `scripts/build-changelog.sh` concatenates `changelog/v*.md`
  into `CHANGELOG.md`, gopapy-style.
- `scripts/feature-coverage.sh` plus `scripts/coverage.tsv`
  generate the bunpy-vs-Bun coverage table for
  `docs/COVERAGE.md`.
- `docs/` landing pages: `ARCHITECTURE.md`, `ROADMAP.md`,
  `CLI.md`, `API.md`, `COVERAGE.md`, `COMPATIBILITY.md`,
  `DEVIATIONS.md`.
- `.gitignore`, `.editorconfig`.

### Notes

bunpy targets Python 3.14 because gopapy, gocopy, and goipy do.

