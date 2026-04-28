package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setupUpdateFixture mirrors setupOutdatedFixture: seeds a project,
// runs pm lock against the v013 fixture, then optionally rewrites the
// lockfile pin so we can prove `bunpy update` actually moves it.
func setupUpdateFixture(t *testing.T, manifest string, lockVersion string) string {
	return setupOutdatedFixture(t, manifest, lockVersion)
}

func TestUpdateUpgradesPinWithinSpec(t *testing.T) {
	tmp := setupUpdateFixture(t, `[project]
name = "demo"
version = "0.0.1"
dependencies = ["widget>=1.0"]
`, "1.0.0")
	var stdout, stderr bytes.Buffer
	code, err := run([]string{"update", "--no-install"}, &stdout, &stderr)
	if err != nil || code != 0 {
		t.Fatalf("bunpy update: code=%d err=%v stderr=%s", code, err, stderr.String())
	}
	body, err := os.ReadFile(filepath.Join(tmp, "uv.lock"))
	if err != nil {
		t.Fatalf("read lock: %v", err)
	}
	if !strings.Contains(string(body), `version = "1.1.0"`) {
		t.Errorf("expected widget pin to move to 1.1.0, got:\n%s", string(body))
	}
	if !strings.Contains(stdout.String(), "widget 1.0.0 -> 1.1.0") {
		t.Errorf("expected upgrade line, got: %q", stdout.String())
	}
}

func TestUpdateNoChanges(t *testing.T) {
	setupUpdateFixture(t, `[project]
name = "demo"
version = "0.0.1"
dependencies = ["widget>=1.0"]
`, "")
	var stdout, stderr bytes.Buffer
	code, err := run([]string{"update", "--no-install"}, &stdout, &stderr)
	if err != nil || code != 0 {
		t.Fatalf("bunpy update: code=%d err=%v stderr=%s", code, err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "no changes") {
		t.Errorf("expected `no changes`, got: %q", stdout.String())
	}
}

func TestUpdateLatestIgnoresSpec(t *testing.T) {
	// Manifest pins widget to ==1.0.0; uv.lock matches. A bare update
	// must respect the spec (no change), but `--latest widget` strips it
	// and lets the resolver pick the highest non-prerelease (1.1.0).
	tmp := setupUpdateFixture(t, `[project]
name = "demo"
version = "0.0.1"
dependencies = ["widget==1.0.0"]
`, "1.0.0")
	var stdout, stderr bytes.Buffer
	code, err := run([]string{"update", "--no-install"}, &stdout, &stderr)
	if err != nil || code != 0 {
		t.Fatalf("bunpy update: code=%d err=%v stderr=%s", code, err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "no changes") {
		t.Errorf("bare update with ==1.0.0 spec should keep pin, got: %q", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	code, err = run([]string{"update", "--latest", "widget", "--no-install"}, &stdout, &stderr)
	if err != nil || code != 0 {
		t.Fatalf("bunpy update --latest: code=%d err=%v stderr=%s", code, err, stderr.String())
	}
	body, err := os.ReadFile(filepath.Join(tmp, "uv.lock"))
	if err != nil {
		t.Fatalf("read lock: %v", err)
	}
	if !strings.Contains(string(body), `version = "1.1.0"`) {
		t.Errorf("expected --latest to move widget to 1.1.0, got:\n%s", string(body))
	}
}

func TestUpdateLatestRequiresPackageName(t *testing.T) {
	setupUpdateFixture(t, `[project]
name = "demo"
version = "0.0.1"
dependencies = ["widget>=1.0"]
`, "1.0.0")
	var stdout, stderr bytes.Buffer
	code, err := run([]string{"update", "--latest", "--no-install"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error: --latest with no positional pkg")
	}
	if code == 0 {
		t.Error("expected non-zero exit code")
	}
}

func TestUpdateNoInstallSkipsSitePackages(t *testing.T) {
	tmp := setupUpdateFixture(t, `[project]
name = "demo"
version = "0.0.1"
dependencies = ["widget>=1.0"]
`, "1.0.0")
	var stdout, stderr bytes.Buffer
	code, err := run([]string{"update", "--no-install"}, &stdout, &stderr)
	if err != nil || code != 0 {
		t.Fatalf("bunpy update --no-install: code=%d err=%v stderr=%s", code, err, stderr.String())
	}
	if _, err := os.Stat(filepath.Join(tmp, ".bunpy", "site-packages", "widget", "__init__.py")); !os.IsNotExist(err) {
		t.Errorf("--no-install must not touch site-packages: err=%v", err)
	}
}
