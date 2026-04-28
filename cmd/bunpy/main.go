// Command bunpy is one binary for Python: runtime + package manager +
// bundler + test runner. Bun's developer experience, brought to Python.
//
// v0.0.2 wires the runtime: a positional `.py` file argument runs the
// script through gocopy plus goipy. Subcommands land per the ladder
// in docs/ROADMAP.md.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/pprof"
	"strings"
	"time"

	"github.com/tamnd/bunpy/v1/internal/bundler"
	"github.com/tamnd/bunpy/v1/internal/dotenv"
	"github.com/tamnd/bunpy/v1/runtime"
)

func main() {
	os.Exit(mainCode())
}

// mainCode runs the program and returns the exit code. Separated from main so
// that deferred cleanup (pprof stop, file close) runs before os.Exit is called.
func mainCode() int {
	if os.Getenv("BUNPY_PROFILE_STARTUP") == "1" {
		pprofPath := os.Getenv("BUNPY_STARTUP_PPROF")
		if pprofPath == "" {
			pprofPath = "/tmp/bunpy-startup.pprof"
		}
		if f, err := os.Create(pprofPath); err == nil {
			pprof.StartCPUProfile(f)
			defer func() {
				pprof.StopCPUProfile()
				f.Close()
			}()
		}
	}
	code, err := run(os.Args[1:], os.Stdout, os.Stderr)
	if err != nil {
		fmt.Fprintln(os.Stderr, "bunpy:", err)
		if code == 0 {
			code = 1
		}
	}
	return code
}

func run(args []string, stdout, stderr io.Writer) (int, error) {
	if len(args) == 0 {
		usage(stdout)
		return 0, nil
	}

	if args[0] == "-c" {
		if len(args) < 2 {
			fmt.Fprintln(stderr, "usage: bunpy -c <code> [args...]")
			return 1, fmt.Errorf("bunpy -c requires a code argument")
		}
		return runInline(args[1], args[2:], stdout, stderr)
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
	case "sync":
		return syncSubcommand(args[1:], stdout, stderr)
	case "uv":
		return uvSubcommand(args[1:], stdout, stderr)
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
	case "test":
		return testSubcommand(args[1:], stdout, stderr)
	case "build":
		return buildSubcommand(args[1:], stdout, stderr)
	case "fmt":
		return fmtSubcommand(args[1:], stdout, stderr)
	case "check":
		return checkSubcommand(args[1:], stdout, stderr)
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
	return 1, fmt.Errorf("unknown command %q (see bunpy --help)", args[0])
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

	// Parse run-specific flags before the file argument.
	var (
		useCache  bool
		watchMode bool
		cpython   bool
		inspect   bool
		envFiles  []string
		rest      []string
	)
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--cache":
			useCache = true
		case a == "--no-cache" || a == "--cache=false":
			useCache = false
		case a == "--watch":
			watchMode = true
		case a == "--cpython":
			cpython = true
		case a == "--inspect":
			inspect = true
		case a == "--env-file":
			if i+1 < len(args) {
				i++
				envFiles = append(envFiles, args[i])
			}
		case strings.HasPrefix(a, "--env-file="):
			envFiles = append(envFiles, strings.TrimPrefix(a, "--env-file="))
		case a == "-h" || a == "--help":
			return printHelp("run", stdout, stderr)
		case a == "-":
			return 1, fmt.Errorf("bunpy run -: stdin scripts not yet wired")
		default:
			rest = append(rest, a)
		}
	}
	if len(rest) == 0 {
		fmt.Fprintln(stderr, "usage: bunpy run <file.py> [args...]")
		return 1, fmt.Errorf("bunpy run requires a script argument")
	}

	// Load .env files.
	if err := dotenv.LoadFiles(envFiles); err != nil {
		return 1, fmt.Errorf("run: env-file: %w", err)
	}

	if strings.HasSuffix(rest[0], ".pyz") {
		return runPYZ(rest[0], rest[1:])
	}
	if !isFilePath(rest[0]) {
		return 1, fmt.Errorf("bunpy run %q: only file paths ending in .py or .pyz are supported", rest[0])
	}

	path, scriptArgs := rest[0], rest[1:]
	_ = useCache

	if cpython {
		return runWithCPython(path, scriptArgs, stderr)
	}
	if inspect {
		return runWithInspect(path, scriptArgs, stdout, stderr)
	}
	if watchMode {
		return runWithWatch(path, scriptArgs, stdout, stderr)
	}
	return runFile(path, scriptArgs, stdout, stderr)
}

