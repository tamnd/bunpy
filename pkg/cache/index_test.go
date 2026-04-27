package cache

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIndexRoundtrip(t *testing.T) {
	idx, err := NewIndex(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	body := []byte(`{"x":1}`)
	if err := idx.Put("Demo_Pkg", body, "\"abc\""); err != nil {
		t.Fatal(err)
	}
	got, etag, ok := idx.Get("demo-pkg")
	if !ok {
		t.Fatal("Get: not found after Put")
	}
	if string(got) != string(body) {
		t.Errorf("body: got %q, want %q", got, body)
	}
	if etag != "\"abc\"" {
		t.Errorf("etag: got %q", etag)
	}
}

func TestIndexNormalizes(t *testing.T) {
	idx, err := NewIndex(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := idx.Put("Foo_Bar", []byte("v"), ""); err != nil {
		t.Fatal(err)
	}
	for _, alias := range []string{"foo-bar", "Foo_Bar", "FOO_BAR", "foo.bar", "foo--bar", "Foo-Bar"} {
		if _, _, ok := idx.Get(alias); !ok {
			t.Errorf("Get %q: not found", alias)
		}
	}
}

func TestIndexAtomicLeavesNoTemp(t *testing.T) {
	dir := t.TempDir()
	idx, err := NewIndex(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := idx.Put("p", []byte("a"), ""); err != nil {
		t.Fatal(err)
	}
	pkgDir := filepath.Join(dir, "p")
	entries, err := os.ReadDir(pkgDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if len(e.Name()) >= 5 && e.Name()[:5] == ".tmp-" {
			t.Errorf("leftover temp: %s", e.Name())
		}
	}
}

func TestIndexMissing(t *testing.T) {
	idx, err := NewIndex(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, _, ok := idx.Get("nope"); ok {
		t.Error("Get: want false on empty cache")
	}
}

func TestDefaultDir(t *testing.T) {
	t.Setenv("BUNPY_CACHE_DIR", "/explicit/override")
	if got := DefaultDir(); got != "/explicit/override" {
		t.Errorf("DefaultDir with env: got %q", got)
	}
}
