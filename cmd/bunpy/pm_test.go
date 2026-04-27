package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFixture(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "pyproject.toml")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestPmConfigJSON(t *testing.T) {
	path := writeFixture(t, `[project]
name = "demo"
version = "0.1.0"
dependencies = ["click>=8"]

[tool.bunpy]
profile = "fast"
`)
	var stdout, stderr bytes.Buffer
	code, err := run([]string{"pm", "config", path}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("bunpy pm config: %v\nstderr:\n%s", err, stderr.String())
	}
	if code != 0 {
		t.Fatalf("code %d, want 0", code)
	}
	var got map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON %q: %v", stdout.String(), err)
	}
	proj, ok := got["project"].(map[string]any)
	if !ok {
		t.Fatalf("project: not a map, got %T", got["project"])
	}
	if proj["name"] != "demo" {
		t.Errorf("project.name = %v, want demo", proj["name"])
	}
	if proj["version"] != "0.1.0" {
		t.Errorf("project.version = %v, want 0.1.0", proj["version"])
	}
	tool, ok := got["tool"].(map[string]any)
	if !ok {
		t.Fatalf("tool: not a map, got %T", got["tool"])
	}
	raw, ok := tool["raw"].(map[string]any)
	if !ok {
		t.Fatalf("tool.raw: not a map, got %T", tool["raw"])
	}
	if raw["profile"] != "fast" {
		t.Errorf("tool.bunpy.profile = %v, want fast", raw["profile"])
	}
}

