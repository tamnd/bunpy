package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setupInstallFixture seeds a project dir with the given pyproject.toml,
// runs `bunpy pm lock` against the v015 fixture (widget+gizmo) so a
// lockfile exists, then chdirs into it and returns the path. Lane
// flags on the pm-lock invocation are read from the manifest, so the
// fixture chooses which lockfile lanes get tagged.
func setupInstallFixture(t *testing.T, manifest string) string {
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
	var stdout, stderr bytes.Buffer
	code, err := run([]string{"pm", "lock"}, &stdout, &stderr)
	if err != nil || code != 0 {
		t.Fatalf("pm lock seed: code=%d err=%v stderr=%s", code, err, stderr.String())
	}
	return tmp
}

func TestInstallDefaultMainOnly(t *testing.T) {
	tmp := setupInstallFixture(t, `[project]
name = "demo"
version = "0.0.1"
dependencies = ["widget>=1.0"]

[dependency-groups]
dev = ["widget==1.0.0"]
`)
	// Manifest pulls widget into both main and dev. Pin gets both lanes;
	// default install (main only) still installs because widget is in main.
	var stdout, stderr bytes.Buffer
	code, err := run([]string{"install"}, &stdout, &stderr)
	if err != nil || code != 0 {
		t.Fatalf("bunpy install: code=%d err=%v stderr=%s", code, err, stderr.String())
	}
	if _, err := os.Stat(filepath.Join(tmp, ".bunpy", "site-packages", "widget", "__init__.py")); err != nil {
		t.Errorf("widget should be installed (main lane): %v", err)
	}
}

func TestInstallSkipsDevByDefault(t *testing.T) {
	tmp := setupInstallFixture(t, `[project]
name = "demo"
version = "0.0.1"

[dependency-groups]
dev = ["widget>=1.0"]
`)
	var stdout, stderr bytes.Buffer
	code, err := run([]string{"install"}, &stdout, &stderr)
	if err != nil || code != 0 {
		t.Fatalf("bunpy install: code=%d err=%v stderr=%s", code, err, stderr.String())
	}
	if _, err := os.Stat(filepath.Join(tmp, ".bunpy", "site-packages", "widget", "__init__.py")); !os.IsNotExist(err) {
		t.Errorf("widget must not be installed when only in dev lane (default install): err=%v", err)
	}
	if !strings.Contains(stdout.String(), "skipped") {
		t.Errorf("expected skipped message, got: %q", stdout.String())
	}
}

func TestInstallDevFlagInstallsDevPins(t *testing.T) {
	tmp := setupInstallFixture(t, `[project]
name = "demo"
version = "0.0.1"

[dependency-groups]
dev = ["widget>=1.0"]
`)
	var stdout, stderr bytes.Buffer
	code, err := run([]string{"install", "-D"}, &stdout, &stderr)
	if err != nil || code != 0 {
		t.Fatalf("bunpy install -D: code=%d err=%v stderr=%s", code, err, stderr.String())
	}
	if _, err := os.Stat(filepath.Join(tmp, ".bunpy", "site-packages", "widget", "__init__.py")); err != nil {
		t.Errorf("widget should install with -D: %v", err)
	}
}

func TestInstallOptionalFlagInstallsGroup(t *testing.T) {
	tmp := setupInstallFixture(t, `[project]
name = "demo"
version = "0.0.1"

[project.optional-dependencies]
web = ["widget>=1.0"]
`)
	var stdout, stderr bytes.Buffer
	code, err := run([]string{"install", "-O", "web"}, &stdout, &stderr)
	if err != nil || code != 0 {
		t.Fatalf("bunpy install -O web: code=%d err=%v stderr=%s", code, err, stderr.String())
	}
	if _, err := os.Stat(filepath.Join(tmp, ".bunpy", "site-packages", "widget", "__init__.py")); err != nil {
		t.Errorf("widget should install with -O web: %v", err)
	}
}

func TestInstallOptionalFlagSkipsOtherGroup(t *testing.T) {
	tmp := setupInstallFixture(t, `[project]
name = "demo"
version = "0.0.1"

[project.optional-dependencies]
web = ["widget>=1.0"]
`)
	var stdout, stderr bytes.Buffer
	code, err := run([]string{"install", "-O", "cli"}, &stdout, &stderr)
	if err != nil || code != 0 {
		t.Fatalf("bunpy install -O cli: code=%d err=%v stderr=%s", code, err, stderr.String())
	}
	if _, err := os.Stat(filepath.Join(tmp, ".bunpy", "site-packages", "widget", "__init__.py")); !os.IsNotExist(err) {
		t.Errorf("widget must not install when only the wrong optional group is selected: err=%v", err)
	}
}

func TestInstallAllExtras(t *testing.T) {
	tmp := setupInstallFixture(t, `[project]
name = "demo"
version = "0.0.1"

[project.optional-dependencies]
web = ["widget>=1.0"]
`)
	var stdout, stderr bytes.Buffer
	code, err := run([]string{"install", "--all-extras"}, &stdout, &stderr)
	if err != nil || code != 0 {
		t.Fatalf("bunpy install --all-extras: code=%d err=%v stderr=%s", code, err, stderr.String())
	}
	if _, err := os.Stat(filepath.Join(tmp, ".bunpy", "site-packages", "widget", "__init__.py")); err != nil {
		t.Errorf("widget should install with --all-extras: %v", err)
	}
}

func TestInstallProductionRejectsDev(t *testing.T) {
	setupInstallFixture(t, `[project]
name = "demo"
version = "0.0.1"
dependencies = ["widget>=1.0"]
`)
	var stdout, stderr bytes.Buffer
	code, _ := run([]string{"install", "--production", "-D"}, &stdout, &stderr)
	if code == 0 {
		t.Errorf("expected non-zero exit for --production with -D; stderr=%q", stderr.String())
	}
}
