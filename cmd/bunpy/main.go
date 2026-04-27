// Command bunpy is one binary for Python: runtime + package manager +
// bundler + test runner. Bun's developer experience, brought to Python.
//
// v0.0.2 wires the runtime: a positional `.py` file argument runs the
// script through gocopy plus goipy. Subcommands land per the ladder
// in docs/ROADMAP.md.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/tamnd/bunpy/v1/runtime"
)

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
		return versionSubcommand(args[1:], stdout, stderr)
	case "help":
		return helpSubcommand(args[1:], stdout, stderr)
	case "-h", "--help":
		usage(stdout)
		return 0, nil
	case "run":
		return runSubcommand(args[1:], stdout, stderr)
	case "repl":
		return replSubcommand(args[1:], stdout, stderr)
	case "stdlib":
		return stdlibSubcommand(args[1:], stdout, stderr)
	case "pm":
		return pmSubcommand(args[1:], stdout, stderr)
	case "add":
		return addSubcommand(args[1:], stdout, stderr)
	case "install":
		return installSubcommand(args[1:], stdout, stderr)
	case "outdated":
		return outdatedSubcommand(args[1:], stdout, stderr)
	case "update":
		return updateSubcommand(args[1:], stdout, stderr)
	case "remove":
		return removeSubcommand(args[1:], stdout, stderr)
	case "link":
		return linkSubcommand(args[1:], stdout, stderr)
	case "unlink":
		return unlinkSubcommand(args[1:], stdout, stderr)
	case "patch":
		return patchSubcommand(args[1:], stdout, stderr)
	case "why":
		return whySubcommand(args[1:], stdout, stderr)
	case "workspace":
		return workspaceSubcommand(args[1:], stdout, stderr)
	case "audit":
		return auditSubcommand(args[1:], stdout, stderr)
	case "publish":
		return publishSubcommand(args[1:], stdout, stderr)
	case "create":
		return createSubcommand(args[1:], stdout, stderr)
	case "man":
		return manSubcommand(args[1:], stdout, stderr)
	}

	if isFilePath(args[0]) {
		return runFile(args[0], args[1:], stdout, stderr)
	}

	usage(stderr)
	return 1, fmt.Errorf("unknown command %q (v0.2.3 wires --version, --help, `bunpy <file.py>`, `bunpy run`, `bunpy repl`, `bunpy stdlib`, `bunpy pm`, `bunpy add`, `bunpy install`, `bunpy outdated`, `bunpy update`, `bunpy remove`, `bunpy link`, `bunpy unlink`, `bunpy patch`, `bunpy why`, `bunpy workspace`, `bunpy audit`, `bunpy publish`, `bunpy create`, `bunpy help`, `bunpy man`)", args[0])
}

func versionSubcommand(args []string, stdout, stderr io.Writer) (int, error) {
	mode := "plain"
	for _, a := range args {
		switch a {
		case "--short":
			mode = "short"
		case "--json":
			mode = "json"
		case "-h", "--help":
			return printHelp("version", stdout, stderr)
		default:
			return 1, fmt.Errorf("bunpy version: unknown flag %q (known: --short, --json)", a)
		}
	}
	b := runtime.Build()
	switch mode {
	case "short":
		fmt.Fprintln(stdout, b.Version)
		return 0, nil
	case "json":
		data, err := json.Marshal(b)
		if err != nil {
			return 1, fmt.Errorf("bunpy version --json: %w", err)
		}
		fmt.Fprintln(stdout, string(data))
		return 0, nil
	default:
		printVersion(stdout, b)
		return 0, nil
	}
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
		return printHelp("stdlib", stdout, stderr)
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
		return printHelp("run", stdout, stderr)
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

func printVersion(w io.Writer, b runtime.BuildInfo) {
	if b.Commit == "" && b.BuildDate == "" {
		fmt.Fprintf(w, "bunpy %s\n", b.Version)
		fmt.Fprintf(w, "go %s %s/%s\n", b.Go, b.OS, b.Arch)
		return
	}
	parts := []string{}
	if b.Commit != "" {
		parts = append(parts, "commit "+b.Commit)
	}
	if b.BuildDate != "" {
		parts = append(parts, "built "+b.BuildDate)
	}
	fmt.Fprintf(w, "bunpy %s (%s)\n", b.Version, strings.Join(parts, ", "))
	fmt.Fprintf(w, "go %s %s/%s\n", b.Go, b.OS, b.Arch)
	if b.Goipy != "" || b.Gocopy != "" || b.Gopapy != "" {
		fmt.Fprintf(w, "toolchain: gopapy %s / gocopy %s / goipy %s\n", b.Gopapy, b.Gocopy, b.Goipy)
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
	fmt.Fprintln(w, "  bunpy help <command>        Long-form help for a subcommand")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "COMMANDS")
	for _, name := range helpTopics() {
		fmt.Fprintf(w, "  %-9s %s\n", name, helpRegistry[name].Summary)
	}
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Run `bunpy help <command>` for the long form, or `bunpy man")
	fmt.Fprintln(w, "<command>` for the manpage. The aspirational command list")
	fmt.Fprintln(w, "(install, add, build, test, repl, fmt, check) is documented in")
	fmt.Fprintln(w, "docs/CLI.md and lands per docs/ROADMAP.md.")
}
