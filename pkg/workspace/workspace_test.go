package workspace_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tamnd/bunpy/v1/pkg/workspace"
)

func writeManifest(t *testing.T, dir, content string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func rootManifest(members ...string) string {
	parts := make([]string, len(members))
	for i, m := range members {
		parts[i] = `"` + m + `"`
	}
	return `[project]
name = "root"
version = "0.1.0"

[tool.bunpy.workspace]
members = [` + strings.Join(parts, ", ") + `]
`
}

func memberManifest(name string) string {
	return `[project]
name = "` + name + `"
version = "0.1.0"
`
}

func TestLoadThreeMembers(t *testing.T) {
	root := t.TempDir()
	writeManifest(t, root, rootManifest("pkgs/alpha", "pkgs/beta", "pkgs/gamma"))
	writeManifest(t, filepath.Join(root, "pkgs", "alpha"), memberManifest("alpha"))
	writeManifest(t, filepath.Join(root, "pkgs", "beta"), memberManifest("beta"))
	writeManifest(t, filepath.Join(root, "pkgs", "gamma"), memberManifest("gamma"))

	ws, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(ws.Members) != 3 {
		t.Fatalf("want 3 members, got %d", len(ws.Members))
	}
	names := map[string]bool{}
	for _, m := range ws.Members {
		names[m.Name] = true
	}
	for _, want := range []string{"alpha", "beta", "gamma"} {
		if !names[want] {
			t.Errorf("member %q missing", want)
		}
	}
}

func TestLoadGlobExpansion(t *testing.T) {
	root := t.TempDir()
	writeManifest(t, root, rootManifest("packages/*"))
	writeManifest(t, filepath.Join(root, "packages", "alpha"), memberManifest("alpha"))
	writeManifest(t, filepath.Join(root, "packages", "beta"), memberManifest("beta"))

	ws, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(ws.Members) != 2 {
		t.Fatalf("want 2 members, got %d", len(ws.Members))
	}
}

func TestFindRootFromNested(t *testing.T) {
	root := t.TempDir()
	writeManifest(t, root, rootManifest("pkgs/alpha"))
	writeManifest(t, filepath.Join(root, "pkgs", "alpha"), memberManifest("alpha"))

	nested := filepath.Join(root, "pkgs", "alpha", "src", "mypkg")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := workspace.FindRoot(nested)
	if err != nil {
		t.Fatalf("FindRoot: %v", err)
	}
	if got != root {
		t.Errorf("want %s, got %s", root, got)
	}
}

func TestFindRootNotFound(t *testing.T) {
	dir := t.TempDir()
	_, err := workspace.FindRoot(dir)
	if err == nil {
		t.Fatal("want ErrNoWorkspace, got nil")
	}
}

func TestLoadDuplicateNameError(t *testing.T) {
	root := t.TempDir()
	writeManifest(t, root, rootManifest("pkgs/alpha", "pkgs/beta"))
	writeManifest(t, filepath.Join(root, "pkgs", "alpha"), memberManifest("same-name"))
	writeManifest(t, filepath.Join(root, "pkgs", "beta"), memberManifest("same-name"))

	_, err := workspace.Load(root)
	if err == nil {
		t.Fatal("want error for duplicate name, got nil")
	}
}

func TestMemberByCwd(t *testing.T) {
	root := t.TempDir()
	writeManifest(t, root, rootManifest("pkgs/alpha", "pkgs/beta"))
	writeManifest(t, filepath.Join(root, "pkgs", "alpha"), memberManifest("alpha"))
	writeManifest(t, filepath.Join(root, "pkgs", "beta"), memberManifest("beta"))

	ws, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	cwd := filepath.Join(root, "pkgs", "alpha", "src")
	if err := os.MkdirAll(cwd, 0o755); err != nil {
		t.Fatal(err)
	}

	m, ok := workspace.MemberByCwd(ws, cwd)
	if !ok {
		t.Fatal("MemberByCwd: not found")
	}
	if m.Name != "alpha" {
		t.Errorf("want alpha, got %s", m.Name)
	}
}
