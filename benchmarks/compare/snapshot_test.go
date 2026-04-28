package compare_test

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// snapDir returns the path to benchmarks/fixtures/snapshot.
func snapDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "fixtures", "snapshot")
}

// skipIfNoSnapshot skips b/t if snapshot fixtures have not been generated.
func skipIfNoSnapshot(tb testing.TB) {
	tb.Helper()
	if _, err := os.Stat(snapDir()); err != nil {
		tb.Skip("snapshot fixtures not found; run: go run benchmarks/fixtures/build_snapshot.go")
	}
	if snapServerURL == "" {
		tb.Skip("snapshot fixture server not started (fixtures missing at test startup)")
	}
}

// pipBin returns the path to pip (python3 -m pip), or "" if not found.
func pipBin() string {
	py, err := exec.LookPath("python3")
	if err != nil {
		return ""
	}
	return py
}

// snapProfile reads a pyproject.toml from the snapshot fixture dir.
func snapProfile(tb testing.TB, profile string) []byte {
	tb.Helper()
	path := filepath.Join(snapDir(), profile, "pyproject.toml")
	data, err := os.ReadFile(path)
	if err != nil {
		tb.Fatalf("read snapshot profile %s: %v", profile, err)
	}
	return data
}

// --- Resolve (lock) benchmarks ---

// BenchmarkResolveCold_Bunpy_RequestsHTTPX measures bunpy pm lock on the
// requests-httpx profile with a fresh empty cache on every iteration.
func BenchmarkResolveCold_Bunpy_RequestsHTTPX(b *testing.B) {
	skipIfNoSnapshot(b)
	manifest := snapProfile(b, "requests-httpx")
	manifest = append(manifest, []byte("\n[tool.bunpy]\nindex-url = \""+snapServerURL+"/simple/\"\n")...)
	benchmarkBunpyLockCold(b, manifest)
}

// BenchmarkResolveCold_UV_RequestsHTTPX measures uv lock on the same profile,
// no cache (--no-cache). Skips if uv is not in PATH.
func BenchmarkResolveCold_UV_RequestsHTTPX(b *testing.B) {
	skipIfNoSnapshot(b)
	if uvBin() == "" {
		b.Skip("uv not found in PATH")
	}
	manifest := snapProfile(b, "requests-httpx")
	benchmarkUVLockCold(b, manifest)
}

// BenchmarkResolveCold_Pip_RequestsHTTPX measures pip-compile (via uv pip
// compile) on the same profile. Skips if python3 is not in PATH.
func BenchmarkResolveCold_Pip_RequestsHTTPX(b *testing.B) {
	skipIfNoSnapshot(b)
	if pipBin() == "" {
		b.Skip("python3 not found in PATH")
	}
	manifest := snapProfile(b, "requests-httpx")
	benchmarkPipCompileCold(b, manifest)
}

func BenchmarkResolveCold_Bunpy_HTTPXRich(b *testing.B) {
	skipIfNoSnapshot(b)
	manifest := snapProfile(b, "httpx-rich")
	manifest = append(manifest, []byte("\n[tool.bunpy]\nindex-url = \""+snapServerURL+"/simple/\"\n")...)
	benchmarkBunpyLockCold(b, manifest)
}

func BenchmarkResolveCold_UV_HTTPXRich(b *testing.B) {
	skipIfNoSnapshot(b)
	if uvBin() == "" {
		b.Skip("uv not found in PATH")
	}
	manifest := snapProfile(b, "httpx-rich")
	benchmarkUVLockCold(b, manifest)
}

func BenchmarkResolveCold_Pip_HTTPXRich(b *testing.B) {
	skipIfNoSnapshot(b)
	if pipBin() == "" {
		b.Skip("python3 not found in PATH")
	}
	manifest := snapProfile(b, "httpx-rich")
	benchmarkPipCompileCold(b, manifest)
}

// BenchmarkResolveWarm_* — same as Cold but with a pre-warmed cache.

func BenchmarkResolveWarm_Bunpy_RequestsHTTPX(b *testing.B) {
	skipIfNoSnapshot(b)
	manifest := snapProfile(b, "requests-httpx")
	manifest = append(manifest, []byte("\n[tool.bunpy]\nindex-url = \""+snapServerURL+"/simple/\"\n")...)
	benchmarkBunpyLockWarm(b, manifest)
}

