# Changelog

All notable changes to bunpy are recorded here. The format follows
[Keep a Changelog 1.1](https://keepachangelog.com/en/1.1.0/). Once
bunpy reaches 1.0 the project will follow
[Semantic Versioning](https://semver.org/spec/v2.0.0.html); until
then, expect minor version bumps to sometimes include breaking
changes.

## [Unreleased]

## [0.0.7] - 2026-04-27

`bunpy --help` is no longer the only thing that prints help.
This rung adds `bunpy help <cmd>` (long-form help inline) and
`bunpy man <cmd>` (the bundled roff manpage). Both surfaces
ship as bytes embedded in the binary, so no network or
filesystem lookup is required to read them.

A single source of truth keeps the two help routes honest. The
`helpRegistry` map in `cmd/bunpy/help.go` holds one entry per
subcommand; both `bunpy help <cmd>` and `bunpy <cmd> --help`
print the same body. CI asserts byte equality across every
wired command on every push, so the two surfaces cannot drift.

The manpages are a separate artefact. They live under
`internal/manpages/man1/` as roff and are embedded via
`//go:embed`. The new `internal/manpages` Go package exposes
`Page(name)` and `List()`; `bunpy man <cmd>` shells through
`Page`, and `bunpy man --install [dir]` walks the embedded FS
and copies each page into `<dir>/man1/`. The release archives
now ship pre-rendered pages under `share/man/man1/*.1`, picked
up by `install.sh` and the Homebrew formula automatically.

### Added

- `cmd/bunpy/help.go`: `helpRegistry`, `helpSubcommand`,
  `printHelp`, `helpTopics`. Five entries cover the wired
  surface: `run`, `stdlib`, `version`, `help`, `man`.
- `cmd/bunpy/manpages.go`: `bunpy man <cmd>` prints the embedded
  roff to stdout; `bunpy man --install [dir]` writes all pages
  into `<dir>/man1/`. Default dir is `$HOME/.bunpy/share/man`.
- `internal/manpages/`: tiny Go package whose only job is to
  host `//go:embed man1/*.1`. Exports `FS()`, `Page(name)`,
  `List()`, plus the `Root` constant.
- `internal/manpages/man1/*.1`: six roff manpages (bunpy.1,
  bunpy-run.1, bunpy-stdlib.1, bunpy-version.1, bunpy-help.1,
  bunpy-man.1). All start with a `.TH` header.
- Tests: `internal/manpages/manpages_test.go` covers the embed
  load, the `.TH` prefix on every page, and a missing-page
  error. `cmd/bunpy/main_test.go` adds parity tests for
  `bunpy help <cmd>` vs `bunpy <cmd> --help`, the unknown-help
  error path, and the `bunpy man --install` round-trip.

### Changed

- `cmd/bunpy/main.go`: dispatch grows `help` and `man` cases.
  `--help` branches in `run`, `stdlib`, `version`, and `man`
  all route through `printHelp(name, ...)`. The top-level
  `usage()` iterates `helpTopics()` so the command list stays
  in sync with the registry.
- `install.sh`: after the binary is in place, the script runs
  `bunpy man --install $INSTALL_DIR/share/man` and prints a
  `MANPATH` hint alongside the existing `PATH` hint.
- `scripts/formula.rb.tmpl`: the Homebrew install block now
  picks up `share/man/man1/*.1` from the staged archive when
  present. Existing installs upgrade without manual steps.
- `.github/workflows/release.yml`: each matrix leg builds a
  host binary and runs `bunpy man --install` against the
  staged archive so the linux and darwin tarballs ship
  rendered manpages. Windows archives skip this.
- `.github/workflows/ci.yml`: smoke job now asserts
  `bunpy help <cmd>` matches `bunpy <cmd> --help` byte-for-byte
  for every wired command, prints `bunpy man run` and checks
  for the `.TH` header, and round-trips `bunpy man --install`
  into a tempdir.
- `docs/CLI.md`: Meta section grows the `help` and `man`
  entries; the synopsis preamble updates the wired surface
  list.
- `docs/INSTALL.md`: documents `MANPATH` setup and the manual
  manpage install path.
- `docs/ARCHITECTURE.md`: new "Help and manpages" section, and
  `internal/manpages/` lands in the module layout block.
- `docs/ROADMAP.md`: v0.0.7 marked shipped, v0.0.8 (`bunpy
  repl`) next.

### Notes

- Manpages are not generated for Windows archives; the
  release workflow gates on `matrix.goos != 'windows'`.
- The release archives are still backwards-compatible: older
  consumers that ignore `share/man/` keep working unchanged.

## [0.0.6] - 2026-04-27

The release pipeline now actually delivers binaries. Two new
install paths land in this rung: a one-liner shell installer
(`curl ... | bash`) and a Homebrew tap.

The release archive format itself is unchanged. v0.0.6 is purely
about consumption: the install script downloads the right archive
for your os/arch, verifies the SHA-256 against the release's
`SHA256SUMS`, and drops the binary at `$HOME/.bunpy/bin`. The
Homebrew tap is updated automatically by a new `tap` job in
`release.yml` whenever a tag ships.

### Added

- `install.sh` at the repo root. Pure bash, uses only `curl`,
  `tar`, `shasum`/`sha256sum`. Resolves the latest tag from the
  GitHub releases API or pins via `BUNPY_VERSION`. Verifies
  checksum from the aggregated `SHA256SUMS` file. Re-running
  upgrades in place; the previous binary is preserved at
  `bin/bunpy.prev` for rollback.
- `scripts/formula.rb.tmpl`: Homebrew formula template with
  `@@VERSION@@` and four `@@SHA_*@@` placeholders.
- `scripts/render-formula.sh`: reads a `SHA256SUMS` file plus a
  bare version, prints `Formula/bunpy.rb` to a destination path.
  Errors loudly if any of the four supported os/arch sums is
  missing. Windows zips are intentionally not surfaced through
  Homebrew.
- `scripts/test-render-formula.sh` and
  `scripts/test-install-sh.sh`: bash smoke tests wired into
  `.github/workflows/ci.yml` (linux + macOS only).
- `docs/INSTALL.md`: install one-liner, Homebrew tap, manual
  download, env overrides, verifying steps. Linked from
  `README.md`.
- `tamnd/homebrew-bunpy` repo created out-of-band as the formula
  destination. The `tap` job in `release.yml` clones, rewrites
  `Formula/bunpy.rb`, and pushes via the `BREW_TAP_TOKEN` secret.

### Changed

- `.github/workflows/release.yml` grows a `tap` job that runs
  after the existing `release` job and is gated on the
  `BREW_TAP_TOKEN` repo secret. The job is a no-op when the
  secret is absent so forks and contributors do not need it.
- `.github/workflows/release.yml` also uploads `SHA256SUMS` as a
  workflow artifact named `sha256sums`, which the `tap` job
  consumes via `download-artifact`.
- `.github/workflows/ci.yml` runs the install.sh and
  render-formula smokes on every push/PR.
- `README.md` leads with the install one-liner and the Homebrew
  tap. Quick-start swaps to the installed `bunpy` command.
- `docs/CLI.md` and `docs/ARCHITECTURE.md` both gained pointers
  to `docs/INSTALL.md` and the `Distribution` paragraph.
- `docs/ROADMAP.md` marks v0.0.6 shipped, v0.0.7 next.

### Notes

- The `tap` job's first end-to-end run requires a one-time setup
  step: a `BREW_TAP_TOKEN` repo secret on `tamnd/bunpy` (a
  fine-grained PAT with `Contents: write` on
  `tamnd/homebrew-bunpy`, nothing else). Until that secret is
  set, the job logs "no BREW_TAP_TOKEN secret; skipping tap
  update" and exits 0, so the rest of the release succeeds
  regardless.
- Windows users are not yet covered by Homebrew or `install.sh`.
  The release zips ship as before; documented fallback in
  `docs/INSTALL.md`.

## [0.0.5] - 2026-04-27

`bunpy version` is now load-bearing. The version string, commit,
build date, and the three pinned sibling-toolchain commits
(gopapy, gocopy, goipy) are baked into the binary via `-ldflags`
at build time. A locally-built binary stays honest by printing
just `bunpy dev`, hiding the commit/built/toolchain lines that
only release builds get to claim.

This rung also adds machine-readable output. `bunpy version
--short` prints the version string. `bunpy version --json` prints
a one-line JSON object that scripts can pipe into `jq`. CI
verifies the JSON shape on every push so the contract is locked.

The hardcoded `var version = "0.0.x"` in `cmd/bunpy/main.go` is
gone. There was a foot-gun: every release I had to remember to
bump the constant or the binary would lie about what it was.
The single source of truth now is `BUNPY_VERSION` in the build
pipeline, which `release.yml` derives from the tag and `build.yml`
sets to `dev`.

### Added

- `runtime/buildinfo.go` declares six build-time string vars
  (`Version`, `Commit`, `BuildDate`, `Goipy`, `Gocopy`, `Gopapy`)
  plus a `BuildInfo` struct and a `Build()` accessor that fills
  in `Go`, `OS`, `Arch` from `runtime.GOOS`/`GOARCH`/`Version`.
- `runtime/buildinfo_test.go` covers dev defaults, host fields,
  and JSON shape.
- `cmd/bunpy/main.go` grows a `versionSubcommand` that handles
  `--short` and `--json`. The plain form prints a multi-line
  banner; the dev form skips the commit/built/toolchain lines.
- `scripts/build-ldflags.sh` prints the `-ldflags` string used by
  every build pipeline. It reads `BUNPY_VERSION`, `BUNPY_COMMIT`,
  and `BUNPY_BUILD_DATE` from the environment (with sensible
  defaults) and pulls pinned dep commits from
  `scripts/sync-deps.sh` so there is one source of truth.
- `.github/workflows/ci.yml` smoke job builds with metadata,
  asserts `bunpy version --short` prints `dev`, and validates
  `bunpy version --json` shape via host `python3`.
- Five new go tests in `cmd/bunpy/main_test.go`:
  `TestVersionShort`, `TestVersionJSON`, `TestVersionDevBuild`,
  `TestVersionUnknownFlag`, plus an updated `TestVersion` that
  no longer depends on a hardcoded constant.

### Changed

- `cmd/bunpy/main.go` no longer carries `var version`,
  `var commit`, `var buildDate`. All metadata flows through
  `runtime.Build()`.
- `.github/workflows/build.yml` and
  `.github/workflows/release.yml` source
  `scripts/build-ldflags.sh` instead of inlining `-X main.foo=...`
  flags. Tag-driven builds set `BUNPY_VERSION` from
  `${GITHUB_REF_NAME#v}`; main builds set it to `dev`.
- `bunpy version` output is now multi-line with a separate
  go/os/arch line. Plain and `(commit ..., built ...)` forms
  documented in `docs/CLI.md`.
- `docs/ROADMAP.md` marks v0.0.5 shipped, v0.0.6 next.
- `docs/ARCHITECTURE.md` adds a "Build metadata" paragraph.
- `docs/CLI.md` documents `--short` and `--json` and the
  dev/release distinction.
- The version banner in `bunpy --help` lists the five wired
  capabilities, with the version subcommand spelled out as
  `--version (with --short and --json)`.

## [0.0.4] - 2026-04-27

`bunpy stdlib` exposes the list of Python stdlib modules baked
into the binary by goipy. The list is generated at build time
from goipy's module switch and pinned in `runtime/stdlib_index.go`,
so it never lies about what `import X` will find. CI re-runs the
generator into a tempdir and diffs against the checked-in file,
which catches drift the moment the goipy pin moves.

This rung is on the runtime ladder (not the package manager
ladder). It does not download anything, talk to PyPI, or read
the project's `pyproject.toml`. It only answers "what stdlib
modules ship inside this binary right now". Real `import`
support waits for gocopy to compile import statements; that
arrives on a later v0.0.x rung.

### Added

- `bunpy stdlib` subcommand in `cmd/bunpy/main.go`, with three
  modes: `ls` (default), `count`, and `--help`.
- `runtime/stdlib_index.go`: generated, sorted, deduplicated list
  of 184 Python stdlib modules embedded by goipy.
- `runtime/stdlib.go` exports `StdlibModules()` (returns an
  isolated copy so callers cannot mutate the index) and
  `StdlibCount()`.
- `runtime/stdlib_test.go`: tests that the index is non-empty,
  sorted, deduplicated, contains a critical set
  (`os`, `sys`, `json`, `math`, `re`, `asyncio`, `typing`,
  `collections`, `io`, `pathlib`), and that the public
  `StdlibModules()` copy is isolated from the underlying slice.
- `scripts/sync-stdlib-index.sh`: regenerates
  `runtime/stdlib_index.go` from `.deps/goipy/vm/asyncio.go`
  using awk on the `case "name":` switch. Sorted, deduplicated,
  with a "DO NOT EDIT" header.
- `.github/workflows/ci.yml` runs the generator into a tempdir
  and `diff -u`'s against the checked-in file on linux and
  macOS, failing the job loudly if a goipy bump landed without
  re-running the script.
- Four go tests in `cmd/bunpy/main_test.go`:
  `TestStdlibSubcommand`, `TestStdlibCount`, `TestStdlibHelp`,
  `TestStdlibUnknownMode`.

### Changed

- Sibling toolchain repos (gopapy, gocopy, goipy) now clone into
  `bunpy/.deps/` instead of the parent directory. The directory
  is `.gitignore`d. This keeps bunpy from disturbing local
  sibling clones the user may have at any merge state, and makes
  CI and local builds reproducible from one place.
- `go.work` points at `./.deps/{gocopy,goipy,gopapy}`.
- `scripts/sync-deps.sh` writes into `$ROOT/.deps/` and
  force-checks-out the pinned commits (so a dirty cached clone
  can never wedge a build).
- `docs/ROADMAP.md` marks v0.0.4 shipped, v0.0.5 next.
- `docs/CLI.md` adds the embedded-stdlib paragraph.
- `docs/ARCHITECTURE.md` documents `runtime/stdlib_index.go` and
  the drift-detection role of the CI step.
- `docs/COVERAGE.md` adds a `bunpy stdlib` row.
- The version banner in `bunpy --help` lists the five wired
  capabilities.

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

