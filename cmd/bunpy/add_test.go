package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupAddFixture(t *testing.T, manifest string) (cwd string) {
	t.Helper()
	tmp := t.TempDir()
	cache := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmp, "pyproject.toml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	old, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	abs, err := filepath.Abs(filepath.Join(old, "..", "..", "tests", "fixtures", "v013", "index"))
	if err != nil {
		t.Fatalf("abs fixtures: %v", err)
	}
	t.Setenv("BUNPY_PYPI_FIXTURES", abs)
	t.Setenv("BUNPY_CACHE_DIR", cache)
	return tmp
}

func TestAddFromFixtureURL(t *testing.T) {
	tmp := setupAddFixture(t, `[project]
name = "demo"
version = "0.0.1"
`)
	var stdout, stderr bytes.Buffer
	code, err := run([]string{"add", "widget"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("bunpy add widget: %v\nstderr: %s", err, stderr.String())
	}
	if code != 0 {
		t.Fatalf("bunpy add widget: exit %d\nstderr: %s", code, stderr.String())
	}
	body, err := os.ReadFile(filepath.Join(tmp, "pyproject.toml"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	if !strings.Contains(string(body), "widget>=1.1.0") {
		t.Errorf("manifest missing widget>=1.1.0:\n%s", body)
	}
	initPy := filepath.Join(tmp, ".bunpy", "site-packages", "widget", "__init__.py")
	if data, err := os.ReadFile(initPy); err != nil {
		t.Errorf("widget/__init__.py not installed: %v", err)
	} else if !strings.Contains(string(data), "widget 1.1.0") {
		t.Errorf("installed wrong version: %s", data)
	}
	if !strings.Contains(stdout.String(), "added widget 1.1.0") {
		t.Errorf("stdout missing summary: %q", stdout.String())
	}
}

func TestAddSpecFilter(t *testing.T) {
	tmp := setupAddFixture(t, `[project]
name = "demo"
version = "0.0.1"
`)
	var stdout, stderr bytes.Buffer
	code, err := run([]string{"add", "widget==1.0.0"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("bunpy add widget==1.0.0: %v\nstderr: %s", err, stderr.String())
	}
	if code != 0 {
		t.Fatalf("exit %d\nstderr: %s", code, stderr.String())
	}
	body, _ := os.ReadFile(filepath.Join(tmp, "pyproject.toml"))
	if !strings.Contains(string(body), "widget==1.0.0") {
		t.Errorf("manifest missing widget==1.0.0:\n%s", body)
	}
	initPy, _ := os.ReadFile(filepath.Join(tmp, ".bunpy", "site-packages", "widget", "__init__.py"))
	if !strings.Contains(string(initPy), "widget 1.0.0") {
		t.Errorf("installed wrong version: %s", initPy)
	}
}

func TestAddNoInstall(t *testing.T) {
	tmp := setupAddFixture(t, `[project]
name = "demo"
version = "0.0.1"
`)
	var stdout, stderr bytes.Buffer
	code, err := run([]string{"add", "widget", "--no-install"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("bunpy add --no-install: %v\nstderr: %s", err, stderr.String())
	}
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	body, _ := os.ReadFile(filepath.Join(tmp, "pyproject.toml"))
	if !strings.Contains(string(body), "widget>=1.1.0") {
		t.Errorf("manifest missing widget>=1.1.0:\n%s", body)
	}
	if _, err := os.Stat(filepath.Join(tmp, ".bunpy", "site-packages")); !os.IsNotExist(err) {
		t.Errorf(".bunpy/site-packages must not exist with --no-install: err=%v", err)
	}
}

func TestAddNoWrite(t *testing.T) {
	tmp := setupAddFixture(t, `[project]
name = "demo"
version = "0.0.1"
`)
	original, _ := os.ReadFile(filepath.Join(tmp, "pyproject.toml"))
	var stdout, stderr bytes.Buffer
	code, err := run([]string{"add", "widget", "--no-write"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("bunpy add --no-write: %v\nstderr: %s", err, stderr.String())
	}
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	body, _ := os.ReadFile(filepath.Join(tmp, "pyproject.toml"))
	if string(body) != string(original) {
		t.Errorf("manifest must be untouched with --no-write\n--- got:\n%s\n--- want:\n%s", body, original)
	}
	if _, err := os.Stat(filepath.Join(tmp, ".bunpy", "site-packages", "widget", "__init__.py")); err != nil {
		t.Errorf("install missing: %v", err)
	}
}

func TestAddNoMatching(t *testing.T) {
	setupAddFixture(t, `[project]
name = "demo"
version = "0.0.1"
`)
	var stdout, stderr bytes.Buffer
	code, _ := run([]string{"add", "widget>=99"}, &stdout, &stderr)
	if code == 0 {
		t.Errorf("expected non-zero exit on no match; stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}

func TestAddNoArg(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code, err := run([]string{"add"}, &stdout, &stderr)
	if err == nil {
		t.Error("expected error for missing arg")
	}
	if code == 0 {
		t.Error("expected non-zero exit for missing arg")
	}
}

func TestAddHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code, err := run([]string{"add", "--help"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("bunpy add --help: %v", err)
	}
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(stdout.String(), "bunpy add:") {
		t.Errorf("help output missing header: %q", stdout.String())
	}
}
