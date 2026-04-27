# CLI reference

bunpy ships as one binary. Subcommands land per-version per the
roadmap. Today (v0.2.4) the wired surface is `--version` (with
`--short` and `--json`), `--help`, positional `bunpy <file.py>`,
`bunpy run <file.py>`, `bunpy repl`, `bunpy stdlib`,
`bunpy pm config`, `bunpy pm info`, `bunpy pm install-wheel`,
`bunpy pm lock`, `bunpy add`, `bunpy install`, `bunpy outdated`,
`bunpy update`, `bunpy remove`, `bunpy link`, `bunpy unlink`,
`bunpy patch`, `bunpy why`, `bunpy workspace`, `bunpy audit`,
`bunpy publish`, `bunpy create`, `bunpy help`, and `bunpy man`.
The companion binary `bunpyx` runs packages from PyPI without installing
them permanently.
This page is the
long-form reference. Running
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
  writing each `[[package]]` row. v0.1.6 resolves every lane
  (`[project].dependencies`, `[project.optional-dependencies]`,
  `[dependency-groups]`, `[tool.bunpy].peer-dependencies`) in one
  pass and tags each pin with the lanes that pulled it in. Rows
  that only belong to `main` omit the `lanes` field for stability
  with v0.1.5 lockfiles. The header carries a `content-hash`
  derived from every lane's sorted, trimmed dep specs, so a cheap
  byte compare detects pyproject drift without a re-resolve. Flags:
  `--check` (verify and exit non-zero on drift; missing lockfile,
  content-hash mismatch, or a direct dep in any lane with no
  lockfile entry), `--index <url>`, `--cache-dir <path>`.

The porcelain `bunpy add <pkg>[<spec>]` hands the requirement to
the resolver, walks transitive edges, and writes the resolved spec
back into the matching manifest table. Flags: `-D`/`--dev` (PEP 735
`[dependency-groups].dev`, or `--group <name>` for a non-dev group),
`-O <group>`/`--optional <group>` (PEP 621
`[project.optional-dependencies].<group>`), `-P`/`--peer`
(`[tool.bunpy].peer-dependencies`), `--no-install` (manifest +
lockfile only), `--no-write` (install only), `--target <dir>`,
`--index <url>`, `--cache-dir <path>`. The lane flags are mutually
exclusive. Re-adding an already-listed package replaces its line.
Pre-releases are skipped unless the spec pins one. Every successful
add rewrites `bunpy.lock` with the full transitive set and tags the
new pin with its lane; `--no-write` suppresses both files.

`bunpy install` walks `bunpy.lock` (treated as the source of
truth, no re-resolve) and installs pins into
`./.bunpy/site-packages/` via the same wheel installer
`pm install-wheel` uses. Lane flags filter the install: the default
keeps only pins tagged `main` (and pins with no `lanes` field).
`-D`/`--dev` adds dev and `group:<name>` lanes; `-O <group>` adds
one `optional:<group>` (may be repeated); `--all-extras` adds every
optional group; `-P`/`--peer` adds `peer`. `--production` is an
alias for the default and is mutually exclusive with the lane
flags (Bun parity). Flags: `--target <dir>`, `--cache-dir <path>`,
`--no-verify`. Run `bunpy pm lock` first when the lockfile is
missing or stale.

`bunpy outdated [pkg]...` (v0.1.7) walks `bunpy.lock` and, for
each pin selected by the lane filters, fetches the project's
PEP 691 page through the same `httpkit` client `pm info` uses.
The output is a four-column table: `current` (lockfile pin),
`wanted` (highest version satisfying the manifest spec, the one
`bunpy update` would pick), `latest` (highest non-yanked,
wheel-bearing version on the index, the one
`bunpy update --latest` would pick), and `lanes`. Read-only:
no manifest, lockfile, or `site-packages` writes. `--json`
emits `{"outdated":[{name, current, wanted, latest, lanes},
...]}` for scripts. Lane flags mirror `install`. Exit status
is 0 even when pins are outdated.

`bunpy update [pkg]...` (v0.1.7) re-runs the resolver against
`pyproject.toml` with `Solver.Locked` seeded from the existing
lockfile. A bare `update` clears every lock and lets minor /
patch upgrades flow in within the manifest spec. Naming
packages on the command line drops only those entries from the
lock hint, so peers stay put. `--latest <pkg>...` strips the
manifest spec for the named packages and picks the highest
non-prerelease wheel; refused without a positional arg to
avoid surprise mass upgrades. After resolving, the new lockfile
is written, each changed pin prints as `name old -> new`, and
unless `--no-install` is set, `./.bunpy/site-packages` is
refreshed via the same install path `bunpy install` uses (with
the same lane filter). Other flags: `--target <dir>`,
`--cache-dir <path>`, `--index <url>`, `--no-verify`,
`--production`.

