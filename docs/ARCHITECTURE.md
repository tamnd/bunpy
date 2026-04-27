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
