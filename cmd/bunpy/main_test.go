package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/tamnd/bunpy/v1/runtime"
)

func TestVersion(t *testing.T) {
	for _, arg := range []string{"version", "-v", "--version"} {
		var stdout, stderr bytes.Buffer
		code, err := run([]string{arg}, &stdout, &stderr)
		if err != nil {
			t.Fatalf("%s: %v", arg, err)
		}
		if code != 0 {
			t.Fatalf("%s: code %d, want 0", arg, code)
		}
		want := "bunpy " + runtime.Build().Version
		if !strings.HasPrefix(stdout.String(), want) {
			t.Errorf("%s: stdout %q does not start with %q", arg, stdout.String(), want)
		}
	}
}

func TestVersionShort(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code, err := run([]string{"version", "--short"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("bunpy version --short: %v", err)
	}
	if code != 0 {
		t.Fatalf("code %d, want 0", code)
	}
	got := strings.TrimSpace(stdout.String())
	if got != runtime.Build().Version {
		t.Errorf("stdout %q, want %q", got, runtime.Build().Version)
	}
}

func TestVersionJSON(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code, err := run([]string{"version", "--json"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("bunpy version --json: %v", err)
	}
	if code != 0 {
		t.Fatalf("code %d, want 0", code)
	}
	var got map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout.String())), &got); err != nil {
		t.Fatalf("invalid JSON %q: %v", stdout.String(), err)
	}
	if got["version"] != runtime.Build().Version {
		t.Errorf("json version = %v, want %q", got["version"], runtime.Build().Version)
	}
	for _, k := range []string{"go", "os", "arch"} {
		if _, ok := got[k]; !ok {
			t.Errorf("json output missing key %q: %s", k, stdout.String())
		}
	}
}

func TestVersionDevBuild(t *testing.T) {
	if runtime.Build().Version != "dev" {
		t.Skipf("not a dev build (version = %q); skipping", runtime.Build().Version)
	}
	var stdout, stderr bytes.Buffer
	code, err := run([]string{"version"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("bunpy version: %v", err)
	}
	if code != 0 {
		t.Fatalf("code %d, want 0", code)
	}
	out := stdout.String()
	if !strings.Contains(out, "bunpy dev") {
		t.Errorf("dev build stdout missing `bunpy dev`: %q", out)
	}
	if strings.Contains(out, "commit ") || strings.Contains(out, "built ") {
		t.Errorf("dev build stdout should not include commit/built lines: %q", out)
	}
	if strings.Contains(out, "toolchain:") {
		t.Errorf("dev build stdout should not include toolchain line: %q", out)
	}
}

func TestVersionUnknownFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code, err := run([]string{"version", "--frobnicate"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for unknown version flag")
	}
	if code == 0 {
		t.Error("expected non-zero exit code")
	}
}

func TestHelp(t *testing.T) {
	for _, arg := range []string{"help", "-h", "--help"} {
		var stdout, stderr bytes.Buffer
		code, err := run([]string{arg}, &stdout, &stderr)
		if err != nil {
			t.Fatalf("%s: %v", arg, err)
		}
		if code != 0 {
			t.Fatalf("%s: code %d, want 0", arg, code)
		}
		if !strings.Contains(stdout.String(), "USAGE") {
			t.Errorf("%s: stdout missing USAGE section: %q", arg, stdout.String())
		}
	}
}

func TestNoArgsPrintsUsage(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code, err := run(nil, &stdout, &stderr)
	if err != nil {
		t.Fatalf("no args: %v", err)
	}
	if code != 0 {
		t.Fatalf("no args: code %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), "USAGE") {
		t.Errorf("no args: stdout missing USAGE section: %q", stdout.String())
	}
}

func TestUnknownCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code, err := run([]string{"frobnicate"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected an error for unknown command")
	}
	if code == 0 {
		t.Errorf("expected non-zero exit code, got 0")
	}
	if !strings.Contains(err.Error(), "frobnicate") {
		t.Errorf("error %q does not mention the bad command", err)
	}
}

func TestRunFile(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"empty.py", ""},
		{"pass.py", "pass\n"},
		{"docstring.py", `"""hello from bunpy"""` + "\n"},
		{"assign.py", "x = 1\n"},
	}
	dir := t.TempDir()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(dir, tc.name)
			if err := os.WriteFile(path, []byte(tc.body), 0o644); err != nil {
				t.Fatal(err)
			}
			var stdout, stderr bytes.Buffer
			code, err := run([]string{path}, &stdout, &stderr)
			if err != nil {
				t.Fatalf("run %s: %v\nstderr:\n%s", tc.name, err, stderr.String())
			}
			if code != 0 {
				t.Fatalf("run %s: code %d, want 0\nstderr:\n%s", tc.name, code, stderr.String())
			}
			if got := stdout.String(); got != "" {
				t.Errorf("run %s: stdout %q, want empty", tc.name, got)
			}
		})
	}
}

