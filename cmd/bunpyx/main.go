// Command bunpyx runs a Python package entry point in a temporary prefix.
// It is the Python analogue of bunx: fetch once, run once, clean up.
package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"github.com/tamnd/bunpy/v1/pkg/build"
	"github.com/tamnd/bunpy/v1/pkg/pypi"
	"github.com/tamnd/bunpy/v1/pkg/runenv"
	"github.com/tamnd/bunpy/v1/pkg/wheel"
	bunpyruntime "github.com/tamnd/bunpy/v1/runtime"
)

const version = "0.2.4"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	var (
		fromModule  string
		pythonPath  string
		cacheDir    string
		noCache     bool
		keep        bool
		pkgArg      string
	)

	positional := []string{}
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--help" || a == "-h":
			printUsage(stdout)
			return 0
		case a == "--version" || a == "-v":
			fmt.Fprintf(stdout, "bunpyx %s\n", version)
			return 0
		case a == "--from":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "bunpyx: --from requires an argument")
				return 1
			}
			i++
			fromModule = args[i]
		case strings.HasPrefix(a, "--from="):
			fromModule = strings.TrimPrefix(a, "--from=")
		case a == "--python":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "bunpyx: --python requires an argument")
				return 1
			}
			i++
			pythonPath = args[i]
		case strings.HasPrefix(a, "--python="):
			pythonPath = strings.TrimPrefix(a, "--python=")
		case a == "--cache-dir":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "bunpyx: --cache-dir requires an argument")
				return 1
			}
			i++
			cacheDir = args[i]
		case strings.HasPrefix(a, "--cache-dir="):
			cacheDir = strings.TrimPrefix(a, "--cache-dir=")
		case a == "--no-cache":
			noCache = true
		case a == "--keep":
			keep = true
		case strings.HasPrefix(a, "-"):
			fmt.Fprintf(stderr, "bunpyx: unknown flag %q\n", a)
			return 1
		default:
			positional = append(positional, a)
		}
	}

	if len(positional) == 0 {
		printUsage(stdout)
		return 0
	}

	pkgArg = positional[0]
	runArgs := positional[1:]

	pkgName, pkgVersion, _ := strings.Cut(pkgArg, "@")

	if pythonPath == "" {
		p, err := build.FindPython()
		if err != nil {
			fmt.Fprintf(stderr, "bunpyx: no Python found on PATH\n")
			return 1
		}
		pythonPath = p
	}

	if cacheDir == "" {
		home, err := os.UserHomeDir()
		if err == nil {
			cacheDir = filepath.Join(home, ".cache", "bunpyx", "wheels")
		}
	}

	wheelPath, err := resolveWheel(pkgName, pkgVersion, cacheDir, noCache, stderr)
	if err != nil {
		fmt.Fprintf(stderr, "bunpyx: %v\n", err)
		return 1
	}

	env, err := runenv.Create(pythonPath)
	if err != nil {
		fmt.Fprintf(stderr, "bunpyx: %v\n", err)
		return 1
	}
	if !keep {
		defer env.Cleanup()
	} else {
		fmt.Fprintf(stderr, "bunpyx: keeping temp prefix at %s\n", env.Dir)
	}

	if err := env.Install(wheelPath); err != nil {
		fmt.Fprintf(stderr, "bunpyx: %v\n", err)
		return 1
	}

	sitePackages := filepath.Join(env.Dir, "site-packages")
	entryName := pkgName

	if fromModule != "" {
		return execPython(pythonPath, sitePackages, append([]string{"-m", fromModule}, runArgs...), stderr)
	}

	shimPath, ok := env.EntryPoint(entryName)
	if !ok {
		fmt.Fprintf(stderr, "bunpyx: entry point %q not found in %s\n", entryName, pkgArg)
		return 1
	}

	return execShim(shimPath, sitePackages, runArgs, stderr)
}

// resolveWheel finds or downloads the wheel for pkgName@version.
func resolveWheel(pkgName, pkgVersion, cacheDir string, noCache bool, stderr io.Writer) (string, error) {
	// check cache first
	if cacheDir != "" && !noCache {
		if p := findCachedWheel(cacheDir, pkgName, pkgVersion); p != "" {
			return p, nil
		}
	}

	// fetch from PyPI
	client := pypi.New()
	client.UserAgent = "bunpyx/" + version
	project, err := client.Get(context.Background(), pkgName)
	if err != nil {
		return "", fmt.Errorf("not found: %s: %w", pkgName, err)
	}

	targetVersion := pkgVersion
	if targetVersion == "" && len(project.Versions) > 0 {
		targetVersion = project.Versions[len(project.Versions)-1]
	}

	// filter to target version
	var candidates []pypi.File
	for _, f := range project.Files {
		if f.Version == targetVersion {
			candidates = append(candidates, f)
		}
	}

	chosen, ok := wheel.Pick(candidates, wheel.HostTags())
	if !ok {
		return "", fmt.Errorf("no wheel found for %s@%s", pkgName, targetVersion)
	}

	// download
	body, err := downloadURL(chosen.URL, "bunpyx/"+version)
	if err != nil {
		return "", fmt.Errorf("download %s: %w", chosen.Filename, err)
	}

	// write to cache if configured
	if cacheDir != "" {
		_ = os.MkdirAll(cacheDir, 0o755)
		dest := filepath.Join(cacheDir, chosen.Filename)
		_ = os.WriteFile(dest, body, 0o644)
		return dest, nil
	}

	// write to temp file
	tmp, err := os.CreateTemp("", "bunpyx-*.whl")
	if err != nil {
		return "", fmt.Errorf("create temp wheel: %w", err)
	}
	if _, err := tmp.Write(body); err != nil {
		tmp.Close()
		return "", fmt.Errorf("write temp wheel: %w", err)
	}
	tmp.Close()
	return tmp.Name(), nil
}

