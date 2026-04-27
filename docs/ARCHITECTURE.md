# Architecture

bunpy is one binary that does what Bun does for JavaScript,
applied to Python. It rolls four jobs into a single executable:

- A runtime that runs `.py` files via the Pure-Go Python
  toolchain (`gopapy` parser, `gocopy` compiler, `goipy`
  bytecode VM).
- A package manager that installs from PyPI, resolves with
  PubGrub over PEP 691 metadata, and locks to a stable
  text-readable `bunpy.lock`.
- A bundler that emits a `.pyz` zipapp by default, and a
  single static Go binary with `--compile`.
- A test runner that discovers pytest-shaped and unittest-shaped
  tests, exposes a Bun-flavoured matcher API, and runs in
  parallel with optional per-file isolation.

This page is the architecture summary. Per-feature design notes
land alongside each version's PR under `docs/`.

## Pipeline

```
.py source
  │
  ├── gopapy.ParseFile    →  ast.Module          (gopapy/v2)
  ├── gocopy.Compile      →  bytes (.pyc bytes)  (gocopy/v1)
  ├── goipy.Eval          →  Python objects      (goipy/v1)
  │
  ├── bunpy/runtime       →  registers bunpy.* and Web platform globals
  ├── bunpy/pkg           →  PyPI + resolver + lockfile + workspaces
  ├── bunpy/build         →  bundler + --compile single-binary emitter
  └── bunpy/test          →  discovery + parallel + isolate + coverage
```

## The gocopy / goipy bridge

bunpy compiles with gocopy and runs with goipy. Both speak the
same CPython 3.14 marshal format, but they live in separate Go
modules with separate type identities. To hand a code object
across, bunpy serializes it on one side and deserializes it on
the other:

```
source bytes
  -> gocopy.compiler.Compile        (*gocopy/bytecode.CodeObject)
  -> gocopy.marshal.Marshal         ([]byte, just the body)
  -> goipy.marshal.Unmarshal        (*goipy/object.Code)
  -> goipy.vm.Interp.Run            (Python objects)
```

The marshal hop is in-memory; nothing touches disk. When gocopy
and goipy unify under one module path the bridge collapses to a
direct hand-off, but the surface bunpy depends on stays the same.

## Distribution

Tagged releases produce six archives (linux/darwin/windows times
amd64/arm64) plus an aggregated `SHA256SUMS` file.
`install.sh` at the repo root is the one-liner installer for
linux and macOS: it resolves the latest tag, downloads the
matching `.tar.gz`, verifies the checksum from `SHA256SUMS`,
and drops the binary at `$HOME/.bunpy/bin/bunpy`. Re-running
upgrades in place; the prior binary is preserved as
`bunpy.prev` so rollbacks are one `mv` away. The Homebrew tap
at `tamnd/homebrew-bunpy` is updated by a `tap` job in
`release.yml`: `scripts/render-formula.sh` materialises
`Formula/bunpy.rb` from a template and the `SHA256SUMS` file,
and the workflow pushes the result with a fine-grained PAT.
The job is gated on the `BREW_TAP_TOKEN` secret; absent the
secret it skips so forks and contributors do not need it.

## Build metadata

Build-time metadata lives in `runtime/buildinfo.go`. Six string
package vars (`Version`, `Commit`, `BuildDate`, plus the three
sibling-toolchain commits `Goipy`, `Gocopy`, `Gopapy`) default to
empty or `"dev"` and are overwritten via `-ldflags "-X ..."`.
`scripts/build-ldflags.sh` is the single producer of that
ldflags string: it reads pinned commits from `scripts/sync-deps.sh`
so the in-binary toolchain commits cannot drift from the workspace
they were built against. The CLI consumes this via
`runtime.Build()` and exposes it as `bunpy version`,
`bunpy version --short`, and `bunpy version --json`. A dev build
prints just `bunpy dev` and hides commit/date/toolchain lines so
local binaries never claim a tag they do not have.