func TestPmConfigDefaultPath(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "pyproject.toml"),
		[]byte("[project]\nname = \"d\"\nversion = \"0.0.1\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cwd, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	var stdout, stderr bytes.Buffer
	code, err := run([]string{"pm", "config"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("bunpy pm config: %v\nstderr:\n%s", err, stderr.String())
	}
	if code != 0 {
		t.Fatalf("code %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), `"name": "d"`) {
		t.Errorf("stdout missing name=d: %s", stdout.String())
	}
}

func TestPmConfigMissingFile(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code, err := run([]string{"pm", "config", "/no/such/path/pyproject.toml"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	if code == 0 {
		t.Error("expected non-zero exit code")
	}
}

func TestPmConfigBadName(t *testing.T) {
	path := writeFixture(t, "[project]\nname = \"with space\"\n")
	var stdout, stderr bytes.Buffer
	code, err := run([]string{"pm", "config", path}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for invalid name")
	}
	if code == 0 {
		t.Error("expected non-zero exit code")
	}
}

func TestPmConfigUnknownFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code, err := run([]string{"pm", "config", "--frobnicate"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for unknown flag")
	}
	if code == 0 {
		t.Error("expected non-zero exit code")
	}
}

func TestPmNoVerb(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code, err := run([]string{"pm"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for missing verb")
	}
	if code == 0 {
		t.Error("expected non-zero exit code")
	}
}

func TestPmUnknownVerb(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code, err := run([]string{"pm", "frobnicate"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for unknown verb")
	}
	if code == 0 {
		t.Error("expected non-zero exit code")
	}
}

func TestPmHelp(t *testing.T) {
	for _, args := range [][]string{{"pm", "--help"}, {"pm", "-h"}, {"pm", "help"}} {
		var stdout, stderr bytes.Buffer
		code, err := run(args, &stdout, &stderr)
		if err != nil {
			t.Fatalf("bunpy %v: %v", args, err)
		}
		if code != 0 {
			t.Fatalf("%v: code %d, want 0", args, code)
		}
		if !strings.Contains(stdout.String(), "bunpy pm") {
			t.Errorf("%v: stdout missing `bunpy pm`: %q", args, stdout.String())
		}
	}
}

func writeFixturePypi(t *testing.T, name, body string) string {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "pypi.org", "simple", name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "index.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestPmInfoFromFixture(t *testing.T) {
	root := writeFixturePypi(t, "demo", `{"name":"demo","files":[{"filename":"demo-1.0-py3-none-any.whl","url":"https://x/demo-1.0-py3-none-any.whl","hashes":{"sha256":"abc"}}],"meta":{"api-version":"1.1"}}`)
	t.Setenv("BUNPY_PYPI_FIXTURES", root)
	t.Setenv("BUNPY_CACHE_DIR", t.TempDir())
	var stdout, stderr bytes.Buffer
	code, err := run([]string{"pm", "info", "demo"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("bunpy pm info: %v\nstderr:\n%s", err, stderr.String())
	}
	if code != 0 {
		t.Fatalf("code %d, want 0", code)
	}
	var got map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout.String())
	}
	if got["name"] != "demo" {
		t.Errorf("name: got %v", got["name"])
	}
	versions, ok := got["versions"].([]any)
	if !ok || len(versions) != 1 || versions[0] != "1.0" {
		t.Errorf("versions: got %v", got["versions"])
	}
}

func TestPmInfoMissing404(t *testing.T) {
	t.Setenv("BUNPY_PYPI_FIXTURES", t.TempDir())
	t.Setenv("BUNPY_CACHE_DIR", t.TempDir())
	var stdout, stderr bytes.Buffer
	code, err := run([]string{"pm", "info", "nopkg-xyzzy"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("want error on 404")
	}
	if code == 0 {
		t.Error("want non-zero exit")
	}
}

func TestPmInfoNoArg(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code, err := run([]string{"pm", "info"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("want error on missing package arg")
	}
	if code == 0 {
		t.Error("want non-zero exit")
	}
}

func TestPmInfoUnknownFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code, err := run([]string{"pm", "info", "--frobnicate", "demo"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("want error on unknown flag")
	}
	if code == 0 {
		t.Error("want non-zero exit")
	}
}

func TestPmInfoIndexFlag(t *testing.T) {
	root := writeFixturePypi(t, "thing", `{"name":"thing","files":[],"meta":{"api-version":"1.1"}}`)
	t.Setenv("BUNPY_PYPI_FIXTURES", root)
	t.Setenv("BUNPY_CACHE_DIR", t.TempDir())
	var stdout, stderr bytes.Buffer
	code, err := run([]string{"pm", "info", "thing", "--index", "https://pypi.org/simple/"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("bunpy pm info --index: %v\nstderr:\n%s", err, stderr.String())
	}
	if code != 0 {
		t.Fatalf("code %d, want 0", code)
	}
}

func TestPmInfoCacheRoundTrip(t *testing.T) {
	root := writeFixturePypi(t, "rt", `{"name":"rt","files":[],"meta":{"api-version":"1.1"}}`)
	cacheDir := t.TempDir()
	t.Setenv("BUNPY_PYPI_FIXTURES", root)
	for i := 0; i < 2; i++ {
		var stdout, stderr bytes.Buffer
		code, err := run([]string{"pm", "info", "rt", "--cache-dir", cacheDir}, &stdout, &stderr)
		if err != nil {
			t.Fatalf("pass %d: %v\nstderr:\n%s", i, err, stderr.String())
		}
		if code != 0 {
			t.Fatalf("pass %d: code %d", i, code)
		}
	}
}

func TestPmInstallWheelLocalPath(t *testing.T) {
	whl, err := filepath.Abs(filepath.Join("..", "..", "tests", "fixtures", "v012", "tinypkg-0.1.0-py3-none-any.whl"))
	if err != nil {
		t.Fatal(err)
	}
	target := t.TempDir()
	var stdout, stderr bytes.Buffer
	code, err := run([]string{"pm", "install-wheel", whl, "--target", target}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("install-wheel: %v\nstderr:\n%s", err, stderr.String())
	}
	if code != 0 {
		t.Fatalf("code %d, want 0", code)
	}
	if _, err := os.Stat(filepath.Join(target, "tinypkg", "__init__.py")); err != nil {
		t.Fatalf("tinypkg/__init__.py not installed: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(target, "tinypkg-0.1.0.dist-info", "INSTALLER"))
	if err != nil {
		t.Fatalf("INSTALLER missing: %v", err)
	}
	if string(got) != "bunpy\n" {
		t.Errorf("INSTALLER = %q, want %q", got, "bunpy\n")
	}
}

func TestPmInstallWheelFromFixturesURL(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", "tests", "fixtures", "v012", "index"))
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("BUNPY_PYPI_FIXTURES", root)
	t.Setenv("BUNPY_CACHE_DIR", t.TempDir())
	target := t.TempDir()
	var stdout, stderr bytes.Buffer
	url := "https://files.example/tinypkg/tinypkg-0.1.0-py3-none-any.whl"
	code, err := run([]string{"pm", "install-wheel", url, "--target", target}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("install-wheel from URL: %v\nstderr:\n%s", err, stderr.String())
	}
	if code != 0 {
		t.Fatalf("code %d, want 0", code)
	}
	if _, err := os.Stat(filepath.Join(target, "tinypkg", "__init__.py")); err != nil {
		t.Fatalf("tinypkg/__init__.py not installed: %v", err)
	}
}

func TestPmInstallWheelMissingFile(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code, err := run([]string{"pm", "install-wheel", "/no/such/wheel.whl"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("want error on missing file")
	}
	if code == 0 {
		t.Error("want non-zero exit")
	}
}

func TestPmInstallWheelNoArg(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code, err := run([]string{"pm", "install-wheel"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("want error on missing source")
	}
	if code == 0 {
		t.Error("want non-zero exit")
	}
}

func TestPmInstallWheelHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code, err := run([]string{"pm", "install-wheel", "--help"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("--help: %v", err)
	}
	if code != 0 {
		t.Fatalf("code %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), "bunpy pm install-wheel") {
		t.Errorf("stdout missing header: %q", stdout.String())
	}
}

func TestPmConfigHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code, err := run([]string{"pm", "config", "--help"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("bunpy pm config --help: %v", err)
	}
	if code != 0 {
		t.Fatalf("code %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), "bunpy pm config") {
		t.Errorf("stdout missing `bunpy pm config`: %q", stdout.String())
	}
}
