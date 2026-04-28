package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setupPatchProject seeds a project with a pyproject.toml, a
// uv.lock entry for widget@1.0.0, and a pre-populated wheel
// cache so `bunpy patch widget` does not hit the network.
func setupPatchProject(t *testing.T) (proj, cacheDir string) {
	t.Helper()
	proj = t.TempDir()
	cacheDir = t.TempDir()

	mustWrite(t, filepath.Join(proj, "pyproject.toml"), []byte(`[project]
name = "demo"
version = "0.1.0"
dependencies = [
    "widget>=1.0",
]
`))

	mustWrite(t, filepath.Join(proj, "uv.lock"), []byte(`version = 1
requires-python = ">=3.12"

[[package]]
name = "widget"
version = "1.0.0"
source = { registry = "https://pypi.org/simple" }

[[package.wheels]]
url = "https://files.example.com/widget-1.0.0-py3-none-any.whl"
hash = "sha256:abcd"
size = 0
`))

	wheelSrc, err := os.ReadFile("../../tests/fixtures/v013/widget-1.0.0-py3-none-any.whl")
	if err != nil {
		t.Skipf("widget fixture missing: %v", err)
	}
	wheelDst := filepath.Join(cacheDir, "wheels", "widget", "widget-1.0.0-py3-none-any.whl")
	if err := os.MkdirAll(filepath.Dir(wheelDst), 0o755); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, wheelDst, wheelSrc)
	return proj, cacheDir
}

