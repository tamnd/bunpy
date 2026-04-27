package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUnlinkUnregistersCurrentProject(t *testing.T) {
	registry := t.TempDir()
	t.Setenv("BUNPY_LINK_DIR", registry)
	src := setupLinkSource(t, "widget", "1.0.0")
	chdirTo(t, src)

	var stdout, stderr bytes.Buffer
	if code, err := run([]string{"link"}, &stdout, &stderr); err != nil || code != 0 {
		t.Fatalf("register: code=%d err=%v stderr=%s", code, err, stderr.String())
	}
	if _, err := os.Stat(filepath.Join(registry, "widget.json")); err != nil {
		t.Fatalf("entry should exist: %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	code, err := run([]string{"unlink"}, &stdout, &stderr)
	if err != nil || code != 0 {
		t.Fatalf("unlink: code=%d err=%v stderr=%s", code, err, stderr.String())
	}
	if _, err := os.Stat(filepath.Join(registry, "widget.json")); !os.IsNotExist(err) {
		t.Errorf("entry should be gone: err=%v", err)
	}
}

func TestUnlinkRemovesEditableProxy(t *testing.T) {
	t.Setenv("BUNPY_LINK_DIR", t.TempDir())
	src := setupLinkSource(t, "widget", "1.0.0")
	consumer := t.TempDir()
	if err := os.WriteFile(filepath.Join(consumer, "pyproject.toml"), []byte("[project]\nname = \"demo\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Register + link.
	chdirTo(t, src)
	{
		var stdout, stderr bytes.Buffer
		if code, err := run([]string{"link"}, &stdout, &stderr); err != nil || code != 0 {
			t.Fatalf("register: code=%d err=%v stderr=%s", code, err, stderr.String())
		}
	}
	chdirTo(t, consumer)
	{
		var stdout, stderr bytes.Buffer
		if code, err := run([]string{"link", "widget"}, &stdout, &stderr); err != nil || code != 0 {
			t.Fatalf("link widget: code=%d err=%v stderr=%s", code, err, stderr.String())
		}
	}

	// Unlink from the consumer.
	var stdout, stderr bytes.Buffer
	code, err := run([]string{"unlink", "widget"}, &stdout, &stderr)
	if err != nil || code != 0 {
		t.Fatalf("unlink widget: code=%d err=%v stderr=%s", code, err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "unlinked widget 1.0.0") {
		t.Errorf("expected unlink summary, got: %q", stdout.String())
	}
	if _, err := os.Stat(filepath.Join(consumer, ".bunpy", "site-packages", "widget.pth")); !os.IsNotExist(err) {
		t.Errorf(".pth should be gone: err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(consumer, ".bunpy", "site-packages", "widget-1.0.0.dist-info")); !os.IsNotExist(err) {
		t.Errorf("dist-info should be gone: err=%v", err)
	}
}

func TestUnlinkMissingPackageIsNoop(t *testing.T) {
	t.Setenv("BUNPY_LINK_DIR", t.TempDir())
	consumer := t.TempDir()
	if err := os.WriteFile(filepath.Join(consumer, "pyproject.toml"), []byte("[project]\nname = \"demo\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	chdirTo(t, consumer)

	var stdout, stderr bytes.Buffer
	code, err := run([]string{"unlink", "widget"}, &stdout, &stderr)
	if err != nil || code != 0 {
		t.Fatalf("unlink missing: code=%d err=%v stderr=%s", code, err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "no link for widget") {
		t.Errorf("expected `no link for widget`, got: %q", stdout.String())
	}
}

func TestUnlinkUnknownProjectIsNoop(t *testing.T) {
	t.Setenv("BUNPY_LINK_DIR", t.TempDir())
	src := setupLinkSource(t, "widget", "1.0.0")
	chdirTo(t, src)

	var stdout, stderr bytes.Buffer
	code, err := run([]string{"unlink"}, &stdout, &stderr)
	if err != nil || code != 0 {
		t.Fatalf("unlink (no entry): code=%d err=%v stderr=%s", code, err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "unregistered widget") {
		t.Errorf("expected `unregistered widget`, got: %q", stdout.String())
	}
}
