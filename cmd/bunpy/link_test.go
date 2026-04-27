package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setupLinkSource seeds a source package directory with
// pyproject.toml + a Python package, and returns its absolute,
// symlink-resolved path. The test's cwd is left unchanged. The
// resolve step matters on macOS where `/var/folders/...`
// symlinks to `/private/var/folders/...` and `os.Getwd` after
// `os.Chdir` returns the resolved form.
func setupLinkSource(t *testing.T, name, version string) string {
	t.Helper()
	src := t.TempDir()
	manifest := []byte("[project]\nname = \"" + name + "\"\nversion = \"" + version + "\"\n")
	if err := os.WriteFile(filepath.Join(src, "pyproject.toml"), manifest, 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	pkg := filepath.Join(src, name)
	if err := os.MkdirAll(pkg, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkg, "__init__.py"), []byte("VERSION = '"+version+"'\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	resolved, err := filepath.EvalSymlinks(src)
	if err != nil {
		t.Fatal(err)
	}
	return resolved
}

// chdirTo cds into dir for the duration of the test.
func chdirTo(t *testing.T, dir string) {
	t.Helper()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
}

func TestLinkRegistersCurrentProject(t *testing.T) {
	registry := t.TempDir()
	t.Setenv("BUNPY_LINK_DIR", registry)
	src := setupLinkSource(t, "widget", "1.0.0")
	chdirTo(t, src)

	var stdout, stderr bytes.Buffer
	code, err := run([]string{"link"}, &stdout, &stderr)
	if err != nil || code != 0 {
		t.Fatalf("bunpy link: code=%d err=%v stderr=%s", code, err, stderr.String())
	}
	body, err := os.ReadFile(filepath.Join(registry, "widget.json"))
	if err != nil {
		t.Fatalf("read entry: %v", err)
	}
	var entry struct {
		Name    string `json:"name"`
		Version string `json:"version"`
		Source  string `json:"source"`
	}
	if err := json.Unmarshal(body, &entry); err != nil {
		t.Fatalf("parse entry: %v", err)
	}
	if entry.Name != "widget" || entry.Version != "1.0.0" || entry.Source != src {
		t.Errorf("entry = %+v, want widget/1.0.0 src=%s", entry, src)
	}
}

func TestLinkUnknownPackageErrors(t *testing.T) {
	t.Setenv("BUNPY_LINK_DIR", t.TempDir())
	consumer := t.TempDir()
	if err := os.WriteFile(filepath.Join(consumer, "pyproject.toml"), []byte("[project]\nname = \"demo\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	chdirTo(t, consumer)

	var stdout, stderr bytes.Buffer
	code, err := run([]string{"link", "widget"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error: unknown link")
	}
	if code == 0 {
		t.Error("expected non-zero exit")
	}
}

func TestLinkInstallsEditableProxy(t *testing.T) {
	t.Setenv("BUNPY_LINK_DIR", t.TempDir())
	src := setupLinkSource(t, "widget", "1.0.0")
	consumerRaw := t.TempDir()
	consumer, err := filepath.EvalSymlinks(consumerRaw)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(consumer, "pyproject.toml"), []byte("[project]\nname = \"demo\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Register from the source dir.
	chdirTo(t, src)
	{
		var stdout, stderr bytes.Buffer
		if code, err := run([]string{"link"}, &stdout, &stderr); err != nil || code != 0 {
			t.Fatalf("register: code=%d err=%v stderr=%s", code, err, stderr.String())
		}
	}
	// Install from the consumer.
	chdirTo(t, consumer)
	{
		var stdout, stderr bytes.Buffer
		code, err := run([]string{"link", "widget"}, &stdout, &stderr)
		if err != nil || code != 0 {
			t.Fatalf("link widget: code=%d err=%v stderr=%s", code, err, stderr.String())
		}
		if !strings.Contains(stdout.String(), "linked widget 1.0.0") {
			t.Errorf("expected linked summary, got: %q", stdout.String())
		}
	}

	pth := filepath.Join(consumer, ".bunpy", "site-packages", "widget.pth")
	body, err := os.ReadFile(pth)
	if err != nil {
		t.Fatalf("read .pth: %v", err)
	}
	if strings.TrimSpace(string(body)) != src {
		t.Errorf(".pth = %q, want %q", string(body), src)
	}
	installer, err := os.ReadFile(filepath.Join(consumer, ".bunpy", "site-packages", "widget-1.0.0.dist-info", "INSTALLER"))
	if err != nil {
		t.Fatalf("read INSTALLER: %v", err)
	}
	if strings.TrimSpace(string(installer)) != "bunpy-link" {
		t.Errorf("INSTALLER = %q, want bunpy-link", string(installer))
	}
}

func TestLinkListPrintsRegistry(t *testing.T) {
	t.Setenv("BUNPY_LINK_DIR", t.TempDir())
	src := setupLinkSource(t, "widget", "1.0.0")
	chdirTo(t, src)

	var stdout, stderr bytes.Buffer
	if code, err := run([]string{"link"}, &stdout, &stderr); err != nil || code != 0 {
		t.Fatalf("register: code=%d err=%v stderr=%s", code, err, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code, err := run([]string{"link", "--list"}, &stdout, &stderr)
	if err != nil || code != 0 {
		t.Fatalf("--list: code=%d err=%v stderr=%s", code, err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "widget") || !strings.Contains(stdout.String(), src) {
		t.Errorf("--list output = %q, want widget + %s", stdout.String(), src)
	}
}

func TestLinkRequiresProjectName(t *testing.T) {
	t.Setenv("BUNPY_LINK_DIR", t.TempDir())
	src := t.TempDir()
	// pyproject.toml with no [project] table
	if err := os.WriteFile(filepath.Join(src, "pyproject.toml"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	chdirTo(t, src)

	var stdout, stderr bytes.Buffer
	code, err := run([]string{"link"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error: missing project")
	}
	if code == 0 {
		t.Error("expected non-zero exit")
	}
}
