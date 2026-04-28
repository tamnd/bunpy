// Package benchmarks establishes the v0.12.x performance baseline.
// Run with: go test -bench=. -benchmem -benchtime=3s ./benchmarks/
// Before running, generate fixtures once: go run ./benchmarks/fixtures/build_fixtures.go
package benchmarks_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/tamnd/bunpy/v1/internal/testrunner"
	"github.com/tamnd/bunpy/v1/pkg/wheel"
)

// bunpyBin is the path to the bunpy binary built once in TestMain.
var bunpyBin string

func TestMain(m *testing.M) {
	tmp, err := os.MkdirTemp("", "bunpy-bench-*")
	if err != nil {
		panic("mktemp: " + err.Error())
	}
	defer os.RemoveAll(tmp)

	bunpyBin = filepath.Join(tmp, "bunpy")
	out, err := exec.Command("go", "build", "-o", bunpyBin, "../cmd/bunpy").CombinedOutput()
	if err != nil {
		panic("build bunpy: " + err.Error() + "\n" + string(out))
	}

	// Prime the binary in the OS page cache before any timed run.
	_ = exec.Command(bunpyBin, "--version").Run()

	os.Exit(m.Run())
}

// BenchmarkPMLock_47pkgs measures bunpy pm lock on a 47-package flat
// dependency tree using the fixture PyPI index (no network, no disk
// index cache — pure resolver + serialisation overhead).
func BenchmarkPMLock_47pkgs(b *testing.B) {
	indexAbs, err := filepath.Abs("fixtures/index")
	if err != nil {
		b.Fatal(err)
	}
	manifest, err := os.ReadFile("fixtures/47pkg/pyproject.toml")
	if err != nil {
		b.Fatalf("fixtures/47pkg/pyproject.toml missing; run: go run ./benchmarks/fixtures/build_fixtures.go")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		runDir, _ := os.MkdirTemp("", "bench-pmlock-*")
		_ = os.WriteFile(filepath.Join(runDir, "pyproject.toml"), manifest, 0o644)
		b.StartTimer()

		var out bytes.Buffer
		cmd := exec.Command(bunpyBin, "pm", "lock")
		cmd.Dir = runDir
		cmd.Env = append(os.Environ(), "BUNPY_PYPI_FIXTURES="+indexAbs)
		cmd.Stdout = &out
		cmd.Stderr = &out
		runErr := cmd.Run()

		b.StopTimer()
		os.RemoveAll(runDir)
		if runErr != nil {
			b.Fatalf("pm lock: %v\n%s", runErr, out.String())
		}
		b.StartTimer()
	}
}

// BenchmarkInstall_47pkgs measures sequential wheel installation for
// 47 pre-opened wheel archives into a fresh target directory.
func BenchmarkInstall_47pkgs(b *testing.B) {
	paths, err := filepath.Glob("fixtures/index/files.example/pkg*/*.whl")
	if err != nil || len(paths) == 0 {
		b.Fatalf("no wheel fixtures found; run: go run ./benchmarks/fixtures/build_fixtures.go")
	}

	wheels := make([]*wheel.Wheel, 0, len(paths))
	for _, p := range paths {
		w, werr := wheel.Open(p)
		if werr != nil {
			b.Fatalf("open wheel %s: %v", p, werr)
		}
		wheels = append(wheels, w)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		target, _ := os.MkdirTemp("", "bench-install-*")
		b.StartTimer()

		for _, w := range wheels {
			if _, werr := w.Install(target, wheel.InstallOptions{}); werr != nil {
				b.StopTimer()
				os.RemoveAll(target)
				b.Fatalf("install %s: %v", w.Filename, werr)
			}
		}

		b.StopTimer()
		os.RemoveAll(target)
		b.StartTimer()
	}
}

// BenchmarkTestRunner_100tests measures RunParallel on 100 synthetic
// test files (5 statements each) — one goroutine per file, all
// going through the full compile → marshal → unmarshal → VM pipeline.
func BenchmarkTestRunner_100tests(b *testing.B) {
	files, err := filepath.Glob("fixtures/100tests/test_*.py")
	if err != nil || len(files) == 0 {
		b.Fatalf("no test fixtures found; run: go run ./benchmarks/fixtures/build_fixtures.go")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = testrunner.RunParallel(files, testrunner.RunOptions{})
	}
}

// BenchmarkStartup measures the wall time for running a trivial Python
// script from fork to process exit — end-to-end cold-start overhead per
// invocation. Uses an existing fixture file to avoid temp-file overhead.
func BenchmarkStartup(b *testing.B) {
	// Pick the first test fixture as a minimal script; it's just
	// simple assignments that run instantly once compiled.
	script, err := filepath.Abs("fixtures/100tests/test_000.py")
	if err != nil {
		b.Fatal(err)
	}
	if _, err = os.Stat(script); err != nil {
		b.Fatalf("startup fixture missing; run: go run ./benchmarks/fixtures/build_fixtures.go")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := exec.Command(bunpyBin, script).Run(); err != nil {
			b.Fatal(err)
		}
	}
}
