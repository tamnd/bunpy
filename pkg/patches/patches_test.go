package patches

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tamnd/bunpy/v1/pkg/manifest"
)

func writeTree(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for rel, body := range files {
		full := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}
}

func readAll(t *testing.T, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		body, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, p)
		out[filepath.ToSlash(rel)] = string(body)
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	return out
}

func TestDiffApplyRoundTripModifiesFile(t *testing.T) {
	pristine := t.TempDir()
	scratch := t.TempDir()
	writeTree(t, pristine, map[string]string{
		"widget/__init__.py": "VERSION = '1.0.0'\n",
	})
	writeTree(t, scratch, map[string]string{
		"widget/__init__.py": "RUNTIME_VERSION = '1.0.0'\n",
	})

	body, err := Diff(pristine, scratch)
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	if !strings.Contains(string(body), "--- a/widget/__init__.py") {
		t.Errorf("missing left header: %q", body)
	}
	if !strings.Contains(string(body), "+++ b/widget/__init__.py") {
		t.Errorf("missing right header: %q", body)
	}

	target := t.TempDir()
	writeTree(t, target, map[string]string{
		"widget/__init__.py": "VERSION = '1.0.0'\n",
	})
	if err := Apply(target, body); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	got := readAll(t, target)
	if got["widget/__init__.py"] != "RUNTIME_VERSION = '1.0.0'\n" {
		t.Errorf("after Apply got %q", got["widget/__init__.py"])
	}
}

func TestDiffApplyRoundTripAddsFile(t *testing.T) {
	pristine := t.TempDir()
	scratch := t.TempDir()
	writeTree(t, pristine, map[string]string{
		"widget/__init__.py": "x = 1\n",
	})
	writeTree(t, scratch, map[string]string{
		"widget/__init__.py": "x = 1\n",
		"widget/extra.py":    "y = 2\n",
	})

	body, err := Diff(pristine, scratch)
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	if !strings.Contains(string(body), "--- /dev/null") {
		t.Errorf("missing /dev/null left header: %q", body)
	}

	target := t.TempDir()
	writeTree(t, target, map[string]string{
		"widget/__init__.py": "x = 1\n",
	})
	if err := Apply(target, body); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	got := readAll(t, target)
	if got["widget/extra.py"] != "y = 2\n" {
		t.Errorf("after Apply: extra.py = %q", got["widget/extra.py"])
	}
}

func TestDiffApplyRoundTripRemovesFile(t *testing.T) {
	pristine := t.TempDir()
	scratch := t.TempDir()
	writeTree(t, pristine, map[string]string{
		"widget/__init__.py": "x = 1\n",
		"widget/old.py":      "z = 3\n",
	})
	writeTree(t, scratch, map[string]string{
		"widget/__init__.py": "x = 1\n",
	})

	body, err := Diff(pristine, scratch)
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	target := t.TempDir()
	writeTree(t, target, map[string]string{
		"widget/__init__.py": "x = 1\n",
		"widget/old.py":      "z = 3\n",
	})
	if err := Apply(target, body); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "widget", "old.py")); !os.IsNotExist(err) {
		t.Errorf("old.py should be gone: err=%v", err)
	}
}

func TestApplyRejectsContextMismatch(t *testing.T) {
	body := []byte(`--- a/x.py
+++ b/x.py
@@ -1,1 +1,1 @@
-pristine
+patched
`)
	target := t.TempDir()
	writeTree(t, target, map[string]string{"x.py": "different\n"})
	err := Apply(target, body)
	if err == nil {
		t.Fatal("expected error on context mismatch")
	}
	if !strings.Contains(err.Error(), "x.py") {
		t.Errorf("error should name file: %v", err)
	}
}

func TestDiffRejectsBinary(t *testing.T) {
	pristine := t.TempDir()
	scratch := t.TempDir()
	writeTree(t, pristine, map[string]string{"blob.bin": "alpha"})
	if err := os.WriteFile(filepath.Join(scratch, "blob.bin"), []byte{0x00, 0x01, 0x02}, 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Diff(pristine, scratch)
	if err == nil {
		t.Fatal("expected binary refusal")
	}
	if !strings.Contains(err.Error(), "binary") {
		t.Errorf("error should mention binary: %v", err)
	}
}

func TestApplyRejectsPathEscape(t *testing.T) {
	body := []byte("--- /dev/null\n+++ b/../escape.txt\n@@ -0,0 +1,1 @@\n+haha\n")
	target := t.TempDir()
	if err := Apply(target, body); err == nil {
		t.Fatal("expected path-escape refusal")
	}
}

func TestReadFromManifest(t *testing.T) {
	src := []byte(`[project]
name = "demo"

[tool.bunpy.patches]
"flask@2.3.0" = "patches/flask+2.3.0.patch"
"requests@2.32.3" = "patches/requests+2.32.3.patch"
`)
	m, err := manifest.Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	entries, err := Read(m)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("entries = %d, want 2", len(entries))
	}
	if entries[0].Key() != "flask@2.3.0" {
		t.Errorf("first key = %q", entries[0].Key())
	}
	got, ok := Lookup(entries, "Flask", "2.3.0")
	if !ok || got.Path != "patches/flask+2.3.0.patch" {
		t.Errorf("Lookup(Flask) = %+v, %v", got, ok)
	}
}

func TestDiffNoChangesReturnsEmpty(t *testing.T) {
	pristine := t.TempDir()
	scratch := t.TempDir()
	writeTree(t, pristine, map[string]string{"x.py": "same\n"})
	writeTree(t, scratch, map[string]string{"x.py": "same\n"})
	body, err := Diff(pristine, scratch)
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	if !bytes.Equal(body, nil) && len(body) != 0 {
		t.Errorf("expected empty diff, got %q", body)
	}
}
