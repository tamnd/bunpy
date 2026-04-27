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
