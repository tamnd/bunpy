package bytecache_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tamnd/bunpy/v1/internal/bytecache"
)

func TestCacheSaveLoad(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "mod.py")
	src := []byte("x = 1\n")
	ir := []byte("compiled-ir-bytes")

	if err := bytecache.Save(srcPath, src, ir); err != nil {
		t.Fatal(err)
	}

	got, ok := bytecache.Load(srcPath, src)
	if !ok {
		t.Fatal("expected cache hit")
	}
	if string(got) != string(ir) {
		t.Errorf("got %q, want %q", got, ir)
	}
}

func TestCacheMiss(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "mod.py")
	src := []byte("x = 1\n")

	_, ok := bytecache.Load(srcPath, src)
	if ok {
		t.Error("expected cache miss for non-existent entry")
	}
}

func TestCacheInvalidated(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "mod.py")
	src := []byte("x = 1\n")
	ir := []byte("ir-v1")

	bytecache.Save(srcPath, src, ir)

	// Different source content → different hash → miss.
	srcV2 := []byte("x = 2\n")
	_, ok := bytecache.Load(srcPath, srcV2)
	if ok {
		t.Error("expected cache miss for different source content")
	}
}

func TestCacheDirCreated(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "sub", "mod.py")
	src := []byte("y = 99\n")
	ir := []byte("ir-data")

	if err := bytecache.Save(srcPath, src, ir); err != nil {
		t.Fatal(err)
	}

	cacheDir := filepath.Join(dir, "sub", "__pycache__")
	if _, err := os.Stat(cacheDir); err != nil {
		t.Errorf("__pycache__ dir not created: %v", err)
	}
}