func downloadURL(url, userAgent string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	if userAgent != "" {
		req.Header.Set("User-Agent", userAgent)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("HTTP %s", resp.Status)
	}
	return io.ReadAll(resp.Body)
}

func findCachedWheel(cacheDir, pkgName, pkgVersion string) string {
	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		return ""
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".whl") {
			continue
		}
		parts := strings.SplitN(name, "-", 3)
		if len(parts) < 2 {
			continue
		}
		if !strings.EqualFold(parts[0], strings.ReplaceAll(pkgName, "-", "_")) {
			continue
		}
		if pkgVersion != "" && parts[1] != pkgVersion {
			continue
		}
		return filepath.Join(cacheDir, name)
	}
	return ""
}

// execShim execs the shim (Unix: syscall.Exec; Windows: StartProcess + Exit).
func execShim(shimPath, sitePackages string, args []string, stderr io.Writer) int {
	env := shimEnv(sitePackages)
	if runtime.GOOS == "windows" {
		cmd := exec.Command(shimPath, args...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Env = env
		if err := cmd.Run(); err != nil {
			if ex, ok := err.(*exec.ExitError); ok {
				return ex.ExitCode()
			}
			fmt.Fprintln(stderr, "bunpyx:", err)
			return 1
		}
		return 0
	}
	argv := append([]string{shimPath}, args...)
	if err := syscall.Exec(shimPath, argv, env); err != nil {
		fmt.Fprintln(stderr, "bunpyx:", err)
		return 1
	}
	return 0
}

// execPython runs python -m module (no exec on Windows, syscall.Exec on Unix).
func execPython(pythonPath, sitePackages string, args []string, stderr io.Writer) int {
	env := shimEnv(sitePackages)
	if runtime.GOOS == "windows" {
		cmd := exec.Command(pythonPath, args...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Env = env
		if err := cmd.Run(); err != nil {
			if ex, ok := err.(*exec.ExitError); ok {
				return ex.ExitCode()
			}
			fmt.Fprintln(stderr, "bunpyx:", err)
			return 1
		}
		return 0
	}
	argv := append([]string{pythonPath}, args...)
	if err := syscall.Exec(pythonPath, argv, env); err != nil {
		fmt.Fprintln(stderr, "bunpyx:", err)
		return 1
	}
	return 0
}

func shimEnv(sitePackages string) []string {
	env := os.Environ()
	for i, e := range env {
		if strings.HasPrefix(e, "PYTHONPATH=") {
			env[i] = "PYTHONPATH=" + sitePackages + string(os.PathListSeparator) + strings.TrimPrefix(e, "PYTHONPATH=")
			return env
		}
	}
	return append(env, "PYTHONPATH="+sitePackages)
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "bunpyx: run a Python package without installing it permanently.")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "USAGE")
	fmt.Fprintln(w, "  bunpyx <pkg>[@version] [args...]")
	fmt.Fprintln(w, "  bunpyx --from <module> <pkg>[@version] [args...]")
	fmt.Fprintln(w, "  bunpyx --version")
	fmt.Fprintln(w, "  bunpyx --help")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "OPTIONS")
	fmt.Fprintln(w, "  --from <module>       run python -m <module> instead of the default entry point")
	fmt.Fprintln(w, "  --python <path>       path to the Python executable (default: python3 on PATH)")
	fmt.Fprintln(w, "  --cache-dir <dir>     wheel cache directory (default: ~/.cache/bunpyx/wheels)")
	fmt.Fprintln(w, "  --no-cache            skip reading from the wheel cache; always fetch from PyPI")
	fmt.Fprintln(w, "  --keep                keep the temp prefix after the process exits (prints path to stderr)")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "EXAMPLES")
	fmt.Fprintln(w, "  bunpyx black .")
	fmt.Fprintln(w, "  bunpyx black@24.10.0 .")
	fmt.Fprintln(w, "  bunpyx --from black black --version")
	fmt.Fprintln(w, "  bunpyx ruff check .")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Exit code mirrors the invoked process. See bunpyx(1) for the full manpage.")
}

// bunpyruntime is imported to ensure version ldflags are consistent.
var _ = bunpyruntime.Version