## Embedded stdlib

goipy ships ~184 Python stdlib modules baked into the binary.
bunpy mirrors that list in `runtime/stdlib_index.go`, generated
from goipy's module switch by `scripts/sync-stdlib-index.sh`.
The list is the only thing bunpy needs to know at build time:
the actual module bodies live inside goipy and are imported
through goipy's normal `__import__` machinery once gocopy lands
import statements. Until then, `bunpy stdlib` is the answer to
"what would `import X` find" without having to spin up a Python
program. CI re-runs the generator against a clean checkout and
diffs against the checked-in file so a goipy bump that adds or
removes a module fails CI loudly.

## Help and manpages

Subcommand help has one source of truth: the `helpRegistry` map
in `cmd/bunpy/help.go`. Each entry has a name, a one-line
summary, and a long-form body. The router uses the body for
both `bunpy help <cmd>` and `bunpy <cmd> --help`, so the two
surfaces cannot drift; the CI smoke job asserts byte equality
across both routes for every wired command.

The roff manpages are a separate artefact, embedded via
`internal/manpages` (a Go package whose only job is to host
`//go:embed man1/*.1`). `bunpy man <cmd>` writes the bytes
straight to stdout; `bunpy man --install <dir>` walks the
embedded FS and copies each page into `<dir>/man1/`. The
release workflow builds an ubuntu-host binary on every matrix
leg and runs `bunpy man --install` against the staged archive
so the linux and darwin tarballs ship `share/man/man1/*.1`
alongside the binary; `install.sh` and the Homebrew formula
both pick those up. Windows archives skip the manpages.

## REPL

`bunpy repl` is a thin line-driver around `runtime.Run`. Each
input chunk is accumulated until a blank line, then handed to
the same compile-marshal-eval pipeline that `bunpy run` uses.
v0.0.8 is stateless: each flush starts with a fresh module
globals dict. Persistent globals across chunks would need a
goipy entry point that takes a caller-supplied dict; that lands
once gocopy grows expression and call compilation and the use
case becomes meaningful.

The shell itself lives in `internal/repl/`. Line editing is
plain `bufio.Scanner`; raw-mode terminal editing (arrow keys,
Ctrl-A/E, completion) lands in v0.7.x. Meta commands prefixed
with `:` are reserved syntax (Python statements never start
with `:`) so they never conflict with user code.

## Manifest

The package manager begins at the parser. `pkg/manifest/` reads
`pyproject.toml` into a `Manifest` with three slots: a typed
`Project` (PEP 621), a `Tool` that holds `[tool.bunpy]` verbatim,
and an `Other` map that preserves anything else (`[build-system]`,
`[tool.ruff]`, ...) so we never lose fidelity for tools we do not
understand yet. `Project.Raw` keeps the original `[project]` table
so `bunpy add` can round-trip it back to disk in v0.1.3.

Validation is deliberately narrow: in strict mode we reject a
missing `[project]` table, a missing or PEP 503-invalid name, and
any `project.dynamic` entry that is also set literally (PEP 621
§5.4). Everything else is accepted as written; PEP 508 marker
parsing lives under `pkg/marker/` and lands with the resolver in
v0.1.5. Soft mode collects the same checks as warnings, for
callers (`bunpy pm config`) that want to surface issues without
hard-failing.

`bunpy pm config` is the porcelain on top: it loads the manifest
and prints it as indented JSON. Every later v0.1.x rung consumes
the same `manifest.Manifest` shape: the resolver reads
`Project.Dependencies`, the installer reads `Project.Name`,
`bunpy add` writes back via `Project.Raw`. One parser, many
callers.

## PyPI client