`bunpy remove <pkg>...` (v0.1.8) is the inverse of `bunpy add`.
A bare `bunpy remove <pkg>` deletes the named package from every
lane it appears in (`[project].dependencies`, every PEP 735
`[dependency-groups]`, every PEP 621
`[project.optional-dependencies]`, and
`[tool.bunpy].peer-dependencies`). Lane flags `-D`/`--dev`,
`-O <group>`/`--optional <group>`, and `-P`/`--peer` restrict
the delete to one lane (mutually exclusive); `--group <name>`
requires `-D` and picks one non-default group. After the
manifest edit, `bunpy.lock` is rewritten via the same resolver
path `bunpy update` uses, with `Solver.Locked` seeded from the
surviving pins minus the named packages; pins that lose every
root fall off the lockfile. Unless `--no-install` is set, the
dropped pins are removed from `./.bunpy/site-packages` via a
RECORD walk, with a best-effort fallback to `<name>/`
directory cleanup. `--no-write` skips the manifest edit. The
verb is idempotent: removing a name that is not listed prints
`removed 0 packages` and exits 0.

`bunpy link` and `bunpy unlink` (v0.1.9) are the Bun-style pair
for editable installs. A bare `bunpy link` reads the current
project's `pyproject.toml` and writes a registry entry under
`$BUNPY_LINK_DIR` (default: the platform user-data dir, e.g.
`$XDG_DATA_HOME/bunpy/links` on Linux,
`~/Library/Application Support/bunpy/links` on macOS,
`%LOCALAPPDATA%/bunpy/links` on Windows). The entry is a JSON
file `<name>.json` with `name`, `version`, `source` (absolute,
symlink-resolved path), and `registered` (timestamp). `--list`
prints the registry as a sorted table. `bunpy link <pkg>...` looks
up each name in the registry and lays down a PEP 660-style
editable proxy in `./.bunpy/site-packages`: a `.pth` file
holding the absolute source path plus a `<name>-<version>.dist-info`
directory with `METADATA`, `RECORD`, `INSTALLER=bunpy-link`, and
`direct_url.json` (PEP 610, `dir_info.editable=true`). `bunpy
install` recognises `INSTALLER=bunpy-link` and skips re-installing
linked packages, printing `kept linked <name> <version>` instead.
`bunpy unlink` mirrors the verb: bare `unlink` deletes the registry
entry for the current project, `bunpy unlink <pkg>...` walks the
proxy's `RECORD` (with the same path-escape guard `bunpy remove`
uses), removes every listed file, and drops the dist-info. Missing
entries are not an error. Flags: `--target <dir>` (consumer-side
site-packages root, default `./.bunpy/site-packages`).

`bunpy patch` and `bunpy patch --commit` (v0.1.10) capture local
diffs against installed packages and re-apply them on every fresh
install. `bunpy patch <pkg>` reads `bunpy.lock`, extracts the
cached wheel into `./.bunpy/patches/.pristine/<name>-<version>/`,
copies it into `./.bunpy/patches/.scratch/<name>-<version>/`, and
prints the absolute scratch path. The user edits files there.
`bunpy patch --commit <pkg>` walks both trees, emits one
whole-file unified-diff hunk per changed file, writes the body to
`./patches/<name>+<version>.patch`, and registers the entry in
`pyproject.toml` under `[tool.bunpy.patches]` (key:
`<name>@<version>`). The scratch is removed on success.
`bunpy install` reads the patches table after each wheel install
and applies the matching patch on top, rewriting `INSTALLER` to
`bunpy-patch`. The applier is strict: a context mismatch (e.g.
the pin moved under you) fails the install with a named-file
error. Refresh by re-running `bunpy patch <pkg>` against the new
pin. Flags: `--commit`, `--list`, `--out <path>`, `--no-write`,
`--target <dir>`, `--cache-dir <path>`, `--print-only`. Linked
packages cannot be patched: edit the source directly. `bunpy
install --no-patches` opts out for emergency recovery.

`bunpy why <pkg>` (v0.1.11) prints the reverse-deps tree for a
pinned package: the chain of intermediate pins that pull `<pkg>`
in, terminating at the project's direct requirements. The graph
is built from `bunpy.lock` plus each cached wheel's
Requires-Dist (markers evaluated against the host environment),
so the output reflects what would actually install on this box.
Each chain ends at a virtual `@project` edge tagged with the
lane that declared the requirement (`main`, `dev`,
`optional:<group>`, `group:<name>`, `peer`). Flags:
`--depth <N>` caps traversal; `--top` collapses to just the
direct-req names (one per line); `--json` emits a structured
result with `package`, `version`, `installer`, `linked`,
`patched`, and `chains`; `--lane <name>` restricts to one lane;
`--cache-dir <path>` overrides the wheel cache root;
`--manifest <path>` and `--lockfile <path>` override the input
files. Linked and patched pins surface their state in the
header (`(linked)`, `(patched)`) and in the JSON `installer`
field.