func TestRunFileMissing(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code, err := run([]string{"/no/such/path/missing.py"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	if code == 0 {
		t.Error("expected non-zero exit code")
	}
}

func TestRunSubcommand(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "pass.py")
	if err := os.WriteFile(path, []byte("pass\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code, err := run([]string{"run", path}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("bunpy run %s: %v\nstderr:\n%s", path, err, stderr.String())
	}
	if code != 0 {
		t.Fatalf("code %d, want 0", code)
	}
	if stdout.String() != "" {
		t.Errorf("stdout %q, want empty", stdout.String())
	}
}

func TestRunSubcommandNoArgs(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code, err := run([]string{"run"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for `bunpy run` with no args")
	}
	if code == 0 {
		t.Error("expected non-zero exit code")
	}
	if !strings.Contains(stderr.String(), "usage") {
		t.Errorf("stderr %q does not mention usage", stderr.String())
	}
}

func TestRunSubcommandHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code, err := run([]string{"run", "--help"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("bunpy run --help: %v", err)
	}
	if code != 0 {
		t.Fatalf("code %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), "bunpy run") {
		t.Errorf("stdout %q missing `bunpy run`", stdout.String())
	}
}

func TestRunSubcommandStdinReserved(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code, err := run([]string{"run", "-"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for stdin script")
	}
	if code == 0 {
		t.Error("expected non-zero exit code")
	}
}

func TestRunSubcommandRejectsNonPyArg(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code, err := run([]string{"run", "frobnicate"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for non-.py argument")
	}
	if code == 0 {
		t.Error("expected non-zero exit code")
	}
}

func TestStdlibSubcommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code, err := run([]string{"stdlib"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("bunpy stdlib: %v\nstderr:\n%s", err, stderr.String())
	}
	if code != 0 {
		t.Fatalf("code %d, want 0", code)
	}
	lines := strings.Split(strings.TrimRight(stdout.String(), "\n"), "\n")
	if len(lines) != runtime.StdlibCount() {
		t.Errorf("stdlib output has %d lines, want %d", len(lines), runtime.StdlibCount())
	}
	found := false
	for _, l := range lines {
		if l == "math" {
			found = true
			break
		}
	}
	if !found {
		t.Error("`bunpy stdlib` output missing `math`")
	}
}

func TestStdlibCount(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code, err := run([]string{"stdlib", "count"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("bunpy stdlib count: %v", err)
	}
	if code != 0 {
		t.Fatalf("code %d, want 0", code)
	}
	got := strings.TrimSpace(stdout.String())
	want := strconv.Itoa(runtime.StdlibCount())
	if got != want {
		t.Errorf("stdlib count = %q, want %q", got, want)
	}
}

func TestStdlibHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code, err := run([]string{"stdlib", "--help"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("bunpy stdlib --help: %v", err)
	}
	if code != 0 {
		t.Fatalf("code %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), "stdlib") {
		t.Errorf("help output missing `stdlib`: %q", stdout.String())
	}
}

func TestStdlibUnknownMode(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code, err := run([]string{"stdlib", "frobnicate"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for unknown stdlib mode")
	}
	if code == 0 {
		t.Error("expected non-zero exit code")
	}
}

func TestRunFileBadSource(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.py")
	// gocopy v0.0.12 rejects function definitions. This test pins the
	// "compile error names the file" contract regardless of which
	// constructs gocopy supports next.
	if err := os.WriteFile(path, []byte("def f():\n    pass\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code, err := run([]string{path}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected compile error for unsupported source")
	}
	if code == 0 {
		t.Error("expected non-zero exit code")
	}
	if !strings.Contains(err.Error(), "bad.py") {
		t.Errorf("error %q does not name the file", err)
	}
}

func TestHelpForEachWiredCommand(t *testing.T) {
	for _, name := range helpTopics() {
		t.Run(name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code, err := run([]string{"help", name}, &stdout, &stderr)
			if err != nil {
				t.Fatalf("bunpy help %s: %v", name, err)
			}
			if code != 0 {
				t.Fatalf("code %d, want 0", code)
			}
			if stdout.Len() == 0 {
				t.Errorf("bunpy help %s produced empty stdout", name)
			}
		})
	}
}

func TestHelpUnknownCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code, err := run([]string{"help", "frobnicate"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for unknown help topic")
	}
	if code == 0 {
		t.Error("expected non-zero exit code")
	}
	if !strings.Contains(stderr.String(), "frobnicate") {
		t.Errorf("stderr %q does not mention the bad topic", stderr.String())
	}
}

func TestHelpFlagAliasesParity(t *testing.T) {
	cases := []struct {
		name      string
		viaSubcmd []string
	}{
		{"run", []string{"run", "--help"}},
		{"run", []string{"run", "-h"}},
		{"stdlib", []string{"stdlib", "--help"}},
		{"stdlib", []string{"stdlib", "-h"}},
		{"version", []string{"version", "--help"}},
		{"version", []string{"version", "-h"}},
		{"man", []string{"man", "--help"}},
		{"man", []string{"man", "-h"}},
		{"pm", []string{"pm", "--help"}},
		{"pm", []string{"pm", "-h"}},
		{"pm-info", []string{"pm", "info", "--help"}},
		{"pm-info", []string{"pm", "info", "-h"}},
		{"pm-install-wheel", []string{"pm", "install-wheel", "--help"}},
		{"pm-install-wheel", []string{"pm", "install-wheel", "-h"}},
	}
	for _, tc := range cases {
		t.Run(strings.Join(tc.viaSubcmd, " "), func(t *testing.T) {
			var aOut, aErr, bOut, bErr bytes.Buffer
			if _, err := run([]string{"help", tc.name}, &aOut, &aErr); err != nil {
				t.Fatalf("bunpy help %s: %v", tc.name, err)
			}
			if _, err := run(tc.viaSubcmd, &bOut, &bErr); err != nil {
				t.Fatalf("bunpy %v: %v", tc.viaSubcmd, err)
			}
			if aOut.String() != bOut.String() {
				t.Errorf("help parity mismatch for %s vs %v\n--- help:\n%s--- subcmd:\n%s",
					tc.name, tc.viaSubcmd, aOut.String(), bOut.String())
			}
		})
	}
}

func TestManRender(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code, err := run([]string{"man", "run"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("bunpy man run: %v", err)
	}
	if code != 0 {
		t.Fatalf("code %d, want 0", code)
	}
	if !strings.HasPrefix(stdout.String(), ".TH BUNPY-RUN 1") {
		t.Errorf("stdout did not start with .TH BUNPY-RUN 1: %q", stdout.String()[:min(stdout.Len(), 64)])
	}
}

func TestManUnknown(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code, err := run([]string{"man", "frobnicate"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for unknown manpage")
	}
	if code == 0 {
		t.Error("expected non-zero exit code")
	}
}

func TestManInstall(t *testing.T) {
	dir := t.TempDir()
	var stdout, stderr bytes.Buffer
	code, err := run([]string{"man", "--install", dir}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("bunpy man --install: %v\nstderr:%s", err, stderr.String())
	}
	if code != 0 {
		t.Fatalf("code %d, want 0", code)
	}
	for _, want := range []string{"bunpy.1", "bunpy-run.1", "bunpy-stdlib.1", "bunpy-version.1", "bunpy-help.1", "bunpy-man.1"} {
		path := filepath.Join(dir, "man1", want)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("missing %s: %v", path, err)
		}
	}
}