`pkg/pypi/` is the PEP 691 simple-index client. It consumes a
`httpkit.RoundTripper` (a tiny interface around `*http.Client`)
and, optionally, a `cache.Index`. Tests substitute the real
transport with `httpkit.FixturesFS(root)`, which serves canned
responses from a directory keyed by URL host plus path. The
binary surfaces this hook through `BUNPY_PYPI_FIXTURES` so
end-to-end fixtures and the `live-pypi` workflow share the same
client; only the transport differs. CI never reaches PyPI in
unit tests.

The cache is plain disk: `${XDG_CACHE_HOME}/bunpy/index/<name>/page.json`
plus a sibling `etag` file, written atomically via tempfile +
rename. `Client.Get` issues `If-None-Match` when an ETag is on
disk; a 304 returns the cached body parsed identically. PEP 503
name normalisation runs everywhere a name touches the network
or the cache, so `Foo_Bar`, `foo-bar`, and `FOO.BAR` all share a
single cache slot.

`bunpy pm info <pkg>` is the porcelain on top: it builds a
default client, swaps in the fixture transport when the env hook
is set, fetches the page, and prints the parsed `pypi.Project`
as JSON. Later v0.1.x rungs (`bunpy add`, the resolver) consume
the same `Project` shape.

## Wheel installer

`pkg/wheel/` is the PEP 427 installer. `wheel.Open(path)` (or
`OpenReader(filename, body)`) reads a `.whl`, parses
`<dist>.dist-info/{WHEEL,METADATA,RECORD}`, and returns a `Wheel`
struct. `(*Wheel).Install(target, opts)` writes the body files
into `target` (typically `./.bunpy/site-packages/`), re-emits
RECORD with the install-side hashes and sizes, and writes
`INSTALLER`.

The v0.1.2 surface is deliberately narrow:

- `Root-Is-Purelib: true` only; `false` is refused.
- No `*.data/` subdirs (purelib, platlib, scripts, headers,
  data); these land when a real wheel forces it.
- Unsafe entries (zip-slip, absolute paths, parent traversal,
  backslashes) are rejected before any byte hits disk.
- RECORD hashes are verified on every install; a mismatch aborts
  before any partial write. `--no-verify` opts out.
- The install is staged under a tempdir inside `target` and
  renamed file-by-file at the end; a mid-install crash leaves
  the existing site-packages untouched.

`bunpy pm install-wheel <url|path>` is the porcelain: a `.whl`
path is read straight off disk, an `https://` URL goes through
`httpkit.RoundTripper` (so `BUNPY_PYPI_FIXTURES` redirects to a
fixture root in tests) and is cached under
`${BUNPY_CACHE_DIR or XDG default}/wheels/<name>/<filename>`.

`pkg/cache.WheelCache` mirrors the index cache: atomic write via
tempfile + rename, PEP 503 normalisation on the project-name
slot so `Foo_Bar` and `foo-bar` share one cache key.

## bunpy add

`pkg/version/` is a deliberate PEP 440 subset: `==`, `!=`, `>=`,
`>`, `<=`, `<`, `~=`, comma-joined, with pre-release ordering
(`a`, `b`, `rc`, `dev`, `post`). Wildcards (`==1.2.*`) and
arbitrary equality (`===`) are out of scope. `Highest` skips
pre-releases by default unless the caller's spec pins one, with a
graceful fallback to pre-releases when nothing else matches.

`pkg/manifest.AddDependency` is a best-effort text rewriter: it
locates the `[project]` section, finds `dependencies = [...]`
(creating the array when absent), and inserts or replaces the
matching line in PEP 503-normalised order. Comments outside the
array survive verbatim because we only touch the byte range
between `[` and `]`. The whole file is otherwise left alone.

`bunpy add <pkg>[<spec>]` glues the three pieces together:

1. Load `./pyproject.toml` in strict mode so a broken file fails
   fast before any network.
2. Build a `pypiRegistry` over `pypi.Client` (live, ETag-cached,
   or `BUNPY_PYPI_FIXTURES` for tests), the host wheel-tag set
   (`wheel.HostTags`), and the host marker environment
   (`marker.DefaultEnv`).
