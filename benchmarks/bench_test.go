// Package benchmarks establishes the v0.12.x performance baseline.
// Run with: go test -bench=. -benchmem -benchtime=3s ./benchmarks/
// Before running, generate fixtures once: go run ./benchmarks/fixtures/build_fixtures.go
package benchmarks_test

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
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
// This is the v0.12.1 baseline — compare against BenchmarkInstallParallel_47pkgs.
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

// BenchmarkInstallParallel_47pkgs is the v0.12.2 parallel counterpart to
// BenchmarkInstall_47pkgs. It installs the same 47 pre-opened wheel archives
// using a bounded goroutine pool (GOMAXPROCS*2 workers), mirroring the
// production installPins implementation introduced in v0.12.2.
func BenchmarkInstallParallel_47pkgs(b *testing.B) {
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

	workers := runtime.GOMAXPROCS(0) * 2
	if workers > len(wheels) {
		workers = len(wheels)
	}
	if workers < 1 {
		workers = 1
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		target, _ := os.MkdirTemp("", "bench-install-par-*")
		b.StartTimer()

		jobs := make(chan *wheel.Wheel, len(wheels))
		for _, w := range wheels {
			jobs <- w
		}
		close(jobs)

		errc := make(chan error, workers)
		var wg sync.WaitGroup
		for range workers {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for w := range jobs {
					if _, werr := w.Install(target, wheel.InstallOptions{}); werr != nil {
						errc <- fmt.Errorf("install %s: %v", w.Filename, werr)
						return
					}
				}
			}()
		}
		wg.Wait()
		close(errc)

		b.StopTimer()
		os.RemoveAll(target)
		if werr := <-errc; werr != nil {
			b.Fatal(werr)
		}
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

// BenchmarkBuild_CacheHit measures the wall time of `bunpy build` when the
// build cache is warm (all inputs unchanged). On a cache hit the binary does
// nothing except verify file hashes and print "cache hit".
func BenchmarkBuild_CacheHit(b *testing.B) {
	// Create a tiny project in a temp directory.
	projectDir, err := os.MkdirTemp("", "bench-build-*")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(projectDir)

	entry := filepath.Join(projectDir, "app.py")
	if err := os.WriteFile(entry, []byte("x = 1\ny = 2\n"), 0o644); err != nil {
		b.Fatal(err)
	}
	outdir := filepath.Join(projectDir, "dist")

	// First build warms the cache (not timed).
	var warmOut bytes.Buffer
	warmCmd := exec.Command(bunpyBin, "build", entry, "--outdir="+outdir)
	warmCmd.Stdout = &warmOut
	warmCmd.Stderr = &warmOut
	if err := warmCmd.Run(); err != nil {
		b.Fatalf("warm build: %v\n%s", err, warmOut.String())
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var out bytes.Buffer
		cmd := exec.Command(bunpyBin, "build", entry, "--outdir="+outdir)
		cmd.Stdout = &out
		cmd.Stderr = &out
		if err := cmd.Run(); err != nil {
			b.Fatalf("cache-hit build: %v\n%s", err, out.String())
		}
		if !bytes.Contains(out.Bytes(), []byte("cache hit")) {
			b.Fatalf("expected cache hit but got: %s", out.String())
		}
	}
}

// BenchmarkBuild_CacheMiss measures the wall time of `bunpy build` on a cold
// cache (first build). This is the baseline that BenchmarkBuild_CacheHit
// improves upon.
func BenchmarkBuild_CacheMiss(b *testing.B) {
	// Keep the source outside the loop; only blow away the cache each iteration.
	projectDir, err := os.MkdirTemp("", "bench-build-miss-*")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(projectDir)

	entry := filepath.Join(projectDir, "app.py")
	if err := os.WriteFile(entry, []byte("x = 1\ny = 2\n"), 0o644); err != nil {
		b.Fatal(err)
	}
	outdir := filepath.Join(projectDir, "dist")
	cacheDir := filepath.Join(projectDir, ".bunpy")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		// Delete cache and output to force a full rebuild.
		os.RemoveAll(cacheDir)
		os.RemoveAll(outdir)
		b.StartTimer()

		var out bytes.Buffer
		cmd := exec.Command(bunpyBin, "build", entry, "--outdir="+outdir)
		cmd.Stdout = &out
		cmd.Stderr = &out
		if err := cmd.Run(); err != nil {
			b.Fatalf("cold build: %v\n%s", err, out.String())
		}
	}
}

// BenchmarkStartup_InlinePass measures the wall time for `bunpy -c "pass"` —
// the minimum-overhead startup path added in v0.12.8. No temp file, no disk
// read, no bunpy module factories loaded.
func BenchmarkStartup_InlinePass(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := exec.Command(bunpyBin, "-c", "pass").Run(); err != nil {
			b.Fatal(err)
		}
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
