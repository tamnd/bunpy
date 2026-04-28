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

func TestAddWritesLockfile(t *testing.T) {
	tmp := setupAddFixture(t, `[project]
name = "demo"
version = "0.0.1"
`)
	var stdout, stderr bytes.Buffer
	code, err := run([]string{"add", "widget"}, &stdout, &stderr)
	if err != nil || code != 0 {
		t.Fatalf("bunpy add widget: code=%d err=%v stderr=%s", code, err, stderr.String())
	}
	body, err := os.ReadFile(filepath.Join(tmp, "uv.lock"))
	if err != nil {
		t.Fatalf("read lockfile: %v", err)
	}
	got := string(body)
	for _, want := range []string{
		"version = 1",
		"[[package]]",
		"name = \"widget\"",
		"version = \"1.1.0\"",
		"widget-1.1.0-py3-none-any.whl",
		"hash = \"sha256:5b9866d1a5e11d85e37f88de9a941f9349ed18f4cd46508b12b1603d2ad63e2b\"",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("lockfile missing %q\n%s", want, got)
		}
	}
}

func TestAddNoWriteSkipsLockfile(t *testing.T) {
	tmp := setupAddFixture(t, `[project]
name = "demo"
version = "0.0.1"
`)
	var stdout, stderr bytes.Buffer
	code, err := run([]string{"add", "widget", "--no-write"}, &stdout, &stderr)
	if err != nil || code != 0 {
		t.Fatalf("bunpy add --no-write: code=%d err=%v", code, err)
	}
	if _, err := os.Stat(filepath.Join(tmp, "uv.lock")); !os.IsNotExist(err) {
		t.Errorf("uv.lock must not exist with --no-write: err=%v", err)
	}
}

func TestAddRefreshesLockfile(t *testing.T) {
	tmp := setupAddFixture(t, `[project]
name = "demo"
version = "0.0.1"
`)
	var stdout, stderr bytes.Buffer
	if code, err := run([]string{"add", "widget==1.0.0"}, &stdout, &stderr); err != nil || code != 0 {
		t.Fatalf("first add: code=%d err=%v stderr=%s", code, err, stderr.String())
	}
	first, err := os.ReadFile(filepath.Join(tmp, "uv.lock"))
	if err != nil {
		t.Fatalf("read lock: %v", err)
	}
	if !strings.Contains(string(first), "version = \"1.0.0\"") {
		t.Fatalf("first lockfile not pinned to 1.0.0:\n%s", first)
	}
	stdout.Reset()
	stderr.Reset()
	if code, err := run([]string{"add", "widget==1.1.0"}, &stdout, &stderr); err != nil || code != 0 {
		t.Fatalf("second add: code=%d err=%v stderr=%s", code, err, stderr.String())
	}
	second, err := os.ReadFile(filepath.Join(tmp, "uv.lock"))
	if err != nil {
		t.Fatalf("read lock 2: %v", err)
	}
	got := string(second)
	if !strings.Contains(got, "version = \"1.1.0\"") {
		t.Errorf("lockfile not upgraded:\n%s", got)
	}
	if strings.Contains(got, "version = \"1.0.0\"") {
		t.Errorf("lockfile still has old 1.0.0 entry:\n%s", got)
	}
	if strings.Count(got, "[[package]]") != 1 {
		t.Errorf("expected single package entry:\n%s", got)
	}
}

func TestAddDevWritesDependencyGroup(t *testing.T) {
	tmp := setupAddFixture(t, `[project]
name = "demo"
version = "0.0.1"
`)
	var stdout, stderr bytes.Buffer
	code, err := run([]string{"add", "widget", "-D"}, &stdout, &stderr)
	if err != nil || code != 0 {
		t.Fatalf("bunpy add widget -D: code=%d err=%v stderr=%s", code, err, stderr.String())
	}
	body, _ := os.ReadFile(filepath.Join(tmp, "pyproject.toml"))
	got := string(body)
	if !strings.Contains(got, "[dependency-groups]") {
		t.Errorf("manifest missing [dependency-groups]:\n%s", got)
	}
	if !strings.Contains(got, `dev = [`) || !strings.Contains(got, `"widget>=1.1.0"`) {
		t.Errorf("manifest missing dev dep:\n%s", got)
	}
	lock, _ := os.ReadFile(filepath.Join(tmp, "uv.lock"))
	if !strings.Contains(string(lock), `groups = ["dev"]`) {
		t.Errorf("lockfile missing dev group tag:\n%s", lock)
	}
}

