# Changelog

All notable changes to bunpy are recorded here. The format follows
[Keep a Changelog 1.1](https://keepachangelog.com/en/1.1.0/). Once
bunpy reaches 1.0 the project will follow
[Semantic Versioning](https://semver.org/spec/v2.0.0.html); until
then, expect minor version bumps to sometimes include breaking
changes.

## [Unreleased]

## [0.3.1] - 2026-04-27

See `changelog/v0.3.1.md` for the full entry.

## [0.3.0] - 2026-04-27

See `changelog/v0.3.0.md` for the full entry.

## [0.2.4] - 2026-04-27

See `changelog/v0.2.4.md` for the full entry.

## [0.2.3] - 2026-04-27

See `changelog/v0.2.3.md` for the full entry.

## [0.2.2] - 2026-04-27

See `changelog/v0.2.2.md` for the full entry.

## [0.2.1] - 2026-04-27

See `changelog/v0.2.1.md` for the full entry.

## [0.2.0] - 2026-04-27

See `changelog/v0.2.0.md` for the full entry.

## [0.1.3] - 2026-04-27

The package manager grows its first piece of porcelain. `bunpy
add <pkg>[<spec>]` reads `pyproject.toml`, fetches the project
from PyPI, picks the highest universal wheel satisfying the
caller's PEP 440 spec, installs it through the v0.1.2 wheel
installer, and writes the resolved spec back into
`[project].dependencies`. v0.1.3 is naive on purpose: no
transitive walk, no lockfile, no resolver. The lockfile lands at
v0.1.4; the PubGrub resolver and platform wheels at v0.1.5. This
rung exists so the porcelain is wired and later rungs only swap
algorithms in behind a stable surface.

### Added

- `pkg/version/`: a deliberate PEP 440 subset. `Spec` carries one
  or more `Clause{Op, Version}` entries; `ParseSpec` accepts bare
  `1.2.3`, `==`, `!=`, `>=`, `>`, `<=`, `<`, `~=`, comma-joined.
  Wildcards (`==1.2.*`) and arbitrary equality (`===`) are
  refused. `Compare` orders versions per PEP 440 release segment
  plus pre-release tier (`dev` < `pre` < final < `post`); local
  segments are kept verbatim and ignored. `Highest` skips
  pre-releases by default and falls back when nothing else
  matches.
- `pkg/manifest.AddDependency`: best-effort text rewriter that
  locates `[project].dependencies`, creates the array when
  absent, and inserts or replaces the line in PEP 503-normalised
  order. Comments outside the array survive verbatim. The
  Manifest now carries `Source []byte` so the rewriter can work
  off the original text.
- `cmd/bunpy/add.go`: `bunpy add <pkg>[<spec>]` with
  `--no-install`, `--no-write`, `--target <dir>`, `--index <url>`,
  `--cache-dir <path>`. Re-adding an already-listed package
  upgrades its line. The fetch path goes through `httpkit` and is
  swappable via `BUNPY_PYPI_FIXTURES` so the v0.1.3 smoke job
  stays offline.
- `internal/manpages/man1/bunpy-add.1`: roff page covering
  synopsis, options, environment, exit status.
- `helpRegistry` entry for `add`; `bunpy help` and `bunpy add
  --help` share one body.
- Tests: `pkg/version/version_test.go` (5 cases including
  ParseSpec, Compare numeric, Compare pre/post/dev, Match for
  every operator, Highest pre-release skipping, and the
  empty-Spec-matches-all invariant), `pkg/manifest/manifest_test.go`
  grows 7 new cases (append, sort, surrounding-comments preserve,
  array creation, pre-existing-section ordering, duplicate
  upgrade, normalised-name dedup), `cmd/bunpy/add_test.go` (7
  cases covering URL fetch, spec filtering, --no-install,
  --no-write, no-match exit, no-arg, --help).
- Fixtures: `tests/fixtures/v013/widget-1.0.0-py3-none-any.whl`
  and `widget-1.1.0-py3-none-any.whl` plus the matching simple
  index entry, frozen by `tests/fixtures/v013/build_widgets.go`.
  `tests/fixtures/v013/widget.add_in` drives the new harness
  handler that exercises `bunpy add` end-to-end against an
  isolated cwd.

### Changed

- `cmd/bunpy/main.go`: subcommand router gains `case "add"`; the
  unknown-command error message updates to v0.1.3.
- `cmd/bunpy/help.go`: `pm` body mentions the v0.1.3 surface.
- `tests/run.sh`: walks `tests/fixtures/v01*/*.add_in`, copies
  the input pyproject into a tempdir, runs `bunpy add` against
  the fixture index, and diffs the resulting manifest plus the
  installed file tree against frozen expectations.
- `.github/workflows/ci.yml`: smoke job adds a `bunpy add`
  round-trip against the v0.1.3 fixture root, asserting the
  wheel lands and the new line is in `pyproject.toml`. The
  help-parity loop now covers `add`.
- `docs/CLI.md`: `bunpy add` lands under Package manager; the
  wired-surface preamble updates to v0.1.3.
- `docs/ARCHITECTURE.md`: new "bunpy add" section covers
  `pkg/version`, `manifest.AddDependency`, and the
  manifest-fetch-pick-install-writeback flow.
- `docs/ROADMAP.md`: v0.1.3 marked shipped; v0.1.4 (`bunpy.lock`
  writer plus reader) next.
- `pkg/pypi/pypi.go`: `UserAgent` bumped to `bunpy/0.1.3`.

### Notes

- `bunpy add` only considers `py3-none-any` wheels. Platform
  wheels land with the resolver in v0.1.5; sdist installs (PEP
  517) stay out of v0.1.x.
- Re-adding the same package replaces its line. An exact-pin
  bump workflow (`bunpy update <pkg>`) lands with v0.1.7.
- The fetch path caches under
  `${BUNPY_CACHE_DIR or XDG default}/wheels/<name>/<filename>`;
  a re-run with the same wheel is offline.

## [0.1.2] - 2026-04-27

The package-manager band gets its filesystem primitive: a wheel
installer. v0.1.2 ships the Go `pkg/wheel` package, a wheel-body
disk cache, and a porcelain verb `bunpy pm install-wheel
<url|path>` that lays one wheel down into a target site-packages
directory per PEP 427. No resolution, no transitive walk, no
lockfile yet.

The exposed surface is narrow on purpose: purelib wheels only
(`Root-Is-Purelib: true`, no `*.data/` subdirs), RECORD hash
verification on by default, unsafe entries (zip-slip, absolute
paths, parent traversal) rejected before any byte hits disk, and
the install staged under a tempdir inside `--target` and renamed
file-by-file at the end. A mid-install crash leaves the existing
site-packages untouched.

### Added

- `pkg/wheel/`: `Open(path)` / `OpenReader(filename, body)` parse
  the wheel zip and dist-info/{WHEEL,METADATA,RECORD}.
  `(*Wheel).Install(target, opts)` does the actual install: it
  pre-flights the body for unsafe entries, verifies hashes (when
  `VerifyHashes` is on), stages files under a tempdir inside
  `target`, writes `INSTALLER`, re-emits `RECORD` with the
  install-side hashes and sizes, then renames the staged tree
  into place. `Wheel.Tags`, `Wheel.WHEEL`, and `Wheel.RECORD` are
  parsed structurally; `Wheel.Metadata` is kept verbatim and
  consumed by the v0.1.5 resolver.
- `pkg/cache/wheel.go`: `WheelCache{Dir}` with `Path`, `Has`,
  `Put`. Atomic writes via tempfile + rename, PEP 503 name
  normalisation on the project-name slot.
- `cmd/bunpy/pm.go`: `bunpy pm install-wheel <url|path>` with
  flags `--target <dir>` (default `./.bunpy/site-packages`),
  `--no-verify`, `--installer <name>` (default `bunpy`). URL
  fetches go through `httpkit.RoundTripper` so
  `BUNPY_PYPI_FIXTURES` redirects to a fixture root in tests;
  the body is cached under `<cache>/wheels/<name>/<filename>`.
- `internal/manpages/man1/bunpy-pm-install-wheel.1`: roff page
  covering synopsis, options, environment, exit status.
- `helpRegistry` entry for `pm-install-wheel`; the existing `pm`
  body lists all three verbs.
- Tests: `pkg/wheel/wheel_test.go` (10 cases including
  Open/parse, RECORD parse, purelib install round-trip,
  zip-slip rejection, absolute-path rejection, Root-Is-Purelib
  false rejection, .data subdir rejection, hash-mismatch
  rejection, --no-verify hatch, INSTALLER write),
  `pkg/cache/wheel_test.go` (round-trip + atomic-write check),
  plus 5 new cases in `cmd/bunpy/pm_test.go` covering local
  path, URL via fixture transport, missing file, missing arg,
  and help routing.
- Fixtures: `tests/fixtures/v012/tinypkg-0.1.0-py3-none-any.whl`
  is a frozen tiny wheel built once by
  `tests/fixtures/v012/build_tinypkg.go` and committed (RECORD
  hashes must stay byte-stable).
  `tests/fixtures/v012/index/files.example/tinypkg/` carries the
  same bytes for the URL-fetch path.

### Changed

- `internal/httpkit/fixtures.go`: the fixture transport now
  serves binary file URLs alongside the existing PEP 691
  directory URLs. A path ending in `/` resolves to
  `<root>/<host>/<path>/index.json` (unchanged); any other path
  resolves to a raw file at `<root>/<host>/<path>`. Wheel
  downloads use the second shape.
- `cmd/bunpy/main.go`: unknown-command error message updates to
  v0.1.2.
- `tests/run.sh`: walks `tests/fixtures/v01*/*.whl` and asserts
  `find <tempdir>` output matches `expected_<name>.txt`
  byte-for-byte.
- `.github/workflows/ci.yml`: smoke job adds a `bunpy pm
  install-wheel` round-trip (path + URL) and asserts the
  expected files land plus `INSTALLER` carries `bunpy`. The
  existing help-parity loop covers `pm-install-wheel`
  automatically via the registry iteration.
- `docs/CLI.md`: `bunpy pm install-wheel` lands under Package
  manager; the wired-surface preamble updates.
- `docs/ARCHITECTURE.md`: new "Wheel installer" section covers
  the offline-first surface, the install rules (purelib only,
  hash verify, atomic stage + rename), and the `WheelCache`
  layout.
- `docs/ROADMAP.md`: v0.1.2 marked shipped; v0.1.3 (`bunpy
  add`) next.

### Notes

- `*.data/` subdirs (purelib, platlib, scripts, headers, data),
  wheel signing (PEP 458), editable installs (PEP 660), and
  `.pyc` generation are explicitly out of scope. They land at
  the rungs the roadmap pins them to.
- The downloaded wheel cache is keyed under
  `${BUNPY_CACHE_DIR or XDG default}/wheels/<name>/<filename>`;
  a re-run with the same URL is offline.

## [0.1.1] - 2026-04-27

The package-manager band gets its network primitive. v0.1.1
lands the PEP 691 simple-index client, an ETag-keyed disk
cache, and a fixture transport that lets every later v0.1.x
test pin every byte of every PyPI exchange. CI never reaches
the live index in unit tests.

The exposed verb is `bunpy pm info <package>`. It fetches a
project page, parses every release artefact, and prints the
parsed view as JSON. Later rungs (`pm install-wheel`,
`bunpy add`, the resolver) consume the same `pypi.Project`
shape, so the network primitive only has to be written once.

### Added

- `pkg/pypi/`: `Client.Get(ctx, name)` returns a parsed PEP 691
  project page. The result carries `Name` (PEP 503 normalised),
  `Files` (filename, url, hashes, requires_python, yanked flag,
  version, kind), `Versions` (sorted unique), and `Meta`
  (api_version, last_serial, etag). Wheels and sdists are
  classified by extension; unknown filenames are still recorded
  so callers do not lose fidelity. `NotFoundError` is the typed
  error for 404s.
- `internal/httpkit/`: a tiny `RoundTripper` interface, a
  `Default(perHost)` constructor wrapping `*http.Client` with
  sane timeouts plus a per-host concurrency limiter, and a
  `FixturesFS(root)` transport that serves canned responses
  from a directory tree keyed by URL host plus path. The
  fixture transport handles `If-None-Match` so ETag-revalidation
  is part of the test contract.
- `pkg/cache/`: an ETag-keyed `Index` for PEP 691 pages.
  Atomic writes via tempfile plus rename; PEP 503 name
  normalisation everywhere a name touches disk so
  `Foo_Bar`, `foo-bar`, and `FOO.BAR` share one cache slot.
  `DefaultDir()` honours `XDG_CACHE_HOME`, falls back to
  `$HOME/Library/Caches/bunpy` on macOS and
  `%LOCALAPPDATA%\bunpy` on Windows; `BUNPY_CACHE_DIR`
  overrides everything.
- `cmd/bunpy/pm.go`: `bunpy pm info <package>` with flags
  `--no-cache`, `--index <url>`, `--cache-dir <path>`. The
  `BUNPY_PYPI_FIXTURES` env hook swaps the live transport
  for `httpkit.FixturesFS` so smokes and end-to-end fixtures
  stay offline. The `pm` plumbing dispatch grows the `info`
  verb alongside `config`.
- `internal/manpages/man1/bunpy-pm-info.1`: roff page covering
  synopsis, options, caching, environment, exit status.
- `helpRegistry` entry for `pm-info`; the existing `pm` body
  lists both verbs.
- Tests: `pkg/pypi/pypi_test.go` (8 cases covering name
  normalisation, kind classification, version extraction, page
  parsing, ETag revalidation through a recording transport,
  404 surfacing, invalid JSON), `pkg/cache/index_test.go` (5
  cases including a normalisation alias check and an
  atomic-write left-no-temp check), `internal/httpkit/fixtures_test.go`
  (4 cases including `If-None-Match` and status override),
  plus 6 new cases in `cmd/bunpy/pm_test.go` exercising the
  binary against a fixture root.

### Changed

- `cmd/bunpy/main.go`: unknown-command error message updates
  to v0.1.1.
- `tests/run.sh`: walks `tests/fixtures/v011/index/` for a
  fixture transport root and asserts `bunpy pm info widget`
  output matches `expected_widget.json` byte-for-byte.
- `tests/fixtures/v011/`: a frozen `widget` page plus the
  expected JSON output.
- `.github/workflows/ci.yml`: smoke job adds a `bunpy pm
  info` round-trip against a tempdir fixture root (asserts
  `demo + 1.0 + wheel` in the output and a non-zero exit on a
  404). Help-parity loop now covers `pm-info` automatically
  via the registry iteration.
- `docs/CLI.md`: `bunpy pm info` lands under Package manager;
  the wired-surface preamble updates.
- `docs/ARCHITECTURE.md`: new "PyPI client" section covers
  the offline-first transport contract, the cache layout, and
  the `Client.Get` surface that later rungs consume. Module
  layout grows `internal/httpkit/`.
- `docs/ROADMAP.md`: v0.1.1 marked shipped; v0.1.2 (wheel
  installer) next.

### Notes

- The metadata cache (`metadata/` subdir under the cache
  root) is intentionally not wired here; PubGrub is the
  caller that reads `.dist-info/METADATA`, and that lands
  with v0.1.5.
- Authentication, mirrors, and fall-back indexes are out of
  scope for v0.1.1. They live with workspaces in v0.2.x.
- A separate `live-pypi` workflow that exercises the real
  index against a tiny curated set lands as a follow-up; CI
  unit tests stay strictly offline.

## [0.1.0] - 2026-04-27

v0.1.0 opens the package-manager band. It is parser-only: no
network, no installs, no lockfile. The exposed surface is the
`pkg/manifest` Go package and one porcelain verb, `bunpy pm
config`, which prints the parsed `pyproject.toml` as JSON.

Every later v0.1.x rung consumes this same `Manifest` shape:
the resolver reads `Project.Dependencies`, the wheel installer
reads `Project.Name`, `bunpy add` writes back via `Project.Raw`.
Landing the parser first means the next eleven rungs never
fight over its shape.

### Added

- `pkg/manifest/`: PEP 621 `[project]` parser with all the
  modelled fields (name, version, description, requires-python,
  dependencies, optional-dependencies, authors, maintainers,
  license, readme, scripts, gui-scripts, entry-points, urls,
  keywords, classifiers, dynamic). The original table is kept
  in `Project.Raw` so `bunpy add` can round-trip it back to
  disk in v0.1.3. `[tool.bunpy]` is preserved verbatim under
  `Tool.Raw`; any other top-level table goes into
  `Manifest.Other` so unknown keys do not get dropped.
  `Load`, `LoadOpts`, `Parse`, `ParseOpts` cover the four
  call shapes; `LoadOptions{Strict: false}` collects validation
  failures as warnings instead of errors.
- `cmd/bunpy/pm.go`: `bunpy pm <verb>` plumbing tree. v0.1.0
  wires `bunpy pm config [path]`, which loads the manifest in
  strict mode and prints it as indented JSON. Strict mode
  rejects a missing `[project]` table, a missing or
  PEP 503-invalid `name`, and any `project.dynamic` entry that
  is also set literally (PEP 621 §5.4).
- `internal/manpages/man1/bunpy-pm.1` and
  `bunpy-pm-config.1`: roff manpages for the two new help
  topics.
- `helpRegistry` entries for `pm` and `pm-config`. The CI
  parity loop iterates the registry, so the new entries are
  covered automatically.
- Fixtures under `tests/fixtures/v010/`: `minimal`, `full`,
  and `tool-bunpy` pyproject + expected JSON triples.
  `tests/run.sh` grows a `*.pyproject.toml` paired with
  `expected_<name>.json` handler.
- Tests: `pkg/manifest/manifest_test.go` (15 cases covering
  minimal, full, license shorthand, readme shorthand, missing
  project, missing name, invalid names, valid names,
  dynamic-conflict, dynamic-no-conflict, tool.bunpy preserved,
  tool.<other> preserved, unknown top-level preserved, bad
  TOML) and `cmd/bunpy/pm_test.go` (9 cases covering JSON
  shape, default path, missing file, bad name, unknown flag,
  no verb, unknown verb, pm help routes, pm-config help).

### Changed

- `cmd/bunpy/main.go`: dispatch grows a `pm` case; the
  unknown-command error message updates to v0.1.0.
- `tests/run.sh`: walks `tests/fixtures/v01*/*.pyproject.toml`
  and compares `bunpy pm config <toml>` against
  `expected_<name>.json`.
- `.github/workflows/ci.yml`: smoke job adds a `bunpy pm
  config` round-trip (writes a temp pyproject, asserts the
  printed JSON contains the project name, version, and the
  `[tool.bunpy]` profile, and that a missing path exits
  non-zero); the help-parity loop now includes `pm`.
- `docs/CLI.md`: Package manager section is rewritten to lead
  with the wired `bunpy pm config` and demote the rest to the
  per-rung roadmap.
- `docs/ARCHITECTURE.md`: new "Manifest" section covers the
  three-slot `Manifest` shape (`Project`, `Tool`, `Other`),
  the strict-vs-soft validation split, and the contract that
  later rungs consume the same parser output.
- `docs/ROADMAP.md`: v0.1.x table grows the per-rung statuses;
  v0.1.0 is shipped, v0.1.1 (PyPI client) is next.

### Notes

- Validation is deliberately narrow. PEP 508 marker parsing
  lives under `pkg/marker/` and lands with the resolver in
  v0.1.5. PEP 621 metadata validation beyond name and
  dynamic-conflict is also out of scope here.
- `[build-system]` is parsed and preserved verbatim under
  `Manifest.Other`. Build-backend dispatch is a v0.6.x
  bundler concern.
- Writing `pyproject.toml` back to disk is not in v0.1.0.
  `bunpy add` needs that and lands in v0.1.3; until then
  `Project.Raw` is the round-trip handle.
- `github.com/BurntSushi/toml v1.6.0` is the new direct
  dependency. Pure Go, MIT, no transitive deps.

## [0.0.8] - 2026-04-27

`bunpy` no longer needs a script file to run. v0.0.8 lands the
interactive prompt: `bunpy repl` reads chunks of Python from
stdin, accumulates a multi-line buffer, and flushes through
`bunpy run` on every blank line.

The REPL is intentionally a line-driver, not a stateful
interpreter. Each flushed chunk is its own one-shot module:
globals do not survive across flushes. Persistent state will
ride alongside gocopy's expression and call support; the
shell's CLI surface is stable so the language story can grow
under it without breaking callers.

`:` is reserved as the meta-command prefix. `:help`, `:quit`,
`:exit`, `:history`, `:clear` drive the shell without ever
reaching the compiler. History is persisted at
`$HOME/.bunpy_history`, capped, and overridable via env.

### Added

- `bunpy repl` subcommand. Flags: `--quiet` (no banner, no
  prompts; for piped stdin and fixtures), `-h`/`--help`.
- `internal/repl/`: `Loop` (the state machine) and a small
  `history` (load, append, persist) helper. No third-party
  deps, no terminal raw mode in v0.0.8.
- `internal/manpages/man1/bunpy-repl.1`: roff manpage covering
  synopsis, options, meta commands, environment, exit status.
- `helpRegistry` entry for `repl` in `cmd/bunpy/help.go`. The
  parity smoke automatically asserts `bunpy help repl` matches
  `bunpy repl --help` byte-for-byte.
- `tests/fixtures/v008/`: `assign.repl_in` and
  `multiline.repl_in` plus matching empty `expected_*.txt`,
  exercising the flush path through `bunpy repl --quiet < input`.
- Tests: `internal/repl/repl_test.go` (12 cases covering quit,
  EOF, single and multi-line flush, error recovery, every
  meta command, prompt suppression, and the loud-mode banner)
  and `internal/repl/history_test.go` (round-trip, cap,
  size-zero disable, and a multiline-on-disk escape check).

### Changed

- `cmd/bunpy/main.go`: dispatch grows a `repl` case; the
  unknown-command error message updates to v0.0.8.
- `tests/run.sh`: walks `tests/fixtures/v00*/*.repl_in` and
  pipes each through `bunpy repl --quiet`, comparing stdout
  against `expected_<name>.txt`.
- `.github/workflows/ci.yml`: smoke job adds a `bunpy repl
  --quiet` round-trip and a banner check on the loud path; the
  help-parity loop now includes `repl`; the man-install smoke
  asserts `bunpy-repl.1` is materialised.
- `docs/CLI.md`, `docs/ARCHITECTURE.md`, `docs/ROADMAP.md`:
  REPL paragraph, layout entry, roadmap status flip.

### Notes

- v0.0.8 does not change runtime semantics: every flush is
  still a fresh `runtime.Run` call. The compiler still only
  accepts what gocopy supports today (assignments,
  no-op statements, docstrings). The REPL surfaces compile
  errors verbatim and recovers; the loop continues.
- Raw-mode line editing (arrow history, Ctrl-A/E) is
  deliberately out of scope. v0.7.x grows the runtime polish
  along with `--hot`, `--watch`, and friends.
- The history file format is one entry per line; embedded
  newlines are escaped as `\n` on disk and restored on load.

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