`bunpy workspace` (v0.2.0) manages multi-member repositories. A
workspace is a root `pyproject.toml` that declares member directories
in a `[tool.bunpy.workspace]` table:

```toml
[tool.bunpy.workspace]
members = ["packages/alpha", "packages/beta", "apps/*"]
```

All member dependencies are resolved together into a single
`bunpy.lock` at the workspace root. Members share a lock; there is
no per-member lock. bunpy auto-detects the workspace root by walking
up the directory tree from cwd, so all verbs work without any
explicit path flag when run from inside a member directory.

```
bunpy workspace --list               list member names and relative paths
bunpy add --member alpha requests    add a dep to the alpha member
bunpy install                        install from workspace-root lock
```

`bunpy audit` (v0.2.1) queries the OSV (Open Source Vulnerabilities)
database for every pinned package in `bunpy.lock`. Exit code 1 when
any unfiltered vulnerability is found.

```
bunpy audit                              table output
bunpy audit --json                       JSON array of findings
bunpy audit --quiet                      count only
bunpy audit --ignore GHSA-xxxx-yyyy      suppress one advisory
bunpy audit --lockfile path/bunpy.lock   override lockfile path
bunpy audit --workspace <root>           audit from workspace root
```

Severity is mapped from OSV's `database_specific.severity` or CVSS
score (>= 9.0: CRITICAL, >= 7.0: HIGH, >= 4.0: MEDIUM, else LOW).
`--ignore` accepts GHSA and CVE identifiers; comparison is
case-insensitive.

`bunpy publish` (v0.2.2) builds a wheel and/or sdist from the current
project using its declared PEP 517 build backend, then uploads to
PyPI (or a configured alternative registry).

```
bunpy publish                       build sdist + wheel, upload both
bunpy publish --wheel-only          wheel only
bunpy publish --sdist-only          sdist only
bunpy publish --dry-run             build but do not upload
bunpy publish --registry <url>      override upload endpoint
bunpy publish --token <token>       override PYPI_TOKEN
bunpy publish --manifest <path>     pyproject.toml path
```

Token resolution: `--token` flag, then `PYPI_TOKEN` env var. The
build backend must be installed before running `bunpy publish`; run
`bunpy install` first.

The rest of the package-manager surface lands per the v0.2.x ladder
in `docs/ROADMAP.md`:

`bunpy create` (v0.2.3) scaffolds new projects from built-in templates.
Four templates ship: `app` (CLI app with src/ layout), `lib` (library),
`script` (single .py file with shebang), and `workspace` (multi-member
root with two member stubs).

```
bunpy create app my-cli         interactive prompts
bunpy create lib my-lib --yes   accept all defaults
bunpy create workspace mono     workspace skeleton
bunpy create --list             list templates
```

### Project scaffolding

`bunpy create <template>` scaffolds from a template (app, lib,
script, workspace). Lands in v0.2.3.

## bunpyx

`bunpyx` is a companion binary shipped in the same archive as `bunpy`.
It runs a Python package entry point in a temporary prefix without
making it a permanent dependency.

```
bunpyx black .
bunpyx black@24.10.0 .
bunpyx --from black black --version
bunpyx ruff check .
```

### Execution model

1. Resolve the latest compatible version unless `@version` is given.
2. Check the wheel cache (`~/.cache/bunpyx/wheels/` by default). Download
   only on a cache miss.
3. Unpack the wheel into a temp directory, write console-script shims.
4. Exec the shim (Unix) or run it as a child process (Windows), then
   clean up.

The exit code mirrors the invoked process.

### Options

| Flag | Description |
|---|---|
| `--from <module>` | run `python -m <module>` instead of the console-script entry point |
| `--python <path>` | path to Python executable (default: `python3` on PATH) |
| `--cache-dir <dir>` | wheel cache directory (default: `~/.cache/bunpyx/wheels`) |
| `--no-cache` | skip the cache; always fetch from PyPI |
| `--keep` | keep temp prefix after exit; path printed to stderr |

### Windows note

On Windows `bunpyx` cannot use `exec(2)` semantics. It starts the
entry-point shim as a child process and exits with the child's exit
code. Signal forwarding is not implemented in v0.2.4.

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
