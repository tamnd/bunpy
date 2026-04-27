package build_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tamnd/bunpy/v1/pkg/build"
)

func hasPython() bool {
	_, err := build.FindPython()
	return err == nil
}

func TestFindPythonFound(t *testing.T) {
	p, err := build.FindPython()
	if err != nil {
		t.Skipf("python not on PATH: %v", err)
	}
	if p == "" {
		t.Error("FindPython returned empty path")
	}
}

func TestReadBackendHatchling(t *testing.T) {
	dir := t.TempDir()
	content := `[project]
name = "mypkg"
version = "0.1.0"

[build-system]
requires = ["hatchling"]
build-backend = "hatchling.build"
`
	if err := os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := build.ReadBackend(filepath.Join(dir, "pyproject.toml"))
	if err != nil {
		t.Fatalf("ReadBackend: %v", err)
	}
	if got != "hatchling.build" {
		t.Errorf("want hatchling.build, got %s", got)
	}
}

func TestReadBackendDefault(t *testing.T) {
	dir := t.TempDir()
	content := `[project]
name = "mypkg"
version = "0.1.0"
`
	if err := os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := build.ReadBackend(filepath.Join(dir, "pyproject.toml"))
	if err != nil {
		t.Fatalf("ReadBackend: %v", err)
	}
	if got != "hatchling.build" {
		t.Errorf("want default hatchling.build, got %s", got)
	}
}

func TestReadBackendMissingFile(t *testing.T) {
	got, err := build.ReadBackend("/nonexistent/pyproject.toml")
	if err != nil {
		t.Fatalf("ReadBackend with missing file should not error: %v", err)
	}
	if got != "hatchling.build" {
		t.Errorf("want hatchling.build, got %s", got)
	}
}

func TestBuildMissingPython(t *testing.T) {
	dir := t.TempDir()
	req := build.Request{
		ProjectDir: dir,
		Python:     "/nonexistent/python99",
		Backend:    "hatchling.build",
		BuildWheel: true,
	}
	_, err := build.Build(req)
	if err == nil {
		t.Fatal("want error for nonexistent python, got nil")
	}
}

func TestBuildMissingBackend(t *testing.T) {
	if !hasPython() {
		t.Skip("python not on PATH")
	}
	dir := t.TempDir()
	req := build.Request{
		ProjectDir: dir,
		Backend:    "totally_nonexistent_backend_xyz.build",
		BuildWheel: true,
	}
	_, err := build.Build(req)
	if err == nil {
		t.Fatal("want error for missing backend, got nil")
	}
}

func TestBuildOutputDir(t *testing.T) {
	if !hasPython() {
		t.Skip("python not on PATH")
	}
	// Check that hatchling is installed.
	python, _ := build.FindPython()
	out, err := exec.Command(python, "-c", "import hatchling").CombinedOutput()
	if err != nil {
		t.Skipf("hatchling not installed: %s", strings.TrimSpace(string(out)))
	}

	dir := t.TempDir()
	outDir := t.TempDir()

	// Minimal hatchling project.
	if err := os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte(`[project]
name = "testpkg"
version = "0.1.0"

[build-system]
requires = ["hatchling"]
build-backend = "hatchling.build"
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "src", "testpkg"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "src", "testpkg", "__init__.py"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	req := build.Request{
		ProjectDir: dir,
		OutputDir:  outDir,
		BuildWheel: true,
	}
	res, err := build.Build(req)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if res.Wheel == "" {
		t.Error("Wheel path is empty")
		return
	}
	if _, err := os.Stat(res.Wheel); err != nil {
		t.Errorf("Wheel file does not exist: %v", err)
	}
}
