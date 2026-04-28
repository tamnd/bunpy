package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setupRemoveFixture seeds a project with the given pyproject.toml and
// runs `pm lock` against the v013 fixture so a fresh lockfile is on
// disk. Returns the temp dir.
func setupRemoveFixture(t *testing.T, manifest string) string {
	return setupOutdatedFixture(t, manifest, "")
}

func TestRemoveBareDropsFromAllLanes(t *testing.T) {
	tmp := setupRemoveFixture(t, `[project]
name = "demo"
version = "0.0.1"
dependencies = ["widget>=1.0"]

[dependency-groups]
dev = ["widget>=1.0"]
`)
	var stdout, stderr bytes.Buffer
	code, err := run([]string{"remove", "widget", "--no-install"}, &stdout, &stderr)
	if err != nil || code != 0 {
		t.Fatalf("bunpy remove: code=%d err=%v stderr=%s", code, err, stderr.String())
	}
	mf, err := os.ReadFile(filepath.Join(tmp, "pyproject.toml"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	if strings.Contains(string(mf), "widget") {
		t.Errorf("widget still in manifest:\n%s", string(mf))
	}
	lock, err := os.ReadFile(filepath.Join(tmp, "uv.lock"))
	if err != nil {
		t.Fatalf("read lock: %v", err)
	}
	if strings.Contains(string(lock), `name = "widget"`) {
		t.Errorf("widget still in lock:\n%s", string(lock))
	}
	if !strings.Contains(stdout.String(), "removed 2 packages") {
		t.Errorf("expected `removed 2 packages` summary, got: %q", stdout.String())
	}
}

func TestRemoveDevOnlyKeepsMainLane(t *testing.T) {
	tmp := setupRemoveFixture(t, `[project]
name = "demo"
version = "0.0.1"
dependencies = ["widget>=1.0"]

[dependency-groups]
dev = ["widget>=1.0"]
`)
	var stdout, stderr bytes.Buffer
	code, err := run([]string{"remove", "-D", "widget", "--no-install"}, &stdout, &stderr)
	if err != nil || code != 0 {
		t.Fatalf("bunpy remove -D: code=%d err=%v stderr=%s", code, err, stderr.String())
	}
	mf, err := os.ReadFile(filepath.Join(tmp, "pyproject.toml"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	got := string(mf)
	if !strings.Contains(got, `dependencies = ["widget>=1.0"]`) {
		t.Errorf("main lane lost:\n%s", got)
	}
	// dev table should still exist but widget should be gone from it.
	if !strings.Contains(got, "[dependency-groups]") {
		t.Errorf("dev table lost:\n%s", got)
	}
	devStart := strings.Index(got, "[dependency-groups]")
	if devStart >= 0 && strings.Contains(got[devStart:], `"widget`) {
		t.Errorf("widget still in dev table:\n%s", got)
	}
}

func TestRemoveMissingPackageIsNoop(t *testing.T) {
	tmp := setupRemoveFixture(t, `[project]
name = "demo"
version = "0.0.1"
dependencies = ["widget>=1.0"]
`)
	before, err := os.ReadFile(filepath.Join(tmp, "pyproject.toml"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	var stdout, stderr bytes.Buffer
	code, err := run([]string{"remove", "notapkg", "--no-install"}, &stdout, &stderr)
	if err != nil || code != 0 {
		t.Fatalf("bunpy remove notapkg: code=%d err=%v stderr=%s", code, err, stderr.String())
	}
	after, err := os.ReadFile(filepath.Join(tmp, "pyproject.toml"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	if !bytes.Equal(before, after) {
		t.Errorf("manifest mutated on missing remove:\n--- before ---\n%s\n--- after ---\n%s", string(before), string(after))
	}
	if !strings.Contains(stdout.String(), "removed 0 packages") {
		t.Errorf("expected `removed 0 packages`, got: %q", stdout.String())
	}
}

func TestRemoveLaneFlagsAreMutuallyExclusive(t *testing.T) {
	setupRemoveFixture(t, `[project]
name = "demo"
version = "0.0.1"
dependencies = ["widget>=1.0"]
`)
	var stdout, stderr bytes.Buffer
	code, err := run([]string{"remove", "-D", "-P", "widget", "--no-install"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error: -D and -P together")
	}
	if code == 0 {
		t.Error("expected non-zero exit")
	}
}

func TestRemoveRequiresPackageName(t *testing.T) {
	setupRemoveFixture(t, `[project]
name = "demo"
version = "0.0.1"
dependencies = ["widget>=1.0"]
`)
	var stdout, stderr bytes.Buffer
	code, err := run([]string{"remove", "--no-install"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error: bare remove with no pkg")
	}
	if code == 0 {
		t.Error("expected non-zero exit")
	}
}

func TestRemoveNoInstallSkipsSitePackages(t *testing.T) {
	tmp := setupRemoveFixture(t, `[project]
name = "demo"
version = "0.0.1"
dependencies = ["widget>=1.0"]
`)
	// Pre-create a sentinel under site-packages to simulate a prior install.
	site := filepath.Join(tmp, ".bunpy", "site-packages", "widget")
	if err := os.MkdirAll(site, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(site, "__init__.py"), []byte(""), 0o644); err != nil {
		t.Fatalf("write sentinel: %v", err)
	}
	var stdout, stderr bytes.Buffer
	code, err := run([]string{"remove", "widget", "--no-install"}, &stdout, &stderr)
	if err != nil || code != 0 {
		t.Fatalf("bunpy remove --no-install: code=%d err=%v stderr=%s", code, err, stderr.String())
	}
	if _, err := os.Stat(filepath.Join(site, "__init__.py")); err != nil {
		t.Errorf("--no-install must not touch site-packages: err=%v", err)
	}
}