func BenchmarkResolveWarm_UV_RequestsHTTPX(b *testing.B) {
	skipIfNoSnapshot(b)
	if uvBin() == "" {
		b.Skip("uv not found in PATH")
	}
	manifest := snapProfile(b, "requests-httpx")
	benchmarkUVLockWarm(b, manifest)
}

func BenchmarkResolveWarm_Pip_RequestsHTTPX(b *testing.B) {
	skipIfNoSnapshot(b)
	if pipBin() == "" {
		b.Skip("python3 not found in PATH")
	}
	manifest := snapProfile(b, "requests-httpx")
	benchmarkPipCompileWarm(b, manifest)
}

func BenchmarkResolveWarm_Bunpy_HTTPXRich(b *testing.B) {
	skipIfNoSnapshot(b)
	manifest := snapProfile(b, "httpx-rich")
	manifest = append(manifest, []byte("\n[tool.bunpy]\nindex-url = \""+snapServerURL+"/simple/\"\n")...)
	benchmarkBunpyLockWarm(b, manifest)
}

func BenchmarkResolveWarm_UV_HTTPXRich(b *testing.B) {
	skipIfNoSnapshot(b)
	if uvBin() == "" {
		b.Skip("uv not found in PATH")
	}
	manifest := snapProfile(b, "httpx-rich")
	benchmarkUVLockWarm(b, manifest)
}

func BenchmarkResolveWarm_Pip_HTTPXRich(b *testing.B) {
	skipIfNoSnapshot(b)
	if pipBin() == "" {
		b.Skip("python3 not found in PATH")
	}
	manifest := snapProfile(b, "httpx-rich")
	benchmarkPipCompileWarm(b, manifest)
}

// --- Install benchmarks ---

// BenchmarkInstallCold_* measures installing from a pre-generated lockfile
// with a fresh empty wheel cache on every iteration.

func BenchmarkInstallCold_Bunpy_RequestsHTTPX(b *testing.B) {
	skipIfNoSnapshot(b)
	manifest := snapProfile(b, "requests-httpx")
	manifest = append(manifest, []byte("\n[tool.bunpy]\nindex-url = \""+snapServerURL+"/simple/\"\n")...)
	benchmarkBunpyInstallCold(b, "requests-httpx", manifest)
}

func BenchmarkInstallCold_UV_RequestsHTTPX(b *testing.B) {
	skipIfNoSnapshot(b)
	if uvBin() == "" {
		b.Skip("uv not found in PATH")
	}
	manifest := snapProfile(b, "requests-httpx")
	benchmarkUVInstallCold(b, manifest)
}

func BenchmarkInstallCold_Pip_RequestsHTTPX(b *testing.B) {
	skipIfNoSnapshot(b)
	if pipBin() == "" {
		b.Skip("python3 not found in PATH")
	}
	manifest := snapProfile(b, "requests-httpx")
	benchmarkPipInstallCold(b, manifest)
}

func BenchmarkInstallCold_Bunpy_HTTPXRich(b *testing.B) {
	skipIfNoSnapshot(b)
	manifest := snapProfile(b, "httpx-rich")
	manifest = append(manifest, []byte("\n[tool.bunpy]\nindex-url = \""+snapServerURL+"/simple/\"\n")...)
	benchmarkBunpyInstallCold(b, "httpx-rich", manifest)
}

func BenchmarkInstallCold_UV_HTTPXRich(b *testing.B) {
	skipIfNoSnapshot(b)
	if uvBin() == "" {
		b.Skip("uv not found in PATH")
	}
	manifest := snapProfile(b, "httpx-rich")
	benchmarkUVInstallCold(b, manifest)
}

func BenchmarkInstallCold_Pip_HTTPXRich(b *testing.B) {
	skipIfNoSnapshot(b)
	if pipBin() == "" {
		b.Skip("python3 not found in PATH")
	}
	manifest := snapProfile(b, "httpx-rich")
	benchmarkPipInstallCold(b, manifest)
}

// BenchmarkInstallWarm_* — same as Cold but with a pre-warmed wheel cache.

