package scaffold_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tamnd/bunpy/v1/pkg/scaffold"
)

func defaultVars(name string) scaffold.Vars {
	return scaffold.Vars{
		Name:        name,
		SnakeName:   scaffold.SnakeName(name),
		Description: "A bunpy project",
		Author:      "Test User",
		PythonMin:   ">=3.11",
	}
}

func TestRenderApp(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "my-cli")
	vars := defaultVars("my-cli")

	created, err := scaffold.Render("app", vars, dest)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if len(created) == 0 {
		t.Fatal("no files created")
	}

	// pyproject.toml exists and has the project name.
	pyproject := filepath.Join(dest, "pyproject.toml")
	data, err := os.ReadFile(pyproject)
	if err != nil {
		t.Fatalf("pyproject.toml not found: %v", err)
	}
	if !strings.Contains(string(data), `name = "my-cli"`) {
		t.Errorf("pyproject.toml missing name = \"my-cli\"\n%s", data)
	}

	// __main__.py exists under the snake name.
	main := filepath.Join(dest, "src", "my_cli", "__main__.py")
	if _, err := os.Stat(main); err != nil {
		t.Errorf("__main__.py not found at %s: %v", main, err)
	}
}

func TestRenderLib(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "my-lib")
	vars := defaultVars("my-lib")

	_, err := scaffold.Render("lib", vars, dest)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	pyproject := filepath.Join(dest, "pyproject.toml")
	data, _ := os.ReadFile(pyproject)
	if strings.Contains(string(data), "[project.scripts]") {
		t.Error("lib pyproject.toml should not have [project.scripts]")
	}

	init := filepath.Join(dest, "src", "my_lib", "__init__.py")
	if _, err := os.Stat(init); err != nil {
		t.Errorf("__init__.py not found: %v", err)
	}

	testFile := filepath.Join(dest, "tests", "test_my_lib.py")
	if _, err := os.Stat(testFile); err != nil {
		t.Errorf("test file not found: %v", err)
	}
}

func TestRenderScript(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "scraper")
	vars := defaultVars("scraper")

	_, err := scaffold.Render("script", vars, dest)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	script := filepath.Join(dest, "scraper.py")
	data, err := os.ReadFile(script)
	if err != nil {
		t.Fatalf("script file not found: %v", err)
	}
	if !strings.Contains(string(data), "#!/usr/bin/env bunpy") {
		t.Error("script missing shebang")
	}
}

func TestRenderWorkspace(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "mono")
	vars := defaultVars("mono")

	_, err := scaffold.Render("workspace", vars, dest)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	pyproject := filepath.Join(dest, "pyproject.toml")
	data, err := os.ReadFile(pyproject)
	if err != nil {
		t.Fatalf("pyproject.toml not found: %v", err)
	}
	if !strings.Contains(string(data), `members = ["packages/alpha", "packages/beta"]`) {
		t.Errorf("workspace pyproject.toml missing members line\n%s", data)
	}
}

func TestRenderDestExists(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "existing")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := scaffold.Render("app", defaultVars("existing"), dest)
	if err == nil {
		t.Fatal("want error for existing dest, got nil")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestListAndLookup(t *testing.T) {
	templates := scaffold.List()
	if len(templates) != 4 {
		t.Errorf("want 4 templates, got %d", len(templates))
	}
	for _, want := range []string{"app", "lib", "script", "workspace"} {
		found := false
		for _, tmpl := range templates {
			if tmpl.Name == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("template %q not in List()", want)
		}
	}
	_, ok := scaffold.Lookup("app")
	if !ok {
		t.Error("Lookup(app) returned false")
	}
	_, ok = scaffold.Lookup("unknown-template")
	if ok {
		t.Error("Lookup(unknown) returned true")
	}
}
