// Package compare runs cross-tool benchmark comparisons for bunpy.
//
// Usage:
//
//	go test -bench=. -benchmem -benchtime=3s -count=3 ./benchmarks/compare/
//
// uv and CPython benchmarks are skipped gracefully if the respective binary
// is not found in PATH. Build bunpy before running:
//
//	go build -o /tmp/bunpy ./cmd/bunpy
//
// or let TestMain build it.
package compare_test

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/tamnd/bunpy/v1/benchmarks/compare"
)

var (
	bunpyBin     string
	fixtureIndex string
	serverURL    string
	serverStop   func()

	// realworld fixture server (started only when fixtures/realworld/index exists)
	rwServerURL  string
	rwServerStop func()
)

func TestMain(m *testing.M) {
	// Locate fixture index root (two levels up from this package).
	_, file, _, _ := runtime.Caller(0)
	repoRoot := filepath.Join(filepath.Dir(file), "..", "..")
	fixtureIndex = filepath.Join(repoRoot, "benchmarks", "fixtures", "index")

	// Start local fixture HTTP server.
	var err error
	serverURL, serverStop, err = compare.StartServer(fixtureIndex)
	if err != nil {
		fmt.Fprintln(os.Stderr, "compare: start server:", err)
		os.Exit(1)
	}
	defer serverStop()

	// Start realworld fixture server if the fixtures have been generated.
	rwIndex := filepath.Join(repoRoot, "benchmarks", "fixtures", "realworld", "index")
	if _, statErr := os.Stat(rwIndex); statErr == nil {
		rwServerURL, rwServerStop, err = compare.StartServer(rwIndex)
		if err != nil {
			fmt.Fprintln(os.Stderr, "compare: start realworld server:", err)
			os.Exit(1)
		}
		defer rwServerStop()
	}

	// Build bunpy binary once.
	tmp, err := os.MkdirTemp("", "bunpy-compare-*")
	if err != nil {
		fmt.Fprintln(os.Stderr, "compare: mktemp:", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmp)

	bunpyBin = filepath.Join(tmp, "bunpy")
	out, err := exec.Command("go", "build", "-o", bunpyBin,
		filepath.Join(repoRoot, "cmd", "bunpy")).CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "compare: build bunpy: %v\n%s\n", err, out)
		os.Exit(1)
	}
	// Warm the binary.
	_ = exec.Command(bunpyBin, "--version").Run()

	os.Exit(m.Run())
}

// BenchmarkLockBunpy_47pkgs measures `bunpy pm lock` on 47 flat deps using
// the local fixture HTTP server (no BUNPY_PYPI_FIXTURES — real HTTP stack).
// This is the bunpy side of the bunpy-vs-uv comparison.
func BenchmarkLockBunpy_47pkgs(b *testing.B) {
	manifest, err := readManifest()
	if err != nil {
		b.Fatal(err)
	}
	cacheBase, err := os.MkdirTemp("", "bunpy-cmp-cache-*")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(cacheBase)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		runDir, _ := os.MkdirTemp("", "bunpy-cmp-lock-*")
		iterCache, _ := os.MkdirTemp(cacheBase, "iter-*")
		_ = os.WriteFile(filepath.Join(runDir, "pyproject.toml"), manifest, 0o644)
		b.StartTimer()

		var out bytes.Buffer
		cmd := exec.Command(bunpyBin, "pm", "lock")
		cmd.Dir = runDir
		cmd.Env = append(os.Environ(),
			"BUNPY_PYPI_INDEX_URL="+serverURL+"/simple/",
			"BUNPY_CACHE_DIR="+iterCache,
		)
		cmd.Stdout = &out
		cmd.Stderr = &out
		runErr := cmd.Run()

		b.StopTimer()
		os.RemoveAll(runDir)
		os.RemoveAll(iterCache)
		if runErr != nil {
			b.Fatalf("bunpy pm lock: %v\n%s", runErr, out.String())
		}
		b.StartTimer()
	}
}

// BenchmarkLockUV_47pkgs measures `uv lock` on the same 47-package flat tree
// against the same local fixture server. Skips if uv is not in PATH.
func BenchmarkLockUV_47pkgs(b *testing.B) {
	uvBin, err := exec.LookPath("uv")
	if err != nil {
		b.Skip("uv not found in PATH; skipping")
	}

	manifest, err := readManifest()
	if err != nil {
		b.Fatal(err)
	}

	// uv needs requires-python to be present.
	uvManifest := append(manifest, []byte("\n[tool.uv]\nindex-url = \""+serverURL+"/simple/\"\n")...)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		runDir, _ := os.MkdirTemp("", "uv-cmp-lock-*")
		_ = os.WriteFile(filepath.Join(runDir, "pyproject.toml"), uvManifest, 0o644)
		b.StartTimer()

		var out bytes.Buffer
		cmd := exec.Command(uvBin, "lock", "--no-cache", "--no-progress")
		cmd.Dir = runDir
		cmd.Stdout = &out
		cmd.Stderr = &out
		runErr := cmd.Run()

		b.StopTimer()
		os.RemoveAll(runDir)
		if runErr != nil {
			b.Logf("uv lock output: %s", out.String())
			b.Fatalf("uv lock: %v", runErr)
		}
		b.StartTimer()
	}
}

// BenchmarkStartupBunpy measures wall time for a trivial script from fork to exit.
func BenchmarkStartupBunpy(b *testing.B) {
	_, file, _, _ := runtime.Caller(0)
	repoRoot := filepath.Join(filepath.Dir(file), "..", "..")
	script, err := filepath.Abs(filepath.Join(repoRoot, "benchmarks", "fixtures", "100tests", "test_000.py"))
	if err != nil {
		b.Fatal(err)
	}
	if _, err := os.Stat(script); err != nil {
		b.Fatalf("startup fixture missing: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := exec.Command(bunpyBin, script).Run(); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkStartupCPython measures wall time for CPython running the same
// trivial script. Skips if python3 is not in PATH.
func BenchmarkStartupCPython(b *testing.B) {
	py, err := exec.LookPath("python3")
	if err != nil {
		b.Skip("python3 not found in PATH; skipping")
	}

	_, file, _, _ := runtime.Caller(0)
	repoRoot := filepath.Join(filepath.Dir(file), "..", "..")
	script, err := filepath.Abs(filepath.Join(repoRoot, "benchmarks", "fixtures", "100tests", "test_000.py"))
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := exec.Command(py, script).Run(); err != nil {
			b.Fatal(err)
		}
	}
}

func readManifest() ([]byte, error) {
	_, file, _, _ := runtime.Caller(0)
	repoRoot := filepath.Join(filepath.Dir(file), "..", "..")
	return os.ReadFile(filepath.Join(repoRoot, "benchmarks", "fixtures", "47pkg", "pyproject.toml"))
}