func BenchmarkInstallWarm_Bunpy_RequestsHTTPX(b *testing.B) {
	skipIfNoSnapshot(b)
	manifest := snapProfile(b, "requests-httpx")
	manifest = append(manifest, []byte("\n[tool.bunpy]\nindex-url = \""+snapServerURL+"/simple/\"\n")...)
	benchmarkBunpyInstallWarm(b, "requests-httpx", manifest)
}

func BenchmarkInstallWarm_UV_RequestsHTTPX(b *testing.B) {
	skipIfNoSnapshot(b)
	if uvBin() == "" {
		b.Skip("uv not found in PATH")
	}
	manifest := snapProfile(b, "requests-httpx")
	benchmarkUVInstallWarm(b, manifest)
}

func BenchmarkInstallWarm_Pip_RequestsHTTPX(b *testing.B) {
	skipIfNoSnapshot(b)
	if pipBin() == "" {
		b.Skip("python3 not found in PATH")
	}
	manifest := snapProfile(b, "requests-httpx")
	benchmarkPipInstallWarm(b, manifest)
}

func BenchmarkInstallWarm_Bunpy_HTTPXRich(b *testing.B) {
	skipIfNoSnapshot(b)
	manifest := snapProfile(b, "httpx-rich")
	manifest = append(manifest, []byte("\n[tool.bunpy]\nindex-url = \""+snapServerURL+"/simple/\"\n")...)
	benchmarkBunpyInstallWarm(b, "httpx-rich", manifest)
}

func BenchmarkInstallWarm_UV_HTTPXRich(b *testing.B) {
	skipIfNoSnapshot(b)
	if uvBin() == "" {
		b.Skip("uv not found in PATH")
	}
	manifest := snapProfile(b, "httpx-rich")
	benchmarkUVInstallWarm(b, manifest)
}

func BenchmarkInstallWarm_Pip_HTTPXRich(b *testing.B) {
	skipIfNoSnapshot(b)
	if pipBin() == "" {
		b.Skip("python3 not found in PATH")
	}
	manifest := snapProfile(b, "httpx-rich")
	benchmarkPipInstallWarm(b, manifest)
}

// --- Shared benchmark helpers ---

func benchmarkBunpyLockCold(b *testing.B, manifest []byte) {
	b.Helper()
	cacheBase := b.TempDir()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		runDir, _ := os.MkdirTemp("", "snap-bunpy-cold-*")
		iterCache, _ := os.MkdirTemp(cacheBase, "iter-*")
		_ = os.WriteFile(filepath.Join(runDir, "pyproject.toml"), manifest, 0o644)
		b.StartTimer()

		var out bytes.Buffer
		cmd := exec.Command(bunpyBin, "pm", "lock")
		cmd.Dir = runDir
		cmd.Env = append(os.Environ(),
			"BUNPY_PYPI_INDEX_URL="+snapServerURL+"/simple/",
			"BUNPY_CACHE_DIR="+iterCache,
		)
		cmd.Stdout, cmd.Stderr = &out, &out
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

func benchmarkBunpyLockWarm(b *testing.B, manifest []byte) {
	b.Helper()
	// Pre-warm: run once to populate cache.
	warmCache := b.TempDir()
	warmDir := b.TempDir()
	_ = os.WriteFile(filepath.Join(warmDir, "pyproject.toml"), manifest, 0o644)
	warmCmd := exec.Command(bunpyBin, "pm", "lock")
	warmCmd.Dir = warmDir
	warmCmd.Env = append(os.Environ(),
		"BUNPY_PYPI_INDEX_URL="+snapServerURL+"/simple/",
		"BUNPY_CACHE_DIR="+warmCache,
	)
	if out, err := warmCmd.CombinedOutput(); err != nil {
		b.Fatalf("warm-up bunpy pm lock: %v\n%s", err, out)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		runDir, _ := os.MkdirTemp("", "snap-bunpy-warm-*")
		_ = os.WriteFile(filepath.Join(runDir, "pyproject.toml"), manifest, 0o644)
		b.StartTimer()

		var out bytes.Buffer
		cmd := exec.Command(bunpyBin, "pm", "lock")
		cmd.Dir = runDir
		cmd.Env = append(os.Environ(),
			"BUNPY_PYPI_INDEX_URL="+snapServerURL+"/simple/",
			"BUNPY_CACHE_DIR="+warmCache,
		)
		cmd.Stdout, cmd.Stderr = &out, &out
		runErr := cmd.Run()

		b.StopTimer()
		os.RemoveAll(runDir)
		if runErr != nil {
			b.Fatalf("bunpy pm lock (warm): %v\n%s", runErr, out.String())
		}
		b.StartTimer()
	}
}

func uvManifest(manifest []byte) []byte {
	return append(manifest, []byte(fmt.Sprintf("\n[tool.uv]\nindex-url = %q\n", snapServerURL+"/simple/"))...)
}

func benchmarkUVLockCold(b *testing.B, manifest []byte) {
	b.Helper()
	toml := uvManifest(manifest)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		runDir, _ := os.MkdirTemp("", "snap-uv-cold-*")
		_ = os.WriteFile(filepath.Join(runDir, "pyproject.toml"), toml, 0o644)
		b.StartTimer()

		var out bytes.Buffer
		cmd := exec.Command(uvBin(), "lock", "--no-cache", "--no-progress")
		cmd.Dir = runDir
		cmd.Stdout, cmd.Stderr = &out, &out
		runErr := cmd.Run()

		b.StopTimer()
		os.RemoveAll(runDir)
		if runErr != nil {
			b.Fatalf("uv lock: %v\n%s", runErr, out.String())
		}
		b.StartTimer()
	}
}

