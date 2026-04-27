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
