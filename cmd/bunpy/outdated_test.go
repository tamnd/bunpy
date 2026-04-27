package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setupOutdatedFixture seeds a project dir with the given pyproject.toml,
// runs `bunpy pm lock` against the v013 fixture (widget 1.0.0 / 1.1.0)
// to seed bunpy.lock, then optionally rewrites the lock so a specific
// version pin is captured. The v013 index lets us exercise the
// outdated/up-to-date split without authoring a new fixture tree.
func setupOutdatedFixture(t *testing.T, manifest string, lockVersion string) string {
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
	if lockVersion != "" {
		body, err := os.ReadFile(filepath.Join(tmp, "bunpy.lock"))
		if err != nil {
			t.Fatalf("read lock: %v", err)
		}
		// pm lock pins widget 1.1.0 (highest matching). Rewrite to the
		// caller's chosen version with the matching wheel filename and
		// hash from v013/index/pypi.org/simple/widget/index.json.
		out := strings.ReplaceAll(string(body), `version = "1.1.0"`, `version = "`+lockVersion+`"`)
		out = strings.ReplaceAll(out, "widget-1.1.0-py3-none-any.whl", "widget-"+lockVersion+"-py3-none-any.whl")
		if lockVersion == "1.0.0" {
			out = strings.ReplaceAll(out,
				"5b9866d1a5e11d85e37f88de9a941f9349ed18f4cd46508b12b1603d2ad63e2b",
				"86eaa2517187b884e706a792085ed6dabd2530e6f37d0d9220d08b5448c8a796")
		}
		if err := os.WriteFile(filepath.Join(tmp, "bunpy.lock"), []byte(out), 0o644); err != nil {
			t.Fatalf("rewrite lock: %v", err)
		}
	}
	return tmp
}

func TestOutdatedReportsNewerPin(t *testing.T) {
	setupOutdatedFixture(t, `[project]
name = "demo"
version = "0.0.1"
dependencies = ["widget>=1.0"]
`, "1.0.0")
	var stdout, stderr bytes.Buffer
	code, err := run([]string{"outdated"}, &stdout, &stderr)
	if err != nil || code != 0 {
		t.Fatalf("bunpy outdated: code=%d err=%v stderr=%s", code, err, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "widget") {
		t.Errorf("expected widget row, got: %q", out)
	}
	if !strings.Contains(out, "1.0.0") || !strings.Contains(out, "1.1.0") {
		t.Errorf("expected current 1.0.0 and wanted/latest 1.1.0, got: %q", out)
	}
}

func TestOutdatedSilentWhenUpToDate(t *testing.T) {
	setupOutdatedFixture(t, `[project]
name = "demo"
version = "0.0.1"
dependencies = ["widget>=1.0"]
`, "")
	var stdout, stderr bytes.Buffer
	code, err := run([]string{"outdated"}, &stdout, &stderr)
	if err != nil || code != 0 {
		t.Fatalf("bunpy outdated: code=%d err=%v stderr=%s", code, err, stderr.String())
	}
	if got := stdout.String(); got != "" {
		t.Errorf("expected empty stdout when up to date, got: %q", got)
	}
}

func TestOutdatedJSON(t *testing.T) {
	setupOutdatedFixture(t, `[project]
name = "demo"
version = "0.0.1"
dependencies = ["widget>=1.0"]
`, "1.0.0")
	var stdout, stderr bytes.Buffer
	code, err := run([]string{"outdated", "--json"}, &stdout, &stderr)
	if err != nil || code != 0 {
		t.Fatalf("bunpy outdated --json: code=%d err=%v stderr=%s", code, err, stderr.String())
	}
	var got struct {
		Outdated []struct {
			Name    string   `json:"name"`
			Current string   `json:"current"`
			Wanted  string   `json:"wanted"`
			Latest  string   `json:"latest"`
			Lanes   []string `json:"lanes"`
		} `json:"outdated"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout.String())), &got); err != nil {
		t.Fatalf("invalid JSON %q: %v", stdout.String(), err)
	}
	if len(got.Outdated) != 1 {
		t.Fatalf("rows = %d, want 1: %+v", len(got.Outdated), got.Outdated)
	}
	row := got.Outdated[0]
	if row.Name != "widget" || row.Current != "1.0.0" || row.Wanted != "1.1.0" || row.Latest != "1.1.0" {
		t.Errorf("row = %+v", row)
	}
}

func TestOutdatedLaneFilterExcludesDevByDefault(t *testing.T) {
	// widget lives in [dependency-groups].dev only. Default lane set is
	// main, so outdated must not report it. Passing -D pulls it back.
	setupOutdatedFixture(t, `[project]
name = "demo"
version = "0.0.1"

[dependency-groups]
dev = ["widget>=1.0"]
`, "1.0.0")
	var stdout, stderr bytes.Buffer
	code, err := run([]string{"outdated"}, &stdout, &stderr)
	if err != nil || code != 0 {
		t.Fatalf("bunpy outdated (default): code=%d err=%v stderr=%s", code, err, stderr.String())
	}
	if got := stdout.String(); got != "" {
		t.Errorf("default lane set must skip dev-only pin, got: %q", got)
	}
	stdout.Reset()
	stderr.Reset()
	code, err = run([]string{"outdated", "-D"}, &stdout, &stderr)
	if err != nil || code != 0 {
		t.Fatalf("bunpy outdated -D: code=%d err=%v stderr=%s", code, err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "widget") {
		t.Errorf("-D should surface dev pin, got: %q", stdout.String())
	}
}
