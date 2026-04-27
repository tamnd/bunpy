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

## Module layout

```
cmd/bunpy/         CLI entry: subcommand router + per-command files
internal/manpages/ embedded roff manpages (man1/*.1) + Go accessors
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
