package cache

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWheelCacheRoundtrip(t *testing.T) {
	dir := t.TempDir()
	c, err := NewWheelCache(dir)
	if err != nil {
		t.Fatalf("NewWheelCache: %v", err)
	}
	body := []byte("PK\x03\x04 fake wheel body")
	if err := c.Put("Foo_Bar", "foo_bar-1.0-py3-none-any.whl", body); err != nil {
		t.Fatalf("Put: %v", err)
	}
	if !c.Has("foo-bar", "foo_bar-1.0-py3-none-any.whl") {
		t.Fatal("Has returned false after Put with normalised alias")
	}
	got, err := os.ReadFile(c.Path("FOO.BAR", "foo_bar-1.0-py3-none-any.whl"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != string(body) {
		t.Errorf("body mismatch: got %q, want %q", got, body)
	}
}

func TestWheelCacheAtomic(t *testing.T) {
	dir := t.TempDir()
	c, _ := NewWheelCache(dir)
	if err := c.Put("pkg", "pkg-1.0.whl", []byte("body")); err != nil {
		t.Fatalf("Put: %v", err)
	}
	// No leftover .tmp-* files in the package subdir.
	pkgDir := filepath.Join(dir, "pkg")
	entries, err := os.ReadDir(pkgDir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".tmp-") {
			t.Errorf("leftover temp file: %s", e.Name())
		}
	}
}