func benchmarkUVLockWarm(b *testing.B, manifest []byte) {
	b.Helper()
	toml := uvManifest(manifest)
	uvCache := b.TempDir()

	// Pre-warm uv cache.
	warmDir := b.TempDir()
	_ = os.WriteFile(filepath.Join(warmDir, "pyproject.toml"), toml, 0o644)
	warmCmd := exec.Command(uvBin(), "lock", "--no-progress")
	warmCmd.Dir = warmDir
	warmCmd.Env = append(os.Environ(), "UV_CACHE_DIR="+uvCache)
	if out, err := warmCmd.CombinedOutput(); err != nil {
		b.Fatalf("warm-up uv lock: %v\n%s", err, out)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		runDir, _ := os.MkdirTemp("", "snap-uv-warm-*")
		_ = os.WriteFile(filepath.Join(runDir, "pyproject.toml"), toml, 0o644)
		b.StartTimer()

		var out bytes.Buffer
		cmd := exec.Command(uvBin(), "lock", "--no-progress")
		cmd.Dir = runDir
		cmd.Env = append(os.Environ(), "UV_CACHE_DIR="+uvCache)
		cmd.Stdout, cmd.Stderr = &out, &out
		runErr := cmd.Run()

		b.StopTimer()
		os.RemoveAll(runDir)
		if runErr != nil {
			b.Fatalf("uv lock (warm): %v\n%s", runErr, out.String())
		}
		b.StartTimer()
	}
}

// benchmarkPipCompileCold uses `pip-compile` (via uv pip compile) to resolve
// dependencies. Falls back to uv pip compile if pip-tools is not installed.
func benchmarkPipCompileCold(b *testing.B, manifest []byte) {
	b.Helper()
	reqs := manifestToRequirements(manifest)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		runDir, _ := os.MkdirTemp("", "snap-pip-cold-*")
		reqFile := filepath.Join(runDir, "requirements.in")
		_ = os.WriteFile(reqFile, reqs, 0o644)
		b.StartTimer()

		var out bytes.Buffer
		cmd := exec.Command(pipBin(), "-m", "piptools", "compile",
			"--no-header", "--quiet",
			"--index-url", snapServerURL+"/simple/",
			"--no-cache", reqFile,
		)
		cmd.Dir = runDir
		cmd.Stdout, cmd.Stderr = &out, &out
		runErr := cmd.Run()

		b.StopTimer()
		os.RemoveAll(runDir)
		if runErr != nil {
			// pip-tools not available; skip rather than fail
			b.Skipf("pip-tools (pip-compile) not available: %v\n%s", runErr, out.String())
		}
		b.StartTimer()
	}
}

