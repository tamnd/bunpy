package compare_test

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// realworldDir returns the path to benchmarks/fixtures/realworld.
func realworldDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "fixtures", "realworld")
}

// skipIfNoRealworld skips b/t if realworld fixtures have not been generated.
func skipIfNoRealworld(tb testing.TB) {
	tb.Helper()
	if _, err := os.Stat(realworldDir()); err != nil {
		tb.Skip("realworld fixtures not found; run: go run benchmarks/fixtures/build_realworld.go")
	}
	if rwServerURL == "" {
		tb.Skip("realworld fixture server not started (fixtures missing at test startup)")
	}
}

// uvBin returns the path to uv, or "" if not found.
func uvBin() string {
	p, _ := exec.LookPath("uv")
	return p
}

// lockBunpy runs `bunpy pm lock` in a fresh temp dir pointed at the realworld
// fixture server, writes the given pyproject.toml content, and returns the
// generated uv.lock bytes.
func lockBunpy(tb testing.TB, pyproject []byte) []byte {
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
		"BUNPY_PYPI_INDEX_URL="+rwServerURL+"/simple/",
		"BUNPY_CACHE_DIR="+cacheDir,
	)
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		tb.Fatalf("bunpy pm lock: %v\n%s", err, out.String())
	}
	data, err := os.ReadFile(filepath.Join(runDir, "uv.lock"))
	if err != nil {
		tb.Fatalf("read uv.lock: %v", err)
	}
	return data
}

// lockUV runs `uv lock` in a fresh temp dir pointed at the realworld
// fixture server and returns the generated uv.lock bytes.
// The caller must check that uv is available before calling.
func lockUV(tb testing.TB, pyproject []byte) []byte {
	tb.Helper()
	runDir := tb.TempDir()
	// Inject the realworld index URL into [tool.uv] section.
	uvToml := append(pyproject, []byte(fmt.Sprintf("\n[tool.uv]\nindex-url = %q\n", rwServerURL+"/simple/"))...)
	if err := os.WriteFile(filepath.Join(runDir, "pyproject.toml"), uvToml, 0o644); err != nil {
		tb.Fatal(err)
	}
	var out bytes.Buffer
	cmd := exec.Command(uvBin(), "lock", "--no-cache", "--no-progress")
	cmd.Dir = runDir
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		tb.Logf("uv output:\n%s", out.String())
		tb.Fatalf("uv lock: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(runDir, "uv.lock"))
	if err != nil {
		tb.Fatalf("read uv.lock: %v", err)
	}
	return data
}

// parseLockVersions extracts {name: version} from a uv.lock file.
// It scans for [[package]] ... name = "..." ... version = "..." blocks.
// The root project entry (no version line immediately after name) is excluded.
func parseLockVersions(lockData []byte) map[string]string {
	versions := map[string]string{}
	scanner := bufio.NewScanner(bytes.NewReader(lockData))
	var curName string
	inPkg := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "[[package]]" {
			inPkg = true
			curName = ""
			continue
		}
		if !inPkg {
			continue
		}
		if strings.HasPrefix(line, "name = ") {
			curName = strings.Trim(strings.TrimPrefix(line, "name = "), `"`)
			continue
		}
		if strings.HasPrefix(line, "version = ") && curName != "" {
			ver := strings.Trim(strings.TrimPrefix(line, "version = "), `"`)
			versions[curName] = ver
			inPkg = false
			curName = ""
		}
	}
	return versions
}

// readProfile reads a realworld pyproject.toml by profile name.
func readProfile(tb testing.TB, profile string) []byte {
	tb.Helper()
	path := filepath.Join(realworldDir(), profile, "pyproject.toml")
	data, err := os.ReadFile(path)
	if err != nil {
		tb.Fatalf("read profile %s: %v", profile, err)
	}
	return data
}

// --- Benchmarks ---

func BenchmarkLockBunpy_FastAPI(b *testing.B) {
	skipIfNoRealworld(b)
	manifest := readProfile(b, "fastapi-app")
	cacheBase := b.TempDir()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		runDir, _ := os.MkdirTemp("", "rw-bunpy-fastapi-*")
		iterCache, _ := os.MkdirTemp(cacheBase, "iter-*")
		_ = os.WriteFile(filepath.Join(runDir, "pyproject.toml"), manifest, 0o644)
		b.StartTimer()

		var out bytes.Buffer
		cmd := exec.Command(bunpyBin, "pm", "lock")
		cmd.Dir = runDir
		cmd.Env = append(os.Environ(),
			"BUNPY_PYPI_INDEX_URL="+rwServerURL+"/simple/",
			"BUNPY_CACHE_DIR="+iterCache,
		)
		cmd.Stdout, cmd.Stderr = &out, &out
		runErr := cmd.Run()

		b.StopTimer()
		os.RemoveAll(runDir)
		os.RemoveAll(iterCache)
		if runErr != nil {
			b.Fatalf("bunpy pm lock (fastapi): %v\n%s", runErr, out.String())
		}
		b.StartTimer()
	}
}

