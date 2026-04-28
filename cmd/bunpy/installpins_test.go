package main

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/tamnd/bunpy/v1/internal/httpkit"
	"github.com/tamnd/bunpy/v1/pkg/marker"
	"github.com/tamnd/bunpy/v1/pkg/pypi"
	"github.com/tamnd/bunpy/v1/pkg/resolver"
	"github.com/tamnd/bunpy/v1/pkg/wheel"
)

// benchFixtureRegistry returns a pypiRegistry wired against the
// benchmarks/fixtures/index fixture set.
func benchFixtureRegistry(t *testing.T, cacheDir string) *pypiRegistry {
	t.Helper()
	indexAbs, err := filepath.Abs("../../benchmarks/fixtures/index")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(indexAbs); err != nil {
		t.Skipf("benchmark fixtures missing (%v); run: go run benchmarks/fixtures/build_fixtures.go", err)
	}
	client := pypi.New()
	client.HTTP = httpkit.FixturesFS(indexAbs)
	t.Setenv("BUNPY_PYPI_FIXTURES", indexAbs)
	return newPypiRegistry(
		context.Background(),
		client,
		wheel.HostTags(),
		marker.DefaultEnv(),
		func(f pypi.File) ([]byte, error) { return fetchAddWheel(f, f.Filename, cacheDir) },
	)
}

// warmPins pre-populates the registry's pick table for each name
// and returns a Pin slice ready to pass to installPins.
func warmPins(t *testing.T, reg *pypiRegistry, names []string) []resolver.Pin {
	t.Helper()
	var pins []resolver.Pin
	for _, name := range names {
		versions, err := reg.Versions(name)
		if err != nil || len(versions) == 0 {
			t.Fatalf("versions %s: %v", name, err)
		}
		pins = append(pins, resolver.Pin{Name: name, Version: versions[0]})
	}
	return pins
}

// TestInstallParallel_NoDuplicates installs 10 distinct packages in
// parallel and verifies each lands in its own dist-info directory
// with no cross-package file conflicts.
func TestInstallParallel_NoDuplicates(t *testing.T) {
	cache := t.TempDir()
	t.Cleanup(func() { _ = os.RemoveAll(cache) })
	reg := benchFixtureRegistry(t, cache)

	names := []string{
		"pkg01", "pkg02", "pkg03", "pkg04", "pkg05",
		"pkg06", "pkg07", "pkg08", "pkg09", "pkg10",
	}
	pins := warmPins(t, reg, names)

	target := t.TempDir()
	if err := installPins(pins, reg, target, cache); err != nil {
		t.Fatalf("installPins: %v", err)
	}

	for _, name := range names {
		distInfo := filepath.Join(target, name+"-1.0.0.dist-info")
		if _, err := os.Stat(distInfo); err != nil {
			t.Errorf("dist-info missing for %s: %v", name, err)
		}
		initPy := filepath.Join(target, name, "__init__.py")
		if _, err := os.Stat(initPy); err != nil {
			t.Errorf("%s/__init__.py not installed: %v", name, err)
		}
	}
}

// TestInstallParallel_Idempotent verifies that running installPins
// twice for the same pins produces an identical site-packages tree.
func TestInstallParallel_Idempotent(t *testing.T) {
	cache := t.TempDir()
	t.Cleanup(func() { _ = os.RemoveAll(cache) })
	reg := benchFixtureRegistry(t, cache)

	names := []string{"pkg01", "pkg02", "pkg03"}
	pins := warmPins(t, reg, names)

	target := t.TempDir()

	if err := installPins(pins, reg, target, cache); err != nil {
		t.Fatalf("first installPins: %v", err)
	}
	files1 := walkFiles(t, target)

	if err := installPins(pins, reg, target, cache); err != nil {
		t.Fatalf("second installPins: %v", err)
	}
	files2 := walkFiles(t, target)

	if len(files1) != len(files2) {
		t.Errorf("file count changed after second install: %d -> %d", len(files1), len(files2))
	}
	for i := range files1 {
		if i >= len(files2) {
			break
		}
		if files1[i] != files2[i] {
			t.Errorf("file[%d] changed: %q -> %q", i, files1[i], files2[i])
		}
	}
}

func walkFiles(t *testing.T, dir string) []string {
	t.Helper()
	var out []string
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			rel, _ := filepath.Rel(dir, path)
			out = append(out, rel)
		}
		return nil
	})
	sort.Strings(out)
	return out
}