func benchmarkPipCompileWarm(b *testing.B, manifest []byte) {
	b.Helper()
	reqs := manifestToRequirements(manifest)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		runDir, _ := os.MkdirTemp("", "snap-pip-warm-*")
		reqFile := filepath.Join(runDir, "requirements.in")
		_ = os.WriteFile(reqFile, reqs, 0o644)
		b.StartTimer()

		var out bytes.Buffer
		cmd := exec.Command(pipBin(), "-m", "piptools", "compile",
			"--no-header", "--quiet",
			"--index-url", snapServerURL+"/simple/",
			reqFile,
		)
		cmd.Dir = runDir
		cmd.Stdout, cmd.Stderr = &out, &out
		runErr := cmd.Run()

		b.StopTimer()
		os.RemoveAll(runDir)
		if runErr != nil {
			b.Skipf("pip-tools (pip-compile) not available: %v\n%s", runErr, out.String())
		}
		b.StartTimer()
	}
}

// Install benchmark helpers.

func bunpyLockfile(b *testing.B, manifest []byte) []byte {
	b.Helper()
	return snapLockBunpy(b, manifest)
}

func benchmarkBunpyInstallCold(b *testing.B, _ string, manifest []byte) {
	b.Helper()
	lockData := bunpyLockfile(b, manifest)

	cacheBase := b.TempDir()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		runDir, _ := os.MkdirTemp("", "snap-bunpy-install-cold-*")
		iterCache, _ := os.MkdirTemp(cacheBase, "iter-*")
		_ = os.WriteFile(filepath.Join(runDir, "pyproject.toml"), manifest, 0o644)
		_ = os.WriteFile(filepath.Join(runDir, "uv.lock"), lockData, 0o644)
		b.StartTimer()

		var out bytes.Buffer
		cmd := exec.Command(bunpyBin, "install")
		cmd.Dir = runDir
		cmd.Env = append(os.Environ(),
			"BUNPY_PYPI_INDEX_URL="+snapServerURL+"/simple/",
			"BUNPY_CACHE_DIR="+iterCache,
		)
		cmd.Stdout, cmd.Stderr = &out, &out
		runErr := cmd.Run()

		b.StopTimer()
		os.RemoveAll(runDir)
		os.RemoveAll(iterCache)
		if runErr != nil {
			b.Fatalf("bunpy install: %v\n%s", runErr, out.String())
		}
		b.StartTimer()
	}
}

func benchmarkBunpyInstallWarm(b *testing.B, _ string, manifest []byte) {
	b.Helper()
	lockData := bunpyLockfile(b, manifest)

	// Pre-warm wheel cache.
	warmCache := b.TempDir()
	warmDir := b.TempDir()
	_ = os.WriteFile(filepath.Join(warmDir, "pyproject.toml"), manifest, 0o644)
	_ = os.WriteFile(filepath.Join(warmDir, "uv.lock"), lockData, 0o644)
	warmCmd := exec.Command(bunpyBin, "install")
	warmCmd.Dir = warmDir
	warmCmd.Env = append(os.Environ(),
		"BUNPY_PYPI_INDEX_URL="+snapServerURL+"/simple/",
		"BUNPY_CACHE_DIR="+warmCache,
	)
	if out, err := warmCmd.CombinedOutput(); err != nil {
		b.Fatalf("warm-up bunpy install: %v\n%s", err, out)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		runDir, _ := os.MkdirTemp("", "snap-bunpy-install-warm-*")
		_ = os.WriteFile(filepath.Join(runDir, "pyproject.toml"), manifest, 0o644)
		_ = os.WriteFile(filepath.Join(runDir, "uv.lock"), lockData, 0o644)
		b.StartTimer()

		var out bytes.Buffer
		cmd := exec.Command(bunpyBin, "install")
		cmd.Dir = runDir
		cmd.Env = append(os.Environ(),
			"BUNPY_PYPI_INDEX_URL="+snapServerURL+"/simple/",
			"BUNPY_CACHE_DIR="+warmCache,
		)
		cmd.Stdout, cmd.Stderr = &out, &out
		runErr := cmd.Run()

		b.StopTimer()
		os.RemoveAll(runDir)
		if runErr != nil {
			b.Fatalf("bunpy install (warm): %v\n%s", runErr, out.String())
		}
		b.StartTimer()
	}
}

