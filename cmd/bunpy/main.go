// Command bunpy is one binary for Python: runtime + package manager +
// bundler + test runner. Bun's developer experience, brought to Python.
//
// v0.0.2 wires the runtime: a positional `.py` file argument runs the
// script through gocopy plus goipy. Subcommands land per the ladder
// in docs/ROADMAP.md.
package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/tamnd/bunpy/v1/runtime"
)

// version is overwritten at build time via -ldflags "-X main.version=...".
var version = "0.0.4"

// commit is overwritten at build time. Empty in dev builds.
var commit = ""

// buildDate is overwritten at build time (RFC 3339). Empty in dev builds.
var buildDate = ""

func main() {
	code, err := run(os.Args[1:], os.Stdout, os.Stderr)
	if err != nil {
		fmt.Fprintln(os.Stderr, "bunpy:", err)
		if code == 0 {
			code = 1
		}
	}
	os.Exit(code)
}

func run(args []string, stdout, stderr io.Writer) (int, error) {
	if len(args) == 0 {
		usage(stdout)
		return 0, nil
	}

	switch args[0] {
	case "version", "-v", "--version":
		printVersion(stdout)
		return 0, nil
	case "help", "-h", "--help":
		usage(stdout)
		return 0, nil
	case "run":
		return runSubcommand(args[1:], stdout, stderr)
	case "stdlib":
		return stdlibSubcommand(args[1:], stdout, stderr)
	}

	if isFilePath(args[0]) {
		return runFile(args[0], args[1:], stdout, stderr)
	}

	usage(stderr)
	return 1, fmt.Errorf("unknown command %q (v0.0.4 wires --version, --help, `bunpy <file.py>`, `bunpy run`, `bunpy stdlib`)", args[0])
}

func stdlibSubcommand(args []string, stdout, stderr io.Writer) (int, error) {
	mode := "ls"
	if len(args) > 0 {
		mode = args[0]
	}
	switch mode {
	case "ls":
		for _, m := range runtime.StdlibModules() {
			fmt.Fprintln(stdout, m)
		}
		return 0, nil
	case "count":
		fmt.Fprintln(stdout, runtime.StdlibCount())
		return 0, nil
	case "-h", "--help", "help":
		fmt.Fprintln(stdout, "bunpy stdlib: list the Python stdlib modules embedded in this binary.")
		fmt.Fprintln(stdout, "")
		fmt.Fprintln(stdout, "USAGE")
		fmt.Fprintln(stdout, "  bunpy stdlib            list module names, one per line")
		fmt.Fprintln(stdout, "  bunpy stdlib ls         same as `bunpy stdlib`")
		fmt.Fprintln(stdout, "  bunpy stdlib count      print the number of embedded modules")
		fmt.Fprintln(stdout, "")
		fmt.Fprintln(stdout, "The list is baked at build time from goipy's embedded stdlib.")
		return 0, nil
	default:
		return 1, fmt.Errorf("bunpy stdlib %q: known modes are ls, count, --help", mode)
	}
}

func runSubcommand(args []string, stdout, stderr io.Writer) (int, error) {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: bunpy run <file.py> [args...]")
		return 1, fmt.Errorf("bunpy run requires a script argument")
	}
	switch args[0] {
	case "-h", "--help":
		fmt.Fprintln(stdout, "bunpy run: explicit form of `bunpy <file.py>`.")
		fmt.Fprintln(stdout, "")
		fmt.Fprintln(stdout, "USAGE")
		fmt.Fprintln(stdout, "  bunpy run <file.py> [args...]")
		fmt.Fprintln(stdout, "")
		fmt.Fprintln(stdout, "Script names defined in pyproject.toml will be supported once")
		fmt.Fprintln(stdout, "the package manager lands in v0.1.x.")
		return 0, nil
	case "-":
		return 1, fmt.Errorf("bunpy run -: stdin scripts not yet wired")
	}
	if !isFilePath(args[0]) {
		return 1, fmt.Errorf("bunpy run %q: only file paths ending in .py are wired in v0.0.3", args[0])
	}
	return runFile(args[0], args[1:], stdout, stderr)
}

func runFile(path string, args []string, stdout, stderr io.Writer) (int, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return 1, err
	}
	return runtime.Run(path, src, args, stdout, stderr)
}

// isFilePath reports whether arg looks like a script path. A leading '-'
// is reserved for flags. We only auto-run files ending in .py for now;
// this keeps the unknown-command error message useful for typos.
func isFilePath(arg string) bool {
	if strings.HasPrefix(arg, "-") {
		return false
	}
	return strings.HasSuffix(arg, ".py")
}

func printVersion(w io.Writer) {
	fmt.Fprintln(w, version)
	if commit != "" || buildDate != "" {
		extra := ""
		if commit != "" {
			extra = "commit " + commit
		}
		if buildDate != "" {
			if extra != "" {
				extra += ", "
			}
			extra += "built " + buildDate
		}
		fmt.Fprintln(w, extra)
	}
}

func usage(w io.Writer) {
	fmt.Fprintln(w, "bunpy: one binary for Python (runtime, packages, bundler, tests).")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "USAGE")
	fmt.Fprintln(w, "  bunpy <file.py> [args...]   Run a Python script")
	fmt.Fprintln(w, "  bunpy <command> [args]")
	fmt.Fprintln(w, "  bunpy --version")
	fmt.Fprintln(w, "  bunpy --help")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "COMMANDS (planned, see docs/ROADMAP.md for the per-version ladder)")
	fmt.Fprintln(w, "  run <file.py>     Run a Python script")
	fmt.Fprintln(w, "  test              Discover and run tests")
	fmt.Fprintln(w, "  install           Install dependencies from pyproject.toml + bunpy.lock")
	fmt.Fprintln(w, "  add <pkg>         Add a dependency")
	fmt.Fprintln(w, "  remove <pkg>      Remove a dependency")
	fmt.Fprintln(w, "  update [pkg]      Update dependencies")
	fmt.Fprintln(w, "  outdated          List outdated dependencies")
	fmt.Fprintln(w, "  audit             Check for security advisories")
	fmt.Fprintln(w, "  link / unlink     Editable install of cwd")
	fmt.Fprintln(w, "  patch <pkg>       Persist a local patch against an installed package")
	fmt.Fprintln(w, "  publish           Build and publish sdist+wheel to PyPI")
	fmt.Fprintln(w, "  pm                Package-manager utilities (cache, ls, hash, why)")
	fmt.Fprintln(w, "  why <pkg>         Explain why a package is installed")
	fmt.Fprintln(w, "  init              Scaffold a new project")
	fmt.Fprintln(w, "  create <tmpl>     Scaffold from a template")
	fmt.Fprintln(w, "  build             Bundle a project (.pyz or single binary)")
	fmt.Fprintln(w, "  repl              Interactive REPL")
	fmt.Fprintln(w, "  fmt               Format Python source (delegates to gopapy)")
	fmt.Fprintln(w, "  check             Lint Python source (delegates to gopapy)")
	fmt.Fprintln(w, "  stdlib            List Python stdlib modules embedded in the binary")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "v0.0.4 ships --version, --help, `bunpy <file.py>`, `bunpy run`,")
	fmt.Fprintln(w, "and `bunpy stdlib`. Each rung in docs/ROADMAP.md adds one")
	fmt.Fprintln(w, "capability with a green CI matrix on linux, macOS, and Windows.")
}
