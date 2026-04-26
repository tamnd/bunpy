package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestVersion(t *testing.T) {
	for _, arg := range []string{"version", "-v", "--version"} {
		var stdout, stderr bytes.Buffer
		if err := run([]string{arg}, &stdout, &stderr); err != nil {
			t.Fatalf("%s: %v", arg, err)
		}
		if !strings.HasPrefix(stdout.String(), version) {
			t.Errorf("%s: stdout %q does not start with version %q", arg, stdout.String(), version)
		}
	}
}

func TestHelp(t *testing.T) {
	for _, arg := range []string{"help", "-h", "--help"} {
		var stdout, stderr bytes.Buffer
		if err := run([]string{arg}, &stdout, &stderr); err != nil {
			t.Fatalf("%s: %v", arg, err)
		}
		if !strings.Contains(stdout.String(), "USAGE") {
			t.Errorf("%s: stdout missing USAGE section: %q", arg, stdout.String())
		}
	}
}

func TestNoArgsPrintsUsage(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := run(nil, &stdout, &stderr); err != nil {
		t.Fatalf("no args: %v", err)
	}
	if !strings.Contains(stdout.String(), "USAGE") {
		t.Errorf("no args: stdout missing USAGE section: %q", stdout.String())
	}
}

func TestUnknownCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := run([]string{"frobnicate"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected an error for unknown command")
	}
	if !strings.Contains(err.Error(), "frobnicate") {
		t.Errorf("error %q does not mention the bad command", err)
	}
}