func TestAddDevWithGroupName(t *testing.T) {
	tmp := setupAddFixture(t, `[project]
name = "demo"
version = "0.0.1"
`)
	var stdout, stderr bytes.Buffer
	code, err := run([]string{"add", "widget", "-D", "--group", "test"}, &stdout, &stderr)
	if err != nil || code != 0 {
		t.Fatalf("bunpy add widget -D --group test: code=%d err=%v stderr=%s", code, err, stderr.String())
	}
	body, _ := os.ReadFile(filepath.Join(tmp, "pyproject.toml"))
	if !strings.Contains(string(body), `test = [`) {
		t.Errorf("manifest missing test group:\n%s", body)
	}
	lock, _ := os.ReadFile(filepath.Join(tmp, "uv.lock"))
	if !strings.Contains(string(lock), `groups = ["group:test"]`) {
		t.Errorf("lockfile missing group:test group tag:\n%s", lock)
	}
}

func TestAddOptionalWritesProjectOptionalDeps(t *testing.T) {
	tmp := setupAddFixture(t, `[project]
name = "demo"
version = "0.0.1"
`)
	var stdout, stderr bytes.Buffer
	code, err := run([]string{"add", "widget", "-O", "web"}, &stdout, &stderr)
	if err != nil || code != 0 {
		t.Fatalf("bunpy add widget -O web: code=%d err=%v stderr=%s", code, err, stderr.String())
	}
	body, _ := os.ReadFile(filepath.Join(tmp, "pyproject.toml"))
	got := string(body)
	if !strings.Contains(got, "[project.optional-dependencies]") {
		t.Errorf("manifest missing optional-dependencies table:\n%s", got)
	}
	if !strings.Contains(got, `web = [`) {
		t.Errorf("manifest missing web group:\n%s", got)
	}
	lock, _ := os.ReadFile(filepath.Join(tmp, "uv.lock"))
	if !strings.Contains(string(lock), `groups = ["optional:web"]`) {
		t.Errorf("lockfile missing optional:web group tag:\n%s", lock)
	}
}

func TestAddPeerWritesToolBunpy(t *testing.T) {
	tmp := setupAddFixture(t, `[project]
name = "demo"
version = "0.0.1"
`)
	var stdout, stderr bytes.Buffer
	code, err := run([]string{"add", "widget", "-P"}, &stdout, &stderr)
	if err != nil || code != 0 {
		t.Fatalf("bunpy add widget -P: code=%d err=%v stderr=%s", code, err, stderr.String())
	}
	body, _ := os.ReadFile(filepath.Join(tmp, "pyproject.toml"))
	got := string(body)
	if !strings.Contains(got, "[tool.bunpy]") {
		t.Errorf("manifest missing tool.bunpy:\n%s", got)
	}
	if !strings.Contains(got, `peer-dependencies = [`) {
		t.Errorf("manifest missing peer-dependencies:\n%s", got)
	}
	lock, _ := os.ReadFile(filepath.Join(tmp, "uv.lock"))
	if !strings.Contains(string(lock), `groups = ["peer"]`) {
		t.Errorf("lockfile missing peer group tag:\n%s", lock)
	}
}

func TestAddRejectsConflictingLaneFlags(t *testing.T) {
	setupAddFixture(t, `[project]
name = "demo"
version = "0.0.1"
`)
	var stdout, stderr bytes.Buffer
	code, _ := run([]string{"add", "widget", "-D", "-P"}, &stdout, &stderr)
	if code == 0 {
		t.Errorf("expected non-zero exit for -D -P together; stderr=%q", stderr.String())
	}
}

func TestAddRejectsGroupWithoutDev(t *testing.T) {
	setupAddFixture(t, `[project]
name = "demo"
version = "0.0.1"
`)
	var stdout, stderr bytes.Buffer
	code, _ := run([]string{"add", "widget", "--group", "test"}, &stdout, &stderr)
	if code == 0 {
		t.Errorf("expected non-zero exit for --group without -D; stderr=%q", stderr.String())
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