func mustWrite(t *testing.T, path string, body []byte) {
	t.Helper()
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func TestPatchOpenCreatesScratch(t *testing.T) {
	proj, cacheDir := setupPatchProject(t)
	chdirTo(t, proj)

	var stdout, stderr bytes.Buffer
	code, err := run([]string{"patch", "widget", "--cache-dir", cacheDir}, &stdout, &stderr)
	if err != nil || code != 0 {
		t.Fatalf("patch widget: code=%d err=%v stderr=%s", code, err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "scratch widget 1.0.0") {
		t.Errorf("expected scratch line, got %q", stdout.String())
	}
	scratch := filepath.Join(proj, ".bunpy", "patches", ".scratch", "widget-1.0.0")
	if _, err := os.Stat(filepath.Join(scratch, "widget", "__init__.py")); err != nil {
		t.Errorf("scratch missing widget/__init__.py: %v", err)
	}
}

func TestPatchOpenIsIdempotent(t *testing.T) {
	proj, cacheDir := setupPatchProject(t)
	chdirTo(t, proj)

	var stdout, stderr bytes.Buffer
	if code, err := run([]string{"patch", "widget", "--cache-dir", cacheDir}, &stdout, &stderr); err != nil || code != 0 {
		t.Fatalf("first patch: code=%d err=%v", code, err)
	}
	scratchInit := filepath.Join(proj, ".bunpy", "patches", ".scratch", "widget-1.0.0", "widget", "__init__.py")
	if err := os.WriteFile(scratchInit, []byte("DIRTY = 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	if code, err := run([]string{"patch", "widget", "--cache-dir", cacheDir}, &stdout, &stderr); err != nil || code != 0 {
		t.Fatalf("rerun patch: code=%d err=%v", code, err)
	}
	body, err := os.ReadFile(scratchInit)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), "DIRTY") {
		t.Errorf("scratch not refreshed: %s", body)
	}
}

func TestPatchCommitWritesPatchAndManifest(t *testing.T) {
	proj, cacheDir := setupPatchProject(t)
	chdirTo(t, proj)

	var stdout, stderr bytes.Buffer
	if code, err := run([]string{"patch", "widget", "--cache-dir", cacheDir}, &stdout, &stderr); err != nil || code != 0 {
		t.Fatalf("open: code=%d err=%v stderr=%s", code, err, stderr.String())
	}
	scratchInit := filepath.Join(proj, ".bunpy", "patches", ".scratch", "widget-1.0.0", "widget", "__init__.py")
	pristineInit := filepath.Join(proj, ".bunpy", "patches", ".pristine", "widget-1.0.0", "widget", "__init__.py")
	pristineBody, err := os.ReadFile(pristineInit)
	if err != nil {
		t.Fatalf("read pristine: %v", err)
	}
	patched := strings.Replace(string(pristineBody), "VERSION", "RUNTIME_VERSION", 1)
	if patched == string(pristineBody) {
		patched = string(pristineBody) + "PATCHED = True\n"
	}
	if err := os.WriteFile(scratchInit, []byte(patched), 0o644); err != nil {
		t.Fatal(err)
	}

	stdout.Reset()
	stderr.Reset()
	code, err := run([]string{"patch", "--commit", "widget"}, &stdout, &stderr)
	if err != nil || code != 0 {
		t.Fatalf("commit: code=%d err=%v stderr=%s", code, err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "patched widget 1.0.0") {
		t.Errorf("expected commit summary, got %q", stdout.String())
	}
	patchPath := filepath.Join(proj, "patches", "widget+1.0.0.patch")
	patchBody, err := os.ReadFile(patchPath)
	if err != nil {
		t.Fatalf("read patch: %v", err)
	}
	if !strings.Contains(string(patchBody), "--- a/widget/__init__.py") {
		t.Errorf("missing left header in patch: %q", patchBody)
	}
	manifestBody, err := os.ReadFile(filepath.Join(proj, "pyproject.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(manifestBody), "[tool.bunpy.patches]") {
		t.Errorf("manifest missing patches table: %q", manifestBody)
	}
	if !strings.Contains(string(manifestBody), `"widget@1.0.0"`) {
		t.Errorf("manifest missing widget entry: %q", manifestBody)
	}
	if _, err := os.Stat(filepath.Join(proj, ".bunpy", "patches", ".scratch", "widget-1.0.0")); !os.IsNotExist(err) {
		t.Errorf("scratch not cleaned up: %v", err)
	}
}

func TestPatchCommitNoChangesIsNoop(t *testing.T) {
	proj, cacheDir := setupPatchProject(t)
	chdirTo(t, proj)

	var stdout, stderr bytes.Buffer
	if code, err := run([]string{"patch", "widget", "--cache-dir", cacheDir}, &stdout, &stderr); err != nil || code != 0 {
		t.Fatalf("open: code=%d err=%v", code, err)
	}
	stdout.Reset()
	stderr.Reset()
	code, err := run([]string{"patch", "--commit", "widget"}, &stdout, &stderr)
	if err != nil || code != 0 {
		t.Fatalf("commit: code=%d err=%v stderr=%s", code, err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "no changes for widget") {
		t.Errorf("expected no-changes line, got %q", stdout.String())
	}
	if _, err := os.Stat(filepath.Join(proj, "patches", "widget+1.0.0.patch")); !os.IsNotExist(err) {
		t.Errorf("patch file should not exist: %v", err)
	}
}

func TestPatchListPrintsRegisteredPatches(t *testing.T) {
	proj := t.TempDir()
	chdirTo(t, proj)
	mustWrite(t, filepath.Join(proj, "pyproject.toml"), []byte(`[project]
name = "demo"

[tool.bunpy.patches]
"flask@2.3.0" = "patches/flask+2.3.0.patch"
`))

	var stdout, stderr bytes.Buffer
	code, err := run([]string{"patch", "--list"}, &stdout, &stderr)
	if err != nil || code != 0 {
		t.Fatalf("list: code=%d err=%v stderr=%s", code, err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "flask 2.3.0") {
		t.Errorf("expected flask 2.3.0 row, got %q", stdout.String())
	}
}

func TestPatchRequiresPackageName(t *testing.T) {
	proj := t.TempDir()
	chdirTo(t, proj)
	mustWrite(t, filepath.Join(proj, "pyproject.toml"), []byte(`[project]
name = "demo"
`))
	mustWrite(t, filepath.Join(proj, "uv.lock"), []byte(`version = 1
requires-python = ">=3.12"
`))

	var stdout, stderr bytes.Buffer
	code, err := run([]string{"patch"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error: missing package name")
	}
	if code == 0 {
		t.Error("expected non-zero exit")
	}
}
