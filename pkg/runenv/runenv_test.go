package runenv

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// makeWheel builds a minimal .whl zip in memory and writes it to a temp file.
// files is a map of zip path -> content.
func makeWheel(t *testing.T, files map[string]string) string {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	f, err := os.CreateTemp(t.TempDir(), "*.whl")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write(buf.Bytes()); err != nil {
		t.Fatal(err)
	}
	f.Close()
	return f.Name()
}

func TestCreateMakesDir(t *testing.T) {
	e, err := Create("")
	if err != nil {
		t.Fatal(err)
	}
	defer e.Cleanup()
	if _, err := os.Stat(e.Dir); err != nil {
		t.Fatalf("Dir does not exist: %v", err)
	}
}

func TestInstallUnpacksWheel(t *testing.T) {
	whl := makeWheel(t, map[string]string{
		"mylib/__init__.py":               "# mylib\n",
		"mylib-1.0.dist-info/METADATA":    "Name: mylib\nVersion: 1.0\n",
		"mylib-1.0.dist-info/WHEEL":       "Wheel-Version: 1.0\nGenerator: test\nRoot-Is-Purelib: true\n",
		"mylib-1.0.dist-info/RECORD":      "",
	})
	e, err := Create("")
	if err != nil {
		t.Fatal(err)
	}
	defer e.Cleanup()

	if err := e.Install(whl); err != nil {
		t.Fatal(err)
	}
	metadata := filepath.Join(e.Dir, "site-packages", "mylib-1.0.dist-info", "METADATA")
	if _, err := os.Stat(metadata); err != nil {
		t.Fatalf("METADATA not found: %v", err)
	}
}

func TestInstallCreatesShim(t *testing.T) {
	eps := "[console_scripts]\nmylib = mylib:main\n"
	whl := makeWheel(t, map[string]string{
		"mylib/__init__.py":                    "def main(): pass\n",
		"mylib-1.0.dist-info/METADATA":         "Name: mylib\nVersion: 1.0\n",
		"mylib-1.0.dist-info/WHEEL":            "Wheel-Version: 1.0\nGenerator: test\nRoot-Is-Purelib: true\n",
		"mylib-1.0.dist-info/entry_points.txt": eps,
		"mylib-1.0.dist-info/RECORD":           "",
	})
	e, err := Create("")
	if err != nil {
		t.Fatal(err)
	}
	defer e.Cleanup()

	if err := e.Install(whl); err != nil {
		t.Fatal(err)
	}
	// shim should exist somewhere in bin/
	entries, err := os.ReadDir(filepath.Join(e.Dir, "bin"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) == 0 {
		t.Fatal("no shims created in bin/")
	}
}

func TestEntryPointFound(t *testing.T) {
	eps := "[console_scripts]\nmytool = mytool:main\n"
	whl := makeWheel(t, map[string]string{
		"mytool/__init__.py":                    "def main(): pass\n",
		"mytool-2.0.dist-info/METADATA":         "Name: mytool\nVersion: 2.0\n",
		"mytool-2.0.dist-info/WHEEL":            "Wheel-Version: 1.0\nGenerator: test\nRoot-Is-Purelib: true\n",
		"mytool-2.0.dist-info/entry_points.txt": eps,
		"mytool-2.0.dist-info/RECORD":           "",
	})
	e, err := Create("")
	if err != nil {
		t.Fatal(err)
	}
	defer e.Cleanup()

	if err := e.Install(whl); err != nil {
		t.Fatal(err)
	}
	path, ok := e.EntryPoint("mytool")
	if !ok {
		t.Fatal("EntryPoint returned false, expected true")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("shim path not valid: %v", err)
	}
}

func TestEntryPointNotFound(t *testing.T) {
	e, err := Create("")
	if err != nil {
		t.Fatal(err)
	}
	defer e.Cleanup()

	_, ok := e.EntryPoint("nonexistent")
	if ok {
		t.Fatal("EntryPoint returned true for nonexistent entry")
	}
}

func TestCleanupRemovesDir(t *testing.T) {
	e, err := Create("")
	if err != nil {
		t.Fatal(err)
	}
	dir := e.Dir
	if err := e.Cleanup(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("Dir still exists after Cleanup: %s", dir)
	}
}