func benchmarkUVInstallCold(b *testing.B, manifest []byte) {
	b.Helper()
	toml := uvManifest(manifest)

	// Generate uv.lock once before the timed loop.
	lockDir := b.TempDir()
	_ = os.WriteFile(filepath.Join(lockDir, "pyproject.toml"), toml, 0o644)
	lockCmd := exec.Command(uvBin(), "lock", "--no-progress", "--no-cache")
	lockCmd.Dir = lockDir
	if out, err := lockCmd.CombinedOutput(); err != nil {
		b.Fatalf("generate uv lockfile: %v\n%s", err, out)
	}
	lockData, err := os.ReadFile(filepath.Join(lockDir, "uv.lock"))
	if err != nil {
		b.Fatalf("read uv lockfile: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		runDir, _ := os.MkdirTemp("", "snap-uv-install-cold-*")
		_ = os.WriteFile(filepath.Join(runDir, "pyproject.toml"), toml, 0o644)
		_ = os.WriteFile(filepath.Join(runDir, "uv.lock"), lockData, 0o644)
		b.StartTimer()

		var out bytes.Buffer
		cmd := exec.Command(uvBin(), "sync", "--frozen", "--no-progress", "--no-cache")
		cmd.Dir = runDir
		cmd.Stdout, cmd.Stderr = &out, &out
		runErr := cmd.Run()

		b.StopTimer()
		os.RemoveAll(runDir)
		if runErr != nil {
			b.Fatalf("uv sync: %v\n%s", runErr, out.String())
		}
		b.StartTimer()
	}
}

func benchmarkUVInstallWarm(b *testing.B, manifest []byte) {
	b.Helper()
	toml := uvManifest(manifest)
	uvCache := b.TempDir()

	// Generate lockfile.
	lockDir := b.TempDir()
	_ = os.WriteFile(filepath.Join(lockDir, "pyproject.toml"), toml, 0o644)
	lockCmd := exec.Command(uvBin(), "lock", "--no-progress")
	lockCmd.Dir = lockDir
	lockCmd.Env = append(os.Environ(), "UV_CACHE_DIR="+uvCache)
	if out, err := lockCmd.CombinedOutput(); err != nil {
		b.Fatalf("generate uv lockfile: %v\n%s", err, out)
	}
	lockData, _ := os.ReadFile(filepath.Join(lockDir, "uv.lock"))

	// Pre-warm uv cache with a sync.
	warmDir := b.TempDir()
	_ = os.WriteFile(filepath.Join(warmDir, "pyproject.toml"), toml, 0o644)
	_ = os.WriteFile(filepath.Join(warmDir, "uv.lock"), lockData, 0o644)
	warmCmd := exec.Command(uvBin(), "sync", "--frozen", "--no-progress")
	warmCmd.Dir = warmDir
	warmCmd.Env = append(os.Environ(), "UV_CACHE_DIR="+uvCache)
	if out, err := warmCmd.CombinedOutput(); err != nil {
		b.Fatalf("warm-up uv sync: %v\n%s", err, out)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		runDir, _ := os.MkdirTemp("", "snap-uv-install-warm-*")
		_ = os.WriteFile(filepath.Join(runDir, "pyproject.toml"), toml, 0o644)
		_ = os.WriteFile(filepath.Join(runDir, "uv.lock"), lockData, 0o644)
		b.StartTimer()

		var out bytes.Buffer
		cmd := exec.Command(uvBin(), "sync", "--frozen", "--no-progress")
		cmd.Dir = runDir
		cmd.Env = append(os.Environ(), "UV_CACHE_DIR="+uvCache)
		cmd.Stdout, cmd.Stderr = &out, &out
		runErr := cmd.Run()

		b.StopTimer()
		os.RemoveAll(runDir)
		if runErr != nil {
			b.Fatalf("uv sync (warm): %v\n%s", runErr, out.String())
		}
		b.StartTimer()
	}
}

func benchmarkPipInstallCold(b *testing.B, manifest []byte) {
	b.Helper()
	pkgs := snapPinnedPackages(b, manifest)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		targetDir, _ := os.MkdirTemp("", "snap-pip-install-cold-*")
		b.StartTimer()

		args := append([]string{"-m", "pip", "install",
			"--no-deps",
			"--no-cache-dir",
			"--target", targetDir,
			"--index-url", snapServerURL + "/simple/",
			"--quiet",
		}, pkgs...)
		var out bytes.Buffer
		cmd := exec.Command(pipBin(), args...)
		cmd.Stdout, cmd.Stderr = &out, &out
		runErr := cmd.Run()

		b.StopTimer()
		os.RemoveAll(targetDir)
		if runErr != nil {
			b.Fatalf("pip install: %v\n%s", runErr, out.String())
		}
		b.StartTimer()
	}
}