func runPYZ(path string, args []string) (int, error) {
	if err := bundler.RunPYZ(path, args); err != nil {
		return 1, err
	}
	return 0, nil
}

func runFile(path string, args []string, stdout, stderr io.Writer) (int, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return 1, err
	}
	return runtime.Run(path, src, args, stdout, stderr)
}

func runInline(code string, args []string, stdout, stderr io.Writer) (int, error) {
	return runtime.Run("<string>", []byte(code), args, stdout, stderr)
}

func runWithWatch(path string, args []string, stdout, stderr io.Writer) (int, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	setupSignalCancel(cancel)

	runOnce := func() {
		ts := time.Now().Format("15:04:05")
		fmt.Fprintf(stdout, "[%s] running %s\n", ts, path)
		runFile(path, args, stdout, stderr)
	}
	runOnce()

	mtimes := watchCollectMtimes(filepath.Dir(path))
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return 0, nil
		case <-ticker.C:
			cur := watchCollectMtimes(filepath.Dir(path))
			if watchMtimesChanged(mtimes, cur) {
				mtimes = cur
				ts := time.Now().Format("15:04:05")
				fmt.Fprintf(stdout, "[%s] restarting...\n", ts)
				runOnce()
			}
		}
	}
}

func watchCollectMtimes(root string) map[string]time.Time {
	m := map[string]time.Time{}
	filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if strings.HasSuffix(p, ".py") {
			if info, err := d.Info(); err == nil {
				m[p] = info.ModTime()
			}
		}
		return nil
	})
	return m
}

func watchMtimesChanged(old, cur map[string]time.Time) bool {
	if len(old) != len(cur) {
		return true
	}
	for k, t := range cur {
		if old[k] != t {
			return true
		}
	}
	return false
}

func runWithCPython(path string, args []string, stderr io.Writer) (int, error) {
	python, err := findPython3()
	if err != nil {
		return 1, fmt.Errorf("bunpy run --cpython requires Python 3 on PATH: %w", err)
	}
	cmd := exec.Command(python, append([]string{path}, args...)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return ee.ExitCode(), nil
		}
		return 1, err
	}
	return 0, nil
}

func findPython3() (string, error) {
	for _, candidate := range []string{"python3", "python"} {
		p, err := exec.LookPath(candidate)
		if err != nil {
			continue
		}
		// Reject Python 2.
		out, err := exec.Command(p, "--version").CombinedOutput()
		if err == nil && strings.HasPrefix(string(out), "Python 3") {
			return p, nil
		}
	}
	return "", fmt.Errorf("python3 not found")
}

func runWithInspect(path string, args []string, stdout, stderr io.Writer) (int, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return 1, err
	}

	fmt.Fprintf(stdout, "=== bunpy inspect: %s ===\n", path)

	t0 := time.Now()
	stream, compileErr := runtime.CompileToStream(path, src)
	compileTime := time.Since(t0)

	if compileErr != nil {
		fmt.Fprintf(stderr, "compile error: %v\n", compileErr)
		return 1, compileErr
	}
	fmt.Fprintf(stdout, "compile+marshal: %v\n", compileTime.Round(time.Microsecond))
	fmt.Fprintf(stdout, "ir bytes: %d\n", len(stream))

	// Hex dump — first 256 bytes.
	n := len(stream)
	if n > 256 {
		n = 256
	}
	fmt.Fprintf(stdout, "\n--- IR hex dump (first %d bytes) ---\n", n)
	fmt.Fprintf(stdout, "%s", hexDump(stream[:n]))
	fmt.Fprintf(stdout, "--- end IR ---\n\n")

	fmt.Fprintf(stdout, "=== running %s ===\n", path)
	return runtime.Run(path, src, args, stdout, stderr)
}

func hexDump(data []byte) string {
	var sb strings.Builder
	for i := 0; i < len(data); i += 16 {
		end := i + 16
		if end > len(data) {
			end = len(data)
		}
		chunk := data[i:end]
		fmt.Fprintf(&sb, "%04x  ", i)
		for j, b := range chunk {
			fmt.Fprintf(&sb, "%02x ", b)
			if j == 7 {
				sb.WriteByte(' ')
			}
		}
		// Pad short rows.
		for j := len(chunk); j < 16; j++ {
			sb.WriteString("   ")
			if j == 7 {
				sb.WriteByte(' ')
			}
		}
		sb.WriteString(" |")
		for _, b := range chunk {
			if b >= 32 && b < 127 {
				sb.WriteByte(b)
			} else {
				sb.WriteByte('.')
			}
		}
		sb.WriteString("|\n")
	}
	return sb.String()
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
