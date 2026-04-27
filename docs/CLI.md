# CLI reference

bunpy ships as one binary. Subcommands land per-version per the
roadmap. Today (v0.1.5) the wired surface is `--version` (with
`--short` and `--json`), `--help`, positional `bunpy <file.py>`,
`bunpy run <file.py>`, `bunpy repl`, `bunpy stdlib`,
`bunpy pm config`, `bunpy pm info`, `bunpy pm install-wheel`,
`bunpy pm lock`, `bunpy add`, `bunpy install`, `bunpy help`, and
`bunpy man`. This page is the long-form reference. Running
`bunpy help <cmd>` gives the same body inline; `bunpy man <cmd>`
prints the bundled roff manpage. Installing the binary itself:
see `docs/INSTALL.md`.

## Synopsis

```
bunpy <command> [args]
bunpy --version
bunpy --help
```

A bare positional `.py` argument is shorthand for `bunpy run
<file>`. So `bunpy app.py` and `bunpy run app.py` mean the same
thing.

## Commands

### Runtime

- `bunpy run <file.py>` runs a Python script. The bare positional
  form `bunpy <file.py>` is the same call. Flags planned: `--hot`
  (state-preserving reload), `--watch` (full restart on change),
  `-` (read from stdin), `--no-globals` (do not inject Web
  platform globals). v0.0.3 wires the basic file path; the flags
  follow on the v0.7.x rungs.
- `bunpy repl` opens the interactive Python prompt. v0.0.8 ships
  a stateless line-driver: each chunk is read until a blank line
  and handed to `bunpy run` as a one-shot module. Persistent
  globals across chunks waits for gocopy to grow expression and
  call compilation; the CLI surface is stable and will not change
  when that lands. `--quiet` suppresses the banner and prompts
  for piped stdin. Meta commands prefixed with `:` (`:help`,
  `:quit`, `:history`, `:clear`) drive the shell. History is
  persisted at `$HOME/.bunpy_history`; override with
  `BUNPY_HISTORY`, cap with `BUNPY_HISTORY_SIZE` (default 1000,
  `0` disables).

### Package manager

The `pm` tree groups low-level plumbing; porcelain commands
(`add`, `install`, `remove`, ...) live at the top level.

- `bunpy pm config [path]` parses `pyproject.toml` (default
  `./pyproject.toml`) and prints it as JSON. The output has three
  top-level keys: `project` (PEP 621 fields), `tool` (the
  `[tool.bunpy]` table, kept verbatim), and `other` (any
  unrecognised top-level table, kept verbatim so callers do not
  lose fidelity). Strict mode rejects a missing `[project]` table,
  a missing or PEP 503-invalid `name`, and any
  `project.dynamic` entry that is also set literally.
- `bunpy pm info <package>` fetches a project's PEP 691 simple
  index page and prints the parsed metadata as JSON: `name`
  (PEP 503 normalised), `versions` (sorted), `files` (one entry
  per release artefact with filename, url, hashes,
  requires_python, yanked flag, kind), and `meta` (api_version,
  last_serial, etag). Responses are ETag-cached on disk under
  the bunpy cache root; a second call uses `If-None-Match` so a
  304 turns into a cache hit. Flags: `--no-cache`, `--index
  <url>`, `--cache-dir <path>`. Tests pin every byte of every
  PyPI exchange via `BUNPY_PYPI_FIXTURES`; CI never reaches the
  live index.
- `bunpy pm install-wheel <url|path>` installs one wheel into a
  target site-packages directory (default
  `./.bunpy/site-packages/`) per PEP 427. Flags: `--target
  <dir>`, `--no-verify`, `--installer <name>`. v0.1.2 supports
  purelib wheels only (`Root-Is-Purelib: true`, no `*.data/`
  subdirs). RECORD hashes are verified before any byte hits
  disk; unsafe entries (zip-slip, absolute paths, parent
  traversal) are rejected at the entry level. The install is
  staged under a tempdir inside `--target` and renamed into
  place, so a mid-install crash leaves the existing tree intact.
  URL fetches go through the same `httpkit` transport `pm info`
  uses, so `BUNPY_PYPI_FIXTURES` redirects fetches in tests.
- `bunpy pm lock` (re)generates `bunpy.lock` from `pyproject.toml`
  without installing. v0.1.5 hands every direct dep to the
  PubGrub-inspired resolver, walks transitive Requires-Dist edges,
  evaluates PEP 508 markers against the host environment, and
  picks platform wheels (manylinux, musllinux, macosx, win) before
  writing each `[[package]]` row. The header carries a
  `content-hash` derived from the sorted, trimmed dep specs joined
  by newlines, so a cheap byte compare detects pyproject drift
  without a re-resolve. Flags: `--check` (verify and exit non-zero
  on drift; missing lockfile, content-hash mismatch, or a
  pyproject dep with no lockfile entry), `--index <url>`,
  `--cache-dir <path>`.