3. Hand the requirement to `resolver.Solver`. The solver walks
   transitive Requires-Dist edges, evaluates PEP 508 markers, and
   asks the registry for platform wheels via `wheel.Pick` against
   the host tag ladder.
4. For each pin, download via `httpkit` (cache-first, atomic) and
   install via the v0.1.2 wheel installer.
5. Re-write `pyproject.toml` via `manifest.AddDependency` and
   upsert every pin in `bunpy.lock`. The line written into
   `[project].dependencies` is the caller's spec verbatim, or
   `name>=resolved-version` when the caller gave no spec.

`bunpy install` is the read-side companion: it walks the existing
lockfile (no resolver, no marker pass) and installs every pin.
The porcelain surface is fixed; later rungs swap algorithms in
behind it without breaking callers.

## Lockfile (`pkg/lockfile`, `bunpy.lock`)

`pkg/lockfile` reads and writes `bunpy.lock`, the byte-stable
freeze of every dependency `bunpy add` and `bunpy pm lock`
resolve. The schema is version 1 and the on-disk format is a
small TOML subset emitted by a custom serialiser so rewrites are
deterministic regardless of host TOML library version:

- A header with `version`, `generated` (RFC3339, UTC), and
  `content-hash` (`sha256:<hex>` of the sorted, trimmed dep specs
  joined by `\n`).
- One `[[package]]` row per direct dependency with `name`,
  `version`, `filename`, `url`, `hash` (`sha256:<hex>` from the
  PEP 691 page).
- Rows are sorted by PEP 503-normalised name. Line endings are
  LF. Empty/zero values still emit the key so the shape stays
  stable.

`bunpy add` upserts the resolved row by normalised name and
recomputes the content-hash from the post-edit
`[project].dependencies`. `bunpy pm lock` walks the manifest from
scratch; `--check` re-reads the lockfile, compares the
content-hash against pyproject's, and verifies that every
lockfile entry is still listed in `[project].dependencies`.
Either drift exits non-zero so CI can catch a stale lockfile
without a network round-trip.

v0.1.5 fills transitive entries against the same `[[package]]`
shape; the schema does not bump. `--check` re-reads the lockfile,
compares the content-hash against pyproject's, and verifies that
every direct dep in `[project].dependencies` has a matching pin.
Transitive rows are expected and never trigger a drift.

v0.1.6 grows each `[[package]]` row an optional `lanes` array
listing every lane the pin belongs to. Lane labels are `main`,
`dev`, `group:<name>` (non-dev PEP 735 groups), `optional:<group>`
(PEP 621 optional-dependencies), and `peer`
(`[tool.bunpy].peer-dependencies`). Rows that only belong to `main`
omit the field so v0.1.5 fixtures stay byte-identical and the
schema version stays at 1. The content-hash now covers every lane,
sorted, so adding a new optional group triggers drift on
`--check`. `bunpy pm lock` resolves every lane in one pass and
post-processes the registry to compute per-lane closures via BFS
over Requires-Dist edges; `bunpy install` filters by the per-pin
`lanes` tag (default keeps `main` only).

v0.1.7 adds `bunpy outdated` and `bunpy update` against the same
schema. Neither verb bumps the lockfile version. `outdated` is
read-only: it walks the lockfile, fetches each pin's PEP 691
page, and prints `current / wanted / latest` per row.
`update` re-runs the resolver with a new
`Solver.Locked map[string]string` field that biases the
candidate-pick step toward the locked version when the manifest
spec still allows it. A bare `bunpy update` clears the whole map
so any pin can move within its spec; naming packages on the
command line drops only those entries. `--latest <pkg>...`
strips the manifest spec for the named packages before resolving
(the on-disk manifest is not edited; only the in-memory lane map
fed to the solver). The new lockfile is written and, unless
`--no-install` is passed, the install path is the same as
`bunpy install` with the same lane filter.

