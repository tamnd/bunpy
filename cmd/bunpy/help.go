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