The porcelain `bunpy add <pkg>[<spec>]` hands the requirement to
the resolver, walks transitive edges, and writes the resolved spec
back into `[project].dependencies`. Flags: `--no-install` (manifest
+ lockfile only), `--no-write` (install only), `--target <dir>`,
`--index <url>`, `--cache-dir <path>`. Re-adding an already-listed
package replaces its line. Pre-releases are skipped unless the
spec pins one. Every successful add rewrites `bunpy.lock` with the
full transitive set; `--no-write` suppresses both files. Dev and
optional lanes (`-D`, `-O`, `-P`) land at v0.1.6.

`bunpy install` walks `bunpy.lock` (treated as the source of
truth, no re-resolve) and installs every pin into
`./.bunpy/site-packages/` via the same wheel installer
`pm install-wheel` uses. Flags: `--target <dir>`, `--cache-dir
<path>`, `--no-verify`. Run `bunpy pm lock` first when the
lockfile is missing or stale.

The rest of the package-manager surface is aspirational and
lands per the v0.1.x ladder in `docs/ROADMAP.md`:

- `bunpy remove <pkg>` removes a dependency.
- `bunpy update [pkg]` updates one or all packages.
- `bunpy outdated [pkg]` lists packages with newer versions.
- `bunpy audit [--fix]` checks for security advisories.
- `bunpy link [pkg]` and `bunpy unlink [pkg]` do editable
  installs.
- `bunpy patch <pkg>` and `bunpy patch --commit <hash>` persist
  local diffs against installed packages.
- `bunpy publish` builds an sdist plus a wheel and uploads them
  to PyPI.
- `bunpy why <pkg>` prints a reverse-deps tree explaining why a
  package is in the lockfile.
- `bunpy pm cache rm` clears on-disk caches.
- `bunpy pm ls` lists installed packages.
- `bunpy pm hash` prints the lockfile content hash.

### Project scaffolding

- `bunpy init` scaffolds `pyproject.toml`, the src layout, and a
  README.
- `bunpy create <template>` scaffolds from a template (fastapi,
  flask, click, lib, ml).
- `bunpyx <pkg>[@version] [args]` does a one-shot run from PyPI.

### Bundler

- `bunpy build [<entry.py>]` bundles to a `.pyz` (default).
- `bunpy build --compile` bundles into a single static Go binary.
- `bunpy build --target <triple>` produces a cross-target bundle.
- `bunpy build --plugins <list>` runs with bundler plugins.

### Test runner

- `bunpy test [path]` discovers and runs tests. Flags include
  `--parallel[=N]`, `--isolate`, `--shard=I/N`, `--changed`,
  `--coverage[=html|json|lcov]`, `--watch`,
  `--update-snapshots`, `--bail[=N]`, `--timeout=<ms>`.

### Workspace selectors

- `bunpy --filter <selector> <command>` runs a command in a
  matching workspace subset.

### Tooling passthrough

- `bunpy fmt [path]` formats Python source via gopapy.
- `bunpy check [path]` lints Python source via gopapy.

### Embedded stdlib

- `bunpy stdlib` (alias `bunpy stdlib ls`) prints, one per line,
  every Python stdlib module embedded in this binary by goipy.
  The list is baked at build time from goipy's module switch and
  regenerated by `scripts/sync-stdlib-index.sh` whenever the
  goipy pin moves. CI runs the same script into a tempdir and
  diffs against the checked-in index, so drift is caught before
  release.
- `bunpy stdlib count` prints the number of embedded modules.
  Handy for shell scripts that want to gate on stdlib coverage.
- `bunpy stdlib --help` prints the stdlib-scoped help.

### Meta

- `bunpy version` (alias `--version`, `-v`) prints the version,
  commit, build date, host go/os/arch, and pinned toolchain
  commits. A locally-built binary prints just `bunpy dev` plus
  the go/os/arch line so dev builds cannot lie about identity.
- `bunpy version --short` prints just the version string. Useful
  for shell scripts that want to gate on it.
- `bunpy version --json` prints the same metadata as a one-line
  JSON object. Fields: `version`, `commit`, `build_date`,
  `goipy`, `gocopy`, `gopapy`, `go`, `os`, `arch`. Empty string
  fields are omitted.
- `bunpy help` prints the top-level usage. `bunpy help <cmd>`
  prints the same long-form body that `bunpy <cmd> --help`
  prints. The two surfaces share a single source of truth (the
  `helpRegistry` map in `cmd/bunpy/help.go`) so they cannot
  drift; CI asserts byte equality on every push.
- `bunpy man <cmd>` prints the bundled roff manpage to stdout.
  Pipe to `man -l -` to render: `bunpy man run | man -l -`.
- `bunpy man --install [dir]` copies the embedded manpages into
  `<dir>/man1/`. The default `dir` is `$HOME/.bunpy/share/man`,
  matching where `install.sh` puts them. Add the parent to
  `MANPATH` to wire `man bunpy` and `man bunpy-run`.