v0.1.8 lands `bunpy remove` as the symmetric inverse of
`bunpy add`. The manifest editor (`pkg/manifest`) grows
`RemoveDependency`, `RemoveOptionalDependency`,
`RemoveGroupDependency`, `RemovePeerDependency`, and the
all-lanes umbrella `RemoveDependencyAllLanes`. Each mutator
returns `(out []byte, n int, err)` so callers can re-Parse only
when bytes actually moved. PEP 503 normalisation matches `Foo_Bar`
and `foo-bar` even though only one form appears in the source;
removing the last entry in a multiline array preserves the array
shape (`dependencies = [\n]`) so the diff stays small. After the
manifest edit the verb seeds `Solver.Locked` with every surviving
lockfile pin minus the names being removed, re-resolves, and
rewrites `bunpy.lock` so any pin that loses every root falls off.
The uninstall path walks each dropped pin's
`<name>-<version>.dist-info/RECORD`, removing every listed path
(rejecting any that escape the target via a
`strings.HasPrefix(cleaned, abs+sep)` guard) before unlinking the
dist-info; a best-effort fallback removes the top-level package
directory when the wheel was a purelib without a RECORD. The
verb is idempotent: removing a name that is not listed is a
no-op (`removed 0 packages`, exit 0).

v0.1.9 lands `bunpy link` and `bunpy unlink`, the Bun-style pair
for editable installs. `pkg/links` is the tiny store: `Entry`
(name, version, source, registered) is JSON, one file per name
under `Dir()` (the registry root). `Dir` resolves
`$BUNPY_LINK_DIR` first, then the platform user-data dir
(`$XDG_DATA_HOME/bunpy/links` on Linux,
`~/Library/Application Support/bunpy/links` on macOS,
`%LOCALAPPDATA%/bunpy/links` on Windows). Writes are atomic
(tempfile + rename), and `Read` returns a typed `ErrNotFound` so
callers can distinguish "no entry" from "broken JSON". `pkg/editable`
is the consumer-side proxy: `Install` lays down a `.pth` file
holding the absolute source path plus a `<name>-<version>.dist-info`
directory with `METADATA`, `RECORD`, `INSTALLER=bunpy-link`, and
`direct_url.json` (PEP 610, `dir_info.editable=true`); `Uninstall`
reads RECORD with the same path-escape guard `bunpy remove` uses
and drops every listed path before removing the dist-info. The
`INSTALLER=bunpy-link` tag is the opt-out signal: `bunpy install`
checks each pin's installed dist-info before re-installing and
skips the wheel install when the tag matches, printing
`kept linked <name> <version>`. So the workflow is symmetric: the
publisher runs `bunpy link` from the package source to register;
the consumer runs `bunpy link <pkg>` to drop in the proxy; either
side can drop with `bunpy unlink` and the other side keeps
working until it re-installs or re-links.

v0.1.10 lands `bunpy patch` and the patch-apply step inside
`bunpy install`. `pkg/patches` holds the diff/apply pair plus the
wheel-extract helper. The diff is whole-file hunks (one hunk per
changed file): pristine and scratch are walked, files identical
in both trees are skipped, files that differ get a `--- a/<rel>`
/ `+++ b/<rel>` header followed by `@@ -1,N +1,M @@` and the full
old/new line lists. Adds and removes use `/dev/null` on the
appropriate header. Apply is strict: pristine context must match
the target byte-for-byte, no fuzz, no offset slack. Binary files
are refused at diff time. The patch table lives in
`pyproject.toml` under `[tool.bunpy.patches]` with key shape
`<name>@<version>` and a path relative to the project root;
`pkg/manifest` grows `AddPatchEntry` and `RemovePatchEntry` text
mutators that mirror the v0.1.6 lane mutators. `bunpy install`
reads the table once before the install loop, applies the
matching patch in place after every successful wheel install, and
rewrites `INSTALLER` to `bunpy-patch`. Linked packages
(`INSTALLER=bunpy-link`) take precedence: the install loop skips
them entirely so a `bunpy link <pkg>` survives unrelated install
runs even when the pin has a registered patch. The pristine tree
under `.bunpy/patches/.pristine/<name>-<version>` and the scratch
under `.bunpy/patches/.scratch/<name>-<version>` are throwaway
state — the user-visible artefact is `./patches/<name>+<version>.patch`,
which is the input to the resolver-independent reproducible
install.