func BenchmarkLockUV_FastAPI(b *testing.B) {
	skipIfNoRealworld(b)
	if uvBin() == "" {
		b.Skip("uv not found in PATH; skipping")
	}
	manifest := readProfile(b, "fastapi-app")
	uvToml := append(manifest, []byte(fmt.Sprintf("\n[tool.uv]\nindex-url = %q\n", rwServerURL+"/simple/"))...)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		runDir, _ := os.MkdirTemp("", "rw-uv-fastapi-*")
		_ = os.WriteFile(filepath.Join(runDir, "pyproject.toml"), uvToml, 0o644)
		b.StartTimer()

		var out bytes.Buffer
		cmd := exec.Command(uvBin(), "lock", "--no-cache", "--no-progress")
		cmd.Dir = runDir
		cmd.Stdout, cmd.Stderr = &out, &out
		runErr := cmd.Run()

		b.StopTimer()
		os.RemoveAll(runDir)
		if runErr != nil {
			b.Logf("uv output: %s", out.String())
			b.Fatalf("uv lock (fastapi): %v", runErr)
		}
		b.StartTimer()
	}
}

func BenchmarkLockBunpy_Django(b *testing.B) {
	skipIfNoRealworld(b)
	manifest := readProfile(b, "django-app")
	cacheBase := b.TempDir()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		runDir, _ := os.MkdirTemp("", "rw-bunpy-django-*")
		iterCache, _ := os.MkdirTemp(cacheBase, "iter-*")
		_ = os.WriteFile(filepath.Join(runDir, "pyproject.toml"), manifest, 0o644)
		b.StartTimer()

		var out bytes.Buffer
		cmd := exec.Command(bunpyBin, "pm", "lock")
		cmd.Dir = runDir
		cmd.Env = append(os.Environ(),
			"BUNPY_PYPI_INDEX_URL="+rwServerURL+"/simple/",
			"BUNPY_CACHE_DIR="+iterCache,
		)
		cmd.Stdout, cmd.Stderr = &out, &out
		runErr := cmd.Run()

		b.StopTimer()
		os.RemoveAll(runDir)
		os.RemoveAll(iterCache)
		if runErr != nil {
			b.Fatalf("bunpy pm lock (django): %v\n%s", runErr, out.String())
		}
		b.StartTimer()
	}
}

func BenchmarkLockUV_Django(b *testing.B) {
	skipIfNoRealworld(b)
	if uvBin() == "" {
		b.Skip("uv not found in PATH; skipping")
	}
	manifest := readProfile(b, "django-app")
	uvToml := append(manifest, []byte(fmt.Sprintf("\n[tool.uv]\nindex-url = %q\n", rwServerURL+"/simple/"))...)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		runDir, _ := os.MkdirTemp("", "rw-uv-django-*")
		_ = os.WriteFile(filepath.Join(runDir, "pyproject.toml"), uvToml, 0o644)
		b.StartTimer()

		var out bytes.Buffer
		cmd := exec.Command(uvBin(), "lock", "--no-cache", "--no-progress")
		cmd.Dir = runDir
		cmd.Stdout, cmd.Stderr = &out, &out
		runErr := cmd.Run()

		b.StopTimer()
		os.RemoveAll(runDir)
		if runErr != nil {
			b.Logf("uv output: %s", out.String())
			b.Fatalf("uv lock (django): %v", runErr)
		}
		b.StartTimer()
	}
}

func BenchmarkLockBunpy_DataScience(b *testing.B) {
	skipIfNoRealworld(b)
	manifest := readProfile(b, "datascience")
	cacheBase := b.TempDir()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		runDir, _ := os.MkdirTemp("", "rw-bunpy-ds-*")
		iterCache, _ := os.MkdirTemp(cacheBase, "iter-*")
		_ = os.WriteFile(filepath.Join(runDir, "pyproject.toml"), manifest, 0o644)
		b.StartTimer()

		var out bytes.Buffer
		cmd := exec.Command(bunpyBin, "pm", "lock")
		cmd.Dir = runDir
		cmd.Env = append(os.Environ(),
			"BUNPY_PYPI_INDEX_URL="+rwServerURL+"/simple/",
			"BUNPY_CACHE_DIR="+iterCache,
		)
		cmd.Stdout, cmd.Stderr = &out, &out
		runErr := cmd.Run()

		b.StopTimer()
		os.RemoveAll(runDir)
		os.RemoveAll(iterCache)
		if runErr != nil {
			b.Fatalf("bunpy pm lock (datascience): %v\n%s", runErr, out.String())
		}
		b.StartTimer()
	}
}

func BenchmarkLockUV_DataScience(b *testing.B) {
	skipIfNoRealworld(b)
	if uvBin() == "" {
		b.Skip("uv not found in PATH; skipping")
	}
	manifest := readProfile(b, "datascience")
	uvToml := append(manifest, []byte(fmt.Sprintf("\n[tool.uv]\nindex-url = %q\n", rwServerURL+"/simple/"))...)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		runDir, _ := os.MkdirTemp("", "rw-uv-ds-*")
		_ = os.WriteFile(filepath.Join(runDir, "pyproject.toml"), uvToml, 0o644)
		b.StartTimer()

		var out bytes.Buffer
		cmd := exec.Command(uvBin(), "lock", "--no-cache", "--no-progress")
		cmd.Dir = runDir
		cmd.Stdout, cmd.Stderr = &out, &out
		runErr := cmd.Run()

		b.StopTimer()
		os.RemoveAll(runDir)
		if runErr != nil {
			b.Logf("uv output: %s", out.String())
			b.Fatalf("uv lock (datascience): %v", runErr)
		}
		b.StartTimer()
	}
}

