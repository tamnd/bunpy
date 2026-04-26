package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
		if !strings.HasPrefix(stdout.String(), version) {
			t.Errorf("%s: stdout %q does not start with version %q", arg, stdout.String(), version)
		}
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
