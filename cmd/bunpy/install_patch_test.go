package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// runPatchOpenAndCommit drives a patch lifecycle inside proj that
// flips widget/__init__.py's `VERSION` to `RUNTIME_VERSION`, then
// returns the path to the persisted patch file.
func runPatchOpenAndCommit(t *testing.T, proj, cacheDir string) string {
	t.Helper()
	var stdout, stderr bytes.Buffer
	if code, err := run([]string{"patch", "widget", "--cache-dir", cacheDir}, &stdout, &stderr); err != nil || code != 0 {
		t.Fatalf("open: code=%d err=%v stderr=%s", code, err, stderr.String())
	}
	scratchInit := filepath.Join(proj, ".bunpy", "patches", ".scratch", "widget-1.0.0", "widget", "__init__.py")
	body, err := os.ReadFile(scratchInit)
	if err != nil {
		t.Fatalf("read scratch: %v", err)
	}
	patched := strings.Replace(string(body), "VERSION", "RUNTIME_VERSION", 1)
	if patched == string(body) {
		patched = string(body) + "PATCHED = True\n"
	}
	if err := os.WriteFile(scratchInit, []byte(patched), 0o644); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	if code, err := run([]string{"patch", "--commit", "widget"}, &stdout, &stderr); err != nil || code != 0 {
		t.Fatalf("commit: code=%d err=%v stderr=%s", code, err, stderr.String())
	}
	return filepath.Join(proj, "patches", "widget+1.0.0.patch")
}

func TestInstallAppliesRegisteredPatch(t *testing.T) {
	proj, cacheDir := setupPatchProject(t)
	chdirTo(t, proj)
	_ = runPatchOpenAndCommit(t, proj, cacheDir)

	target := filepath.Join(proj, "site-packages-test")
	var stdout, stderr bytes.Buffer
	code, err := run([]string{"install", "--target", target, "--cache-dir", cacheDir, "--no-verify"}, &stdout, &stderr)
	if err != nil || code != 0 {
		t.Fatalf("install: code=%d err=%v stderr=%s", code, err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "patched widget 1.0.0") {
		t.Errorf("expected `patched widget 1.0.0` line, got %q", stdout.String())
	}
	body, err := os.ReadFile(filepath.Join(target, "widget", "__init__.py"))
	if err != nil {
		t.Fatalf("read installed: %v", err)
	}
	if strings.Contains(string(body), "VERSION = ") && !strings.Contains(string(body), "RUNTIME_VERSION") && !strings.Contains(string(body), "PATCHED") {
		t.Errorf("patch did not land: %q", body)
	}
	installer, err := os.ReadFile(filepath.Join(target, "widget-1.0.0.dist-info", "INSTALLER"))
	if err != nil {
		t.Fatalf("read INSTALLER: %v", err)
	}
	if strings.TrimSpace(string(installer)) != "bunpy-patch" {
		t.Errorf("INSTALLER = %q, want bunpy-patch", installer)
	}
}

func TestInstallNoPatchesSkipsApply(t *testing.T) {
	proj, cacheDir := setupPatchProject(t)
	chdirTo(t, proj)
	_ = runPatchOpenAndCommit(t, proj, cacheDir)

	target := filepath.Join(proj, "site-packages-test")
	var stdout, stderr bytes.Buffer
	code, err := run([]string{"install", "--target", target, "--cache-dir", cacheDir, "--no-verify", "--no-patches"}, &stdout, &stderr)
	if err != nil || code != 0 {
		t.Fatalf("install --no-patches: code=%d err=%v stderr=%s", code, err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "installed widget 1.0.0") {
		t.Errorf("expected `installed widget 1.0.0` line, got %q", stdout.String())
	}
	if strings.Contains(stdout.String(), "patched widget") {
		t.Errorf("did not expect patched line, got %q", stdout.String())
	}
}

func TestInstallStalePatchFails(t *testing.T) {
	proj, cacheDir := setupPatchProject(t)
	chdirTo(t, proj)

	bad := filepath.Join(proj, "patches", "widget+1.0.0.patch")
	if err := os.MkdirAll(filepath.Dir(bad), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bad, []byte(`--- a/widget/__init__.py
+++ b/widget/__init__.py
@@ -1,1 +1,1 @@
-NEVER_EXISTS
+REPLACEMENT
`), 0o644); err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(proj, "pyproject.toml")
	body, _ := os.ReadFile(manifestPath)
	body = append(body, []byte("\n[tool.bunpy.patches]\n\"widget@1.0.0\" = \"patches/widget+1.0.0.patch\"\n")...)
	if err := os.WriteFile(manifestPath, body, 0o644); err != nil {
		t.Fatal(err)
	}

	target := filepath.Join(proj, "site-packages-test")
	var stdout, stderr bytes.Buffer
	code, err := run([]string{"install", "--target", target, "--cache-dir", cacheDir, "--no-verify"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error: stale patch should fail apply")
	}
	if code == 0 {
		t.Error("expected non-zero exit")
	}
}