func BenchmarkLockBunpy_CLI(b *testing.B) {
	skipIfNoRealworld(b)
	manifest := readProfile(b, "cli-tool")
	cacheBase := b.TempDir()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		runDir, _ := os.MkdirTemp("", "rw-bunpy-cli-*")
		iterCache, _ := os.MkdirTemp(cacheBase, "iter-*")
		_ = os.WriteFile(filepath.Join(runDir, "pyproject.toml"), manifest, 0o644)
		b.StartTimer()

		var out bytes.Buffer
		cmd := exec.Command(bunpyBin, "pm", "lock")
		cmd.Dir = runDir
		cmd.Env = append(os.Environ(),
			"BUNPY_PYPI_INDEX_URL="+rwServerURL+"/simple/",
			"BUNPY_CACHE_DIR="+iterCache,
		)
		cmd.Stdout, cmd.Stderr = &out, &out
		runErr := cmd.Run()

		b.StopTimer()
		os.RemoveAll(runDir)
		os.RemoveAll(iterCache)
		if runErr != nil {
			b.Fatalf("bunpy pm lock (cli): %v\n%s", runErr, out.String())
		}
		b.StartTimer()
	}
}

func BenchmarkLockUV_CLI(b *testing.B) {
	skipIfNoRealworld(b)
	if uvBin() == "" {
		b.Skip("uv not found in PATH; skipping")
	}
	manifest := readProfile(b, "cli-tool")
	uvToml := append(manifest, []byte(fmt.Sprintf("\n[tool.uv]\nindex-url = %q\n", rwServerURL+"/simple/"))...)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		runDir, _ := os.MkdirTemp("", "rw-uv-cli-*")
		_ = os.WriteFile(filepath.Join(runDir, "pyproject.toml"), uvToml, 0o644)
		b.StartTimer()

		var out bytes.Buffer
		cmd := exec.Command(uvBin(), "lock", "--no-cache", "--no-progress")
		cmd.Dir = runDir
		cmd.Stdout, cmd.Stderr = &out, &out
		runErr := cmd.Run()

		b.StopTimer()
		os.RemoveAll(runDir)
		if runErr != nil {
			b.Logf("uv output: %s", out.String())
			b.Fatalf("uv lock (cli): %v", runErr)
		}
		b.StartTimer()
	}
}

// --- Compatibility tests ---

func testCompatibility(t *testing.T, profile string) {
	t.Helper()
	skipIfNoRealworld(t)
	if uvBin() == "" {
		t.Skip("uv not found in PATH; skipping compatibility test")
	}

	manifest := readProfile(t, profile)
	bunpyLock := lockBunpy(t, manifest)
	uvLock := lockUV(t, manifest)

	bunpyVersions := parseLockVersions(bunpyLock)
	uvVersions := parseLockVersions(uvLock)

	// Remove the root project from both (it won't have a version in uv.lock).
	delete(bunpyVersions, profile)
	delete(uvVersions, profile)

	if len(bunpyVersions) == 0 {
		t.Fatalf("bunpy lock produced no packages for %s", profile)
	}
	if len(uvVersions) == 0 {
		t.Fatalf("uv lock produced no packages for %s", profile)
	}

	var diffs []string

	// Check packages bunpy has that uv doesn't, or disagrees on.
	for name, bVer := range bunpyVersions {
		uVer, ok := uvVersions[name]
		if !ok {
			diffs = append(diffs, fmt.Sprintf("  bunpy has %s==%s, uv omitted it", name, bVer))
		} else if bVer != uVer {
			diffs = append(diffs, fmt.Sprintf("  %s: bunpy==%s  uv==%s", name, bVer, uVer))
		}
	}
	// Check packages uv has that bunpy doesn't.
	for name, uVer := range uvVersions {
		if _, ok := bunpyVersions[name]; !ok {
			diffs = append(diffs, fmt.Sprintf("  uv has %s==%s, bunpy omitted it", name, uVer))
		}
	}

	if len(diffs) > 0 {
		t.Errorf("compatibility divergence for %s (%d package(s)):\n%s",
			profile, len(diffs), strings.Join(diffs, "\n"))
	} else {
		t.Logf("compatible: %d packages agree between bunpy and uv", len(bunpyVersions))
	}
}

func TestCompatibility_FastAPI(t *testing.T)     { testCompatibility(t, "fastapi-app") }
func TestCompatibility_Django(t *testing.T)      { testCompatibility(t, "django-app") }
func TestCompatibility_DataScience(t *testing.T) { testCompatibility(t, "datascience") }
func TestCompatibility_CLI(t *testing.T)         { testCompatibility(t, "cli-tool") }
