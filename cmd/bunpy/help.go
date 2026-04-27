package main

import (
	"fmt"
	"io"
	"sort"
)

// helpEntry is one row in the help registry. Body is the long form
// printed by `bunpy help <name>` and by `bunpy <name> --help`. Both
// routes share the same string so the two surfaces never drift.
type helpEntry struct {
	Name    string
	Summary string
	Body    string
}

// helpRegistry is the source of truth for subcommand help. New
// subcommands add an entry here; tests assert that every wired
// subcommand has one and that `bunpy help <name>` matches
// `bunpy <name> --help`.
var helpRegistry = map[string]helpEntry{
	"run": {
		Name:    "run",
		Summary: "Run a Python script",
		Body: `bunpy run: explicit form of ` + "`bunpy <file.py>`" + `.

USAGE
  bunpy run <file.py> [args...]

The bare positional form ` + "`bunpy <file.py>`" + ` is the same call.
Script names defined in pyproject.toml will be supported once
the package manager lands in v0.1.x.
`,
	},
	"repl": {
		Name:    "repl",
		Summary: "Open the interactive Python prompt",
		Body: `bunpy repl: open the interactive Python prompt.

USAGE
  bunpy repl              start the prompt
  bunpy repl --quiet      no banner, no prompts (for piped stdin)

The REPL is a line-driver: each chunk is read until a blank
line, then handed to ` + "`bunpy run`" + ` as a one-shot module.
v0.0.8 is stateless: chunks do not share globals. Persistent
globals waits for gocopy to grow expression and call support.

Meta commands (prefixed with ` + "`:`" + `):
  :help            print this list of commands
  :quit, :exit     leave the REPL
  :history [N]     print the last N entries (default all)
  :clear           drop the in-flight buffer

History persists at ` + "`$HOME/.bunpy_history`" + `. Override with
` + "`BUNPY_HISTORY`" + `; cap entries with ` + "`BUNPY_HISTORY_SIZE`" + `
(default 1000, ` + "`0`" + ` disables).
`,
	},
	"stdlib": {
		Name:    "stdlib",
		Summary: "List Python stdlib modules embedded in the binary",
		Body: `bunpy stdlib: list the Python stdlib modules embedded in this binary.

USAGE
  bunpy stdlib            list module names, one per line
  bunpy stdlib ls         same as ` + "`bunpy stdlib`" + `
  bunpy stdlib count      print the number of embedded modules

The list is baked at build time from goipy's embedded stdlib.
`,
	},
	"add": {
		Name:    "add",
		Summary: "Add a single package to pyproject.toml and install it",
		Body: `bunpy add: install one package (plus its transitive deps) and write it to pyproject.toml.

USAGE
  bunpy add <pkg>                    pick the highest matching wheel
  bunpy add <pkg>==1.2.3             pin an exact version
  bunpy add <pkg>>=1.2,<2            satisfy a PEP 440 range
  bunpy add <pkg> -D, --dev          add to [dependency-groups].dev
  bunpy add <pkg> -D --group <name>  add to [dependency-groups].<name>
  bunpy add <pkg> -O <group>         add to [project.optional-dependencies].<group>
  bunpy add <pkg> -P, --peer         add to [tool.bunpy].peer-dependencies
  bunpy add <pkg> --no-install       only update the manifest and lockfile
  bunpy add <pkg> --no-write         only install
  bunpy add <pkg> --target <dir>     site-packages target (default ./.bunpy/site-packages)
  bunpy add <pkg> --index <url>      override the simple index
  bunpy add <pkg> --cache-dir <dir>  override the cache root

v0.1.5 hands the requirement to the resolver: every Requires-Dist
edge is walked, PEP 508 markers are evaluated against the host
environment, and platform wheels (manylinux, musllinux, macosx,
win) are picked through the same compatibility-tag ladder pip
uses. The chosen pin and every transitive dep land in
` + "`bunpy.lock`" + `. ` + "`--no-write`" + ` suppresses both the
manifest edit and the lockfile update; ` + "`--no-install`" + ` still
writes the lockfile. The install reuses the v0.1.2 wheel installer
(purelib only, RECORD-verified, atomic stage and rename).

When the spec is omitted, the line written into the manifest is
` + "`<name>>=<resolved-version>`" + `. When the spec is given, it is
written verbatim. Re-adding an already-listed package replaces
its line with the new spec.

v0.1.6 adds dep lanes via Bun-style flags. ` + "`-D`/`--dev`" + ` writes
the spec to PEP 735 ` + "`[dependency-groups].dev`" + ` (or to
` + "`[dependency-groups].<name>`" + ` when paired with ` + "`--group`" + `).
` + "`-O <group>`/`--optional <group>`" + ` writes to PEP 621
` + "`[project.optional-dependencies].<group>`" + `. ` + "`-P`/`--peer`" + `
writes to ` + "`[tool.bunpy].peer-dependencies`" + `. The flags are
mutually exclusive. Each pin in ` + "`bunpy.lock`" + ` is tagged with
its lane, so ` + "`bunpy install`" + ` can pick a subset; the
content-hash covers every lane.

Tests can pin every byte of every PyPI exchange by setting
` + "`BUNPY_PYPI_FIXTURES`" + ` to a directory tree that serves both the
PEP 691 simple index and the wheel bodies the index points at.
`,
	},
	"pm": {
		Name:    "pm",
		Summary: "Package manager plumbing (config, info, install-wheel...)",
		Body: `bunpy pm: package manager plumbing.

USAGE
  bunpy pm config [path]                  print parsed pyproject.toml as JSON
  bunpy pm info <package>                 print PEP 691 project metadata as JSON
  bunpy pm install-wheel <url|path>       install one wheel into site-packages
  bunpy pm lock                           regenerate bunpy.lock from pyproject.toml

The ` + "`pm`" + ` tree groups the low-level package-manager verbs.
Porcelain commands (` + "`add`" + `, ` + "`install`" + `, ` + "`remove`" + `, ` + "`update`" + `,
` + "`outdated`" + `, ` + "`why`" + `, ...) land per docs/ROADMAP.md and call into
the same machinery.

v0.1.3 wires ` + "`pm config`" + ` (parser), ` + "`pm info`" + ` (PyPI client),
and ` + "`pm install-wheel`" + ` (PEP 427 single-wheel installer). v0.1.4
adds ` + "`pm lock`" + ` (lockfile writer plus drift check). v0.1.5
swaps the picker for the PubGrub-inspired resolver: ` + "`pm lock`" + `
and ` + "`bunpy add`" + ` walk transitive deps, evaluate PEP 508
markers, and pick platform wheels. v0.1.6 teaches ` + "`add`" + `,
` + "`pm lock`" + `, and ` + "`install`" + ` to track dep lanes
(main / dev / optional groups / peer) on every lockfile row.
`,
	},
	"pm-lock": {
		Name:    "pm-lock",
		Summary: "Regenerate bunpy.lock from pyproject.toml",
		Body: `bunpy pm lock: regenerate bunpy.lock from pyproject.toml without installing.

USAGE
  bunpy pm lock                       write bunpy.lock from [project].dependencies
  bunpy pm lock --check               verify bunpy.lock matches pyproject.toml
  bunpy pm lock --index <url>         override the simple index
  bunpy pm lock --cache-dir <path>    override the cache root

The default lockfile path is ` + "`./bunpy.lock`" + ` next to ` + "`./pyproject.toml`" + `.
Every direct dep across ` + "`[project].dependencies`" + `,
` + "`[project.optional-dependencies]`" + `, ` + "`[dependency-groups]`" + `,
and ` + "`[tool.bunpy].peer-dependencies`" + ` becomes one
` + "`[[package]]`" + ` row pinning the resolved version, the wheel
filename, the URL, and the sha256 from the PyPI index. Each row
carries a ` + "`lanes`" + ` tag listing every lane it belongs to;
rows that only belong to ` + "`main`" + ` omit the field for stability.
The header records a ` + "`content-hash`" + ` derived from every lane's
sorted, trimmed dep specs, so a cheap byte compare detects
pyproject drift without a re-resolve.

` + "`--check`" + ` exits non-zero when the lockfile is missing, the
content-hash drifts from pyproject.toml, or any direct dep no
longer has a corresponding lockfile entry. Use it in CI to keep
the lockfile honest.

v0.1.5 fills transitive entries: the resolver walks every
Requires-Dist edge, evaluates PEP 508 markers, and picks
platform-aware wheels via the host tag ladder before writing the
lockfile. v0.1.6 resolves every lane (main, dev, optional groups,
peer) in one pass and tags each pin with the lanes that pulled it
in, so ` + "`bunpy install`" + ` can pick a subset without re-resolving.
`,
	},
	"pm-info": {
		Name:    "pm-info",
		Summary: "Print PEP 691 project metadata as JSON",
		Body: `bunpy pm info: fetch a PEP 691 project page and print it as JSON.

USAGE
  bunpy pm info <package>
  bunpy pm info <package> --no-cache
  bunpy pm info <package> --index <url>
  bunpy pm info <package> --cache-dir <path>

The default index is ` + "`https://pypi.org/simple/`" + `. Responses are
ETag-cached on disk under ` + "`${XDG_CACHE_HOME}/bunpy/index/`" + ` (or
the macOS / Windows equivalent); a second invocation issues an
` + "`If-None-Match`" + ` request and a 304 turns into a cache hit.

Output is a JSON object with ` + "`name`" + ` (PEP 503 normalised),
` + "`versions`" + ` (sorted), ` + "`files`" + ` (one entry per release artefact
with filename, url, hashes, requires_python, yanked flag, kind),
and ` + "`meta`" + ` (api_version, last_serial, etag).

Tests can pin every byte of every PyPI exchange by setting
` + "`BUNPY_PYPI_FIXTURES`" + ` to a directory tree. The CI smoke and
the v0.1.x test corpus both use this hook to stay offline.
`,
	},
	"pm-install-wheel": {
		Name:    "pm-install-wheel",
		Summary: "Install one wheel into site-packages (PEP 427)",
		Body: `bunpy pm install-wheel: install one wheel into site-packages.

USAGE
  bunpy pm install-wheel <path-to.whl>
  bunpy pm install-wheel <https-url>
  bunpy pm install-wheel <src> --target <dir>
  bunpy pm install-wheel <src> --no-verify
  bunpy pm install-wheel <src> --installer <name>

The default ` + "`--target`" + ` is ` + "`./.bunpy/site-packages`" + `. A path source
must end in ` + "`.whl`" + `; an ` + "`https://`" + ` URL is fetched through the same
transport ` + "`pm info`" + ` uses, so ` + "`BUNPY_PYPI_FIXTURES`" + ` swaps it for
the fixture root in tests.

v0.1.2 supports purelib wheels only (` + "`Root-Is-Purelib: true`" + `, no
` + "`*.data/`" + ` subdirs). RECORD hashes are verified before any byte
hits disk; a mismatch aborts the install. Unsafe entries (zip-slip,
absolute paths, parent traversal) are rejected at the entry level.
The install is staged under a tempdir inside ` + "`--target`" + ` and renamed
into place, so a mid-install crash leaves the existing tree intact.

The post-install RECORD is re-emitted alongside an ` + "`INSTALLER`" + ` file
under the wheel's dist-info directory.
`,
	},
	"pm-config": {
		Name:    "pm-config",
		Summary: "Print parsed pyproject.toml as JSON",
		Body: `bunpy pm config: print the parsed pyproject.toml as JSON.

USAGE
  bunpy pm config              read ./pyproject.toml
  bunpy pm config <path>       read a specific file

The output is a JSON object with three top-level keys:
  project   PEP 621 fields (name, version, dependencies, ...)
  tool      the [tool.bunpy] table, kept verbatim
  other     any unrecognised top-level table, kept verbatim

In strict mode (the default) bunpy rejects:
  - a missing [project] table
  - a missing or empty project.name
  - a project.name that does not match the PEP 503 regex
  - a project.dynamic entry that is also set literally

Exit status is 1 on any of those, or on a filesystem or TOML
syntax error.
`,
	},
	"version": {
		Name:    "version",
		Summary: "Print version, commit, build date, and toolchain pins",
		Body: `bunpy version: print build metadata.

USAGE
  bunpy version            multi-line banner with commit, date, toolchain
  bunpy version --short    just the version string
  bunpy version --json     one-line JSON object

A locally-built binary prints just ` + "`bunpy dev`" + ` and the go/os/arch
line. Release builds add the commit, build date, and the three
pinned sibling-toolchain commits (gopapy, gocopy, goipy).
`,
	},
	"help": {
		Name:    "help",
		Summary: "Print help for a subcommand",
		Body: `bunpy help: print detailed help for a subcommand.

USAGE
  bunpy help               top-level usage (same as ` + "`bunpy --help`" + `)
  bunpy help <command>     long-form help for one subcommand

Equivalent to ` + "`bunpy <command> --help`" + `.
`,
	},
	"install": {
		Name:    "install",
		Summary: "Install every pin in bunpy.lock into site-packages",
		Body: `bunpy install: install every pinned wheel from bunpy.lock.

USAGE
  bunpy install                       install main-lane pins into ./.bunpy/site-packages
  bunpy install -D, --dev             also install [dependency-groups] pins
  bunpy install -O <group>            also install one optional-dependencies group
  bunpy install --all-extras          also install every optional-dependencies group
  bunpy install -P, --peer            also install [tool.bunpy].peer-dependencies
  bunpy install --production          alias for the default (main only); rejects lane flags
  bunpy install --target <dir>        site-packages target
  bunpy install --cache-dir <dir>     override the wheel cache root
  bunpy install --no-verify           skip RECORD hash verification

v0.1.5 treats ` + "`bunpy.lock`" + ` as the source of truth: the
resolver does not run, every wheel is fetched through the same
httpkit + cache path ` + "`bunpy add`" + ` uses, and each pin is
installed via the v0.1.2 wheel installer. Run
` + "`bunpy pm lock`" + ` first to refresh the lockfile from
` + "`pyproject.toml`" + `.

v0.1.6 reads the per-pin ` + "`lanes`" + ` tag and filters by the
lane flags above. The default keeps only ` + "`main`" + `; pins
without a ` + "`lanes`" + ` field are treated as ` + "`main`" + `.
` + "`-O`" + ` may be repeated to enable several optional groups.
` + "`--production`" + ` is mutually exclusive with the lane flags
and is provided for Bun parity.
`,
	},
	"outdated": {
		Name:    "outdated",
		Summary: "Show pins that have a newer matching release",
		Body: `bunpy outdated: print one row per lockfile pin with a newer release.

USAGE
  bunpy outdated                      walk every pin in bunpy.lock
  bunpy outdated <pkg>...             restrict to the named packages
  bunpy outdated -D, --dev            include [dependency-groups] pins
  bunpy outdated -O <group>           include one optional group
  bunpy outdated --all-extras         include every optional group
  bunpy outdated -P, --peer           include [tool.bunpy].peer-dependencies
  bunpy outdated --production         alias for the default (main only)
  bunpy outdated --json               emit JSON {"outdated":[...]}
  bunpy outdated --index <url>        override the simple index
  bunpy outdated --cache-dir <path>   override the cache root

The table has columns ` + "`package`" + `, ` + "`current`" + ` (lockfile
version), ` + "`wanted`" + ` (highest version that satisfies the
manifest spec, the version ` + "`bunpy update`" + ` would pick),
` + "`latest`" + ` (highest non-yanked release on the index, the
version ` + "`bunpy update --latest`" + ` would pick), and
` + "`lanes`" + ` (comma-joined lane labels from the lockfile row).

Read-only: this verb never writes ` + "`bunpy.lock`" + ` or installs
anything. Exit status is 0 even when pins are outdated; use
` + "`--json`" + ` to scrub the result in scripts. Run
` + "`bunpy pm lock`" + ` first when ` + "`bunpy.lock`" + ` is missing.
`,
	},
	"update": {
		Name:    "update",
		Summary: "Refresh bunpy.lock to the highest matching versions",
		Body: `bunpy update: re-resolve bunpy.lock and refresh site-packages.

USAGE
  bunpy update                        upgrade every pin within its manifest spec
  bunpy update <pkg>...               unlock only the named packages
  bunpy update --latest <pkg>...      ignore manifest spec for those packages
  bunpy update --no-install           rewrite bunpy.lock but skip the install
  bunpy update -D, --dev              include [dependency-groups] in the install
  bunpy update -O <group>             include one optional group in the install
  bunpy update --all-extras           include every optional group in the install
  bunpy update -P, --peer             include peer pins in the install
  bunpy update --production           alias for default (main only); rejects lane flags
  bunpy update --target <dir>         site-packages target
  bunpy update --cache-dir <path>     override the wheel cache root
  bunpy update --no-verify            skip RECORD hash verification
  bunpy update --index <url>          override the simple index

` + "`bunpy update`" + ` runs the v0.1.5 resolver with ` + "`Solver.Locked`" + `
seeded from the existing lockfile. Packages named on the command
line are dropped from ` + "`Locked`" + ` so the resolver picks afresh
within their manifest spec; everything else stays at the locked
version when that version still satisfies any new constraints.
A bare ` + "`bunpy update`" + ` clears every lock and re-resolves
the whole graph.

` + "`--latest <pkg>...`" + ` strips the manifest spec for the named
packages and lets the resolver pick the highest non-yanked,
non-prerelease version. ` + "`--latest`" + ` requires at least one
package name to avoid surprise mass upgrades.

After resolving, ` + "`bunpy.lock`" + ` is rewritten with the new
pins and lane tags. ` + "`stdout`" + ` lists each changed pin as
` + "`name old -> new`" + `; an unchanged graph prints
` + "`no changes`" + `. Unless ` + "`--no-install`" + ` is set, the
selected lanes are then installed via the same wheel installer
` + "`bunpy install`" + ` uses.
`,
	},
	"remove": {
		Name:    "remove",
		Summary: "Drop packages from pyproject.toml and uninstall them",
		Body: `bunpy remove: the inverse of ` + "`bunpy add`" + `.

USAGE
  bunpy remove <pkg>...               drop from every lane
  bunpy remove <pkg> -D, --dev        drop from [dependency-groups].dev
  bunpy remove <pkg> -D --group <n>   drop from [dependency-groups].<n>
  bunpy remove <pkg> -O <group>       drop from [project.optional-dependencies].<group>
  bunpy remove <pkg> -P, --peer       drop from [tool.bunpy].peer-dependencies
  bunpy remove <pkg> --no-install     only edit pyproject.toml and bunpy.lock
  bunpy remove <pkg> --no-write       skip the manifest edit; only re-resolve
  bunpy remove <pkg> --target <dir>   site-packages target
  bunpy remove <pkg> --index <url>    override the simple index

A bare ` + "`bunpy remove <pkg>`" + ` deletes the named package from every
lane it appears in: ` + "`[project].dependencies`" + `, every PEP 735
` + "`[dependency-groups]`" + `, every PEP 621
` + "`[project.optional-dependencies]`" + `, and
` + "`[tool.bunpy].peer-dependencies`" + `. Lane flags ` + "`-D`" + `, ` + "`-O`" + `, and
` + "`-P`" + ` restrict the delete to one lane (and are mutually
exclusive). ` + "`--group <name>`" + ` requires ` + "`-D`" + ` and picks one
non-default group.

After the manifest edit, ` + "`bunpy remove`" + ` re-runs the v0.1.5
resolver with ` + "`Solver.Locked`" + ` seeded from the surviving
lockfile pins (the named packages are dropped from the lock hint).
Pins that lose every root fall off; ` + "`bunpy.lock`" + ` is rewritten
to match. Unless ` + "`--no-install`" + ` is set, the dropped pins are
removed from ` + "`./.bunpy/site-packages`" + ` via a RECORD walk;
top-level package directories are best-effort cleaned up when no
RECORD is present.

Removing a name that is not in the manifest is not an error: the
verb is idempotent. Removing the last entry in an array keeps the
array (` + "`dependencies = [\n]`" + `) so the diff stays small.
`,
	},
	"link": {
		Name:    "link",
		Summary: "Register or install editable links from the global registry",
		Body: `bunpy link: register the current project, or install editable links into a consumer.

USAGE
  bunpy link                          register ./pyproject.toml in the global registry
  bunpy link <pkg>...                 install each named link into ./.bunpy/site-packages
  bunpy link --list                   print every entry in the global registry
  bunpy link --target <dir>           consumer-side site-packages target

A bare ` + "`bunpy link`" + ` reads ` + "`./pyproject.toml`" + `, normalises the
project name, and writes a JSON entry into the global link
registry. The registry root is ` + "`$BUNPY_LINK_DIR`" + ` (override
for tests/CI), or the platform's user-data directory under
` + "`bunpy/links/`" + `: ` + "`$XDG_DATA_HOME/bunpy/links`" + ` on Linux,
` + "`~/Library/Application Support/bunpy/links`" + ` on macOS,
` + "`%LOCALAPPDATA%\\bunpy\\links`" + ` on Windows. Re-registering an
existing entry overwrites it (idempotent).

` + "`bunpy link <pkg>...`" + ` reads each entry from the registry and
lays down a PEP 660 editable proxy under
` + "`./.bunpy/site-packages`" + `: a ` + "`<name>.pth`" + ` pointing at the
absolute source root, plus a
` + "`<name>-<version>.dist-info/`" + ` directory with METADATA,
RECORD, ` + "`direct_url.json`" + ` (PEP 610), and an
` + "`INSTALLER`" + ` file equal to ` + "`bunpy-link`" + `. ` + "`bunpy install`" + `
reads ` + "`INSTALLER`" + ` and skips re-installing any linked
package, so a link survives unrelated installs. To flip a
linked package back to the pinned wheel, run
` + "`bunpy unlink <pkg>`" + ` followed by ` + "`bunpy install`" + `.

The verb is read-only with respect to ` + "`pyproject.toml`" + ` and
` + "`bunpy.lock`" + `: links are tooling state and never appear in
either file.
`,
	},
	"unlink": {
		Name:    "unlink",
		Summary: "Unregister or remove editable links",
		Body: `bunpy unlink: drop a registry entry, or remove links from a consumer.

USAGE
  bunpy unlink                        drop ./pyproject.toml from the global registry
  bunpy unlink <pkg>...               remove each named link from ./.bunpy/site-packages
  bunpy unlink --target <dir>         consumer-side site-packages target

A bare ` + "`bunpy unlink`" + ` deletes the JSON entry for the current
project from the global registry. Existing consumer-side
installs keep working (the ` + "`.pth`" + ` is local) until the
consumer re-links or re-installs. Missing-entry is a no-op.

` + "`bunpy unlink <pkg>...`" + ` finds the
` + "`<name>-<version>.dist-info`" + ` directory under the target,
verifies it carries ` + "`INSTALLER=bunpy-link`" + ` (so the verb
never accidentally removes a pinned wheel), walks RECORD with
the same path-escape guard ` + "`bunpy remove`" + ` uses, and
unlinks every listed file. A name that has no link installed
prints ` + "`no link for <name>`" + ` and exits 0 (idempotent).

After ` + "`bunpy unlink <pkg>`" + ` you can run ` + "`bunpy install`" + ` to
re-fetch the pinned wheel and put it back in place.
`,
	},
	"patch": {
		Name:    "patch",
		Summary: "Open and commit patches against installed packages",
		Body: `bunpy patch: open a mutable copy of an installed package, then commit a unified-diff patch.

USAGE
  bunpy patch <pkg>                  open scratch, print the editable path
  bunpy patch --commit <pkg>         diff scratch vs pristine, write the patch
  bunpy patch --commit <pkg> --out <p>  override the default patch path
  bunpy patch --commit <pkg> --no-write skip the pyproject.toml edit
  bunpy patch --list                 print every registered patch
  bunpy patch --target <dir>         consumer-side site-packages target
  bunpy patch --cache-dir <p>        wheel cache root

` + "`bunpy patch <pkg>`" + ` reads the pin for ` + "`<pkg>`" + ` from
` + "`bunpy.lock`" + `, extracts the cached wheel into
` + "`./.bunpy/patches/.pristine/<name>-<version>/`" + ` (creating
the pristine baseline), then copies it to
` + "`./.bunpy/patches/.scratch/<name>-<version>/`" + ` and prints
the absolute scratch path on stdout. Edit the files there with
your normal editor, then commit.

` + "`bunpy patch --commit <pkg>`" + ` walks the scratch and pristine
trees, emits one whole-file unified-diff hunk per changed file,
writes the body to ` + "`./patches/<name>+<version>.patch`" + `,
and registers the entry in pyproject.toml under
` + "`[tool.bunpy.patches]`" + ` (key shape ` + "`<name>@<version>`" + `).
The scratch directory is removed on success. ` + "`--no-write`" + `
skips the pyproject.toml edit; ` + "`--out`" + ` overrides the
patch path.

` + "`bunpy install`" + ` reads the patch table after every wheel
install lands and applies the matching patch on top. The
applier is strict: a mismatch on context (e.g. the pin moved
under you) fails the install with a named-file error. Re-run
` + "`bunpy patch <pkg>`" + ` against the new pin to refresh.
` + "`--no-patches`" + ` opts out for emergency recovery.

A linked package (` + "`INSTALLER=bunpy-link`" + `, see
` + "`bunpy link`" + `) cannot be patched: edit the source
directly. ` + "`bunpy patch <linked-pkg>`" + ` exits with a clear
error.
`,
	},
	"why": {
		Name:    "why",
		Summary: "Print the reverse-deps tree for a pin in bunpy.lock",
		Body: `bunpy why: print the reverse-deps tree for a pinned package.

USAGE
  bunpy why <pkg>                  tree from <pkg> upward to project requirements
  bunpy why <pkg> --depth <N>      cap traversal depth (0 = unlimited)
  bunpy why <pkg> --top            print only the direct project requirements
                                   that pull <pkg> in (one name per line)
  bunpy why <pkg> --json           machine-readable tree
  bunpy why <pkg> --lane <name>    restrict to chains in one lane
  bunpy why <pkg> --cache-dir <p>  wheel cache root (for Requires-Dist lookup)
  bunpy why <pkg> --manifest <p>   pyproject.toml path (default ./pyproject.toml)
  bunpy why <pkg> --lockfile <p>   bunpy.lock path (default ./bunpy.lock)

` + "`bunpy why`" + ` reads ` + "`bunpy.lock`" + ` and the cached
wheels' Requires-Dist metadata to build a forward dependency
graph, then walks upward from the queried pin to every direct
project requirement that transitively reaches it. Each chain
terminates at a virtual ` + "`@project`" + ` edge tagged with the
lane it was declared in (` + "`main`" + `, ` + "`dev`" + `,
` + "`optional:<group>`" + `, ` + "`group:<name>`" + `, ` + "`peer`" + `).

The default output indents each chain by depth and labels every
parent edge with its name, version, and the spec it declared on
the child. ` + "`--top`" + ` collapses to just the direct-req names
(useful for grep/scripting). ` + "`--json`" + ` emits the full
` + "`Result`" + ` shape (package, version, installer, linked,
patched, chains).

The result also surfaces overlay state: a linked pin shows
` + "`(linked)`" + ` next to the package name and uses installer
` + "`bunpy-link`" + `; a patched pin shows ` + "`(patched)`" + `
and uses installer ` + "`bunpy-patch`" + `.
`,
	},
	"publish": {
		Name:    "publish",
		Summary: "Build and upload a wheel and/or sdist to PyPI",
		Body: `bunpy publish: build artefacts and upload them to a registry.

USAGE
  bunpy publish                      build sdist + wheel, upload both
  bunpy publish --wheel-only         wheel only
  bunpy publish --sdist-only         sdist only
  bunpy publish --dry-run            build but do not upload
  bunpy publish --registry <url>     override upload endpoint
  bunpy publish --token <token>      override PYPI_TOKEN
  bunpy publish --manifest <path>    pyproject.toml path (default ./pyproject.toml)

Reads ` + "`[build-system].build-backend`" + ` from ` + "`pyproject.toml`" + `
(default ` + "`hatchling.build`" + `) and invokes the PEP 517 build hooks
via the Python interpreter found on PATH.

The upload uses HTTP multipart POST to the registry endpoint with
HTTP Basic auth: username ` + "`__token__`" + `, password = token.

Token resolution order: ` + "`--token`" + ` flag, then ` + "`PYPI_TOKEN`" + `
environment variable. Both are required unless ` + "`--dry-run`" + ` is set.
`,
	},
	"audit": {
		Name:    "audit",
		Summary: "Scan pinned packages against the OSV vulnerability database",
		Body: `bunpy audit: check bunpy.lock against OSV for known vulnerabilities.

USAGE
  bunpy audit                      table output; exit 1 if any vuln found
  bunpy audit --json               JSON array of findings
  bunpy audit --quiet              print count only
  bunpy audit --ignore <id>        suppress one advisory (repeatable)
  bunpy audit --lockfile <path>    path to bunpy.lock (default ./bunpy.lock)
  bunpy audit --workspace <root>   audit via workspace-root lock

Queries the OSV (Open Source Vulnerabilities) database at
https://api.osv.dev/v1/querybatch for every pinned package in
` + "`bunpy.lock`" + `. No API key is required for PyPI ecosystem queries.

Exit codes: 0 (no vulns, or all suppressed), 1 (vulns found), 2 (usage error).

` + "`--ignore`" + ` accepts both GHSA and CVE identifiers; comparison is
case-insensitive and may be repeated for multiple suppressions.
`,
	},
	"workspace": {
		Name:    "workspace",
		Summary: "List and navigate workspace members",
		Body: `bunpy workspace: inspect multi-member workspaces.

USAGE
  bunpy workspace --list               list member names and paths
  bunpy workspace --workspace <root>   override workspace root detection

A workspace is a ` + "`pyproject.toml`" + ` at the root that declares:

  [tool.bunpy.workspace]
  members = ["packages/alpha", "packages/beta", "apps/server"]

Paths may contain glob patterns (e.g. ` + "`\"packages/*\"`" + `).
bunpy auto-detects the workspace root by walking up the directory
tree from cwd. A single ` + "`bunpy.lock`" + ` at the workspace root
covers all member dependencies.

Use ` + "`bunpy add --member <name> <pkg>`" + ` to add a dependency to
a specific workspace member. ` + "`bunpy install`" + ` inside any
member directory reads the workspace-root lock automatically.
`,
	},
	"man": {
		Name:    "man",
		Summary: "Print or install the bundled manpages",
		Body: `bunpy man: read or install bunpy's bundled manpages.

USAGE
  bunpy man <command>            print the roff manpage to stdout
  bunpy man --install <dir>      copy all manpages into <dir>/man1/
  bunpy man --install            same, default <dir> = $HOME/.bunpy/share/man

Pipe to ` + "`man -l -`" + ` to render: ` + "`bunpy man run | man -l -`" + `.
The release archives also ship the rendered pages under
share/man/man1/, so install.sh and the Homebrew formula install
them automatically.
`,
	},
}

func helpSubcommand(args []string, stdout, stderr io.Writer) (int, error) {
	if len(args) == 0 {
		usage(stdout)
		return 0, nil
	}
	switch args[0] {
	case "-h", "--help":
		return printHelp("help", stdout, stderr)
	}
	return printHelp(args[0], stdout, stderr)
}

func printHelp(name string, stdout, stderr io.Writer) (int, error) {
	entry, ok := helpRegistry[name]
	if !ok {
		fmt.Fprintln(stderr, "bunpy: no help topic for", name)
		fmt.Fprintln(stderr, "Try `bunpy help` to see the list of subcommands.")
		return 1, fmt.Errorf("no help topic %q", name)
	}
	fmt.Fprint(stdout, entry.Body)
	return 0, nil
}

func helpTopics() []string {
	names := make([]string, 0, len(helpRegistry))
	for n := range helpRegistry {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}