v0.2.0 lands workspaces. `pkg/workspace` loads a root
`pyproject.toml` that has a `[tool.bunpy.workspace]` table, expands
glob patterns in the `members` list, and returns a `Workspace` struct
with one `Member` per subdirectory. `FindRoot` walks up the directory
tree so all verbs find the workspace root automatically from any
member subdirectory. The lockfile schema gains an optional
`[workspace]` section (`members = [...]`) that records the member
paths present when the lock was written. Single-project locks have no
`[workspace]` section and are fully forward-compatible.
`manifest.Tool.Workspace` surfaces the `[tool.bunpy.workspace]` table
to callers without re-parsing TOML. `bunpy install` and `bunpy add`
both detect the workspace root via `FindRoot` and use the workspace
root's `bunpy.lock`; `bunpy add --member <name>` targets a specific
member's `pyproject.toml`.

v0.1.11 lands `bunpy why <pkg>`, the closing rung of the v0.1.x
package manager. `pkg/why` builds a forward dependency graph
from `bunpy.lock` plus per-pin Requires-Dist (read out of the
wheel cache via `wheel.LoadMetadata` / `wheel.ParseMetadata`,
markers evaluated against `marker.DefaultEnv`), then inverts it.
`Walk` is a depth-first enumeration of every path from the
queried pin upward, terminating at a virtual `@project` edge
tagged with the lane the requirement was declared in. Cycles are
guarded by per-path visited sets; the same pin can appear in
multiple chains via different parents (a diamond). Direct-dep
lane membership comes from the manifest's
`[project].dependencies`,
`[project.optional-dependencies]`,
`[dependency-groups]`, and `[tool.bunpy].peer-dependencies`.
Overlay state (`bunpy link`, `bunpy patch`) is harvested from
`[tool.bunpy.links]` and `[tool.bunpy.patches]` so the result
surfaces `Linked` / `Patched` flags and an `Installer` label
(`bunpy-link`, `bunpy-patch`) next to the pin in stdout and in
`--json`. The lockfile schema is unchanged: edges are derived,
not stored, which keeps v0.1.x lockfiles forward-compatible with
the rest of the verbs.

## Module layout

```
cmd/bunpy/         CLI entry: subcommand router + per-command files
internal/httpkit/  RoundTripper, per-host limiter, fixture transport
internal/manpages/ embedded roff manpages (man1/*.1) + Go accessors
internal/repl/     interactive line-driver shell (Loop, history)
runtime/           embeds goipy.VM; module loader; hot reload; env
api/               bunpy.* Python-side API written in Go
pkg/               package manager (resolver, wheel install, lock)
build/             bundler + --compile + plugins + macros
test/              bunpy test runner (discovery + parallel + isolate)
repl/              native REPL on goipy
server/            bunpy run --hot dev server + Markdown rendering
internal/          manifest, marker, cache, lockio, pyclink, platform
tests/             fixtures, corpus, end-to-end run.sh
docs/              these pages
changelog/         per-version release notes; rolled into CHANGELOG.md
scripts/           build-changelog.sh, feature-coverage.sh
.github/workflows/ ci.yml, build.yml, release.yml
```

The detailed pipeline (HTTP server, SQL, Redis, S3, shell, cron,
fetch, WebSocket, FFI, Worker, and the rest) is documented per
feature in `docs/API.md`.