func benchmarkPipInstallWarm(b *testing.B, manifest []byte) {
	b.Helper()
	pkgs := snapPinnedPackages(b, manifest)
	pipCache := b.TempDir()

	// Pre-warm pip wheel cache.
	warmTarget := b.TempDir()
	warmArgs := append([]string{"-m", "pip", "install",
		"--no-deps",
		"--cache-dir", pipCache,
		"--target", warmTarget,
		"--index-url", snapServerURL + "/simple/",
		"--quiet",
	}, pkgs...)
	if out, err := exec.Command(pipBin(), warmArgs...).CombinedOutput(); err != nil {
		b.Fatalf("warm-up pip install: %v\n%s", err, out)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		targetDir, _ := os.MkdirTemp("", "snap-pip-install-warm-*")
		b.StartTimer()

		args := append([]string{"-m", "pip", "install",
			"--no-deps",
			"--cache-dir", pipCache,
			"--target", targetDir,
			"--index-url", snapServerURL + "/simple/",
			"--quiet",
		}, pkgs...)
		var out bytes.Buffer
		cmd := exec.Command(pipBin(), args...)
		cmd.Stdout, cmd.Stderr = &out, &out
		runErr := cmd.Run()

		b.StopTimer()
		os.RemoveAll(targetDir)
		if runErr != nil {
			b.Fatalf("pip install (warm): %v\n%s", runErr, out.String())
		}
		b.StartTimer()
	}
}

// snapLockBunpy runs bunpy pm lock against the snapshot fixture server and
// returns the generated uv.lock bytes.
func snapLockBunpy(tb testing.TB, pyproject []byte) []byte {
	tb.Helper()
	runDir := tb.TempDir()
	cacheDir := tb.TempDir()
	if err := os.WriteFile(filepath.Join(runDir, "pyproject.toml"), pyproject, 0o644); err != nil {
		tb.Fatal(err)
	}
	var out bytes.Buffer
	cmd := exec.Command(bunpyBin, "pm", "lock")
	cmd.Dir = runDir
	cmd.Env = append(os.Environ(),
		"BUNPY_PYPI_INDEX_URL="+snapServerURL+"/simple/",
		"BUNPY_CACHE_DIR="+cacheDir,
	)
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		tb.Fatalf("bunpy pm lock (snap): %v\n%s", err, out.String())
	}
	data, err := os.ReadFile(filepath.Join(runDir, "uv.lock"))
	if err != nil {
		tb.Fatalf("read uv.lock: %v", err)
	}
	return data
}

// snapPinnedPackages runs bunpy pm lock against the snapshot server and
// returns the full pinned set as "name==version" strings for pip install.
// The root project entry (version 0.0.1) is excluded.
func snapPinnedPackages(b testing.TB, manifest []byte) []string {
	b.Helper()
	lock := snapLockBunpy(b, manifest)
	versions := parseLockVersions(lock)
	pkgs := make([]string, 0, len(versions))
	for name, ver := range versions {
		if ver == "0.0.1" {
			continue // root project stub
		}
		pkgs = append(pkgs, name+"=="+ver)
	}
	return pkgs
}

// manifestToRequirements extracts the dependencies list from a pyproject.toml
// and returns a requirements.in-format byte slice for pip-compile.
func manifestToRequirements(manifest []byte) []byte {
	var lines []string
	inDeps := false
	for _, rawLine := range bytes.Split(manifest, []byte("\n")) {
		line := string(bytes.TrimSpace(rawLine))
		if line == "dependencies = [" {
			inDeps = true
			continue
		}
		if inDeps {
			if line == "]" {
				break
			}
			dep := strings.Trim(line, `",`)
			if dep != "" {
				lines = append(lines, dep)
			}
		}
	}
	return []byte(strings.Join(lines, "\n") + "\n")
}
