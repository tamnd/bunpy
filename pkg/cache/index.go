// Package cache holds the on-disk caches used by the v0.1.x
// package manager. v0.1.1 wires the PEP 691 simple-index cache;
// later rungs grow wheels/ and metadata/ subdirs alongside it.
//
// Disk layout (rooted at DefaultDir or an explicit override):
//
//	<root>/index/<normalized-name>/page.json
//	<root>/index/<normalized-name>/etag
//
// Writes are atomic via tempfile + rename: a crash mid-write
// leaves either the previous contents or no file, never a half
// file.
package cache

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Index is the PEP 691 simple-index cache.
type Index struct {
	Dir string
}

// NewIndex creates the directory if it does not exist and returns
// an Index rooted there.
func NewIndex(dir string) (*Index, error) {
	if dir == "" {
		return nil, errors.New("cache: empty dir")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &Index{Dir: dir}, nil
}

// Get returns the cached body and ETag for a project. ok is false
// when no cached body is present.
func (i *Index) Get(name string) (body []byte, etag string, ok bool) {
	dir := filepath.Join(i.Dir, normalize(name))
	b, err := os.ReadFile(filepath.Join(dir, "page.json"))
	if err != nil {
		return nil, "", false
	}
	if e, err := os.ReadFile(filepath.Join(dir, "etag")); err == nil {
		etag = strings.TrimSpace(string(e))
	}
	return b, etag, true
}

// Put writes body and etag atomically.
func (i *Index) Put(name string, body []byte, etag string) error {
	dir := filepath.Join(i.Dir, normalize(name))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if err := atomicWrite(filepath.Join(dir, "page.json"), body); err != nil {
		return err
	}
	if etag != "" {
		if err := atomicWrite(filepath.Join(dir, "etag"), []byte(etag)); err != nil {
			return err
		}
	}
	return nil
}

func atomicWrite(path string, body []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	cleanup := func() { _ = os.Remove(tmpName) }
	if _, err := tmp.Write(body); err != nil {
		_ = tmp.Close()
		cleanup()
		return err
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		cleanup()
		return err
	}
	return nil
}

// DefaultDir returns the default cache root. Honours
// XDG_CACHE_HOME on linux, $HOME/Library/Caches on macOS,
// %LOCALAPPDATA% on Windows.
func DefaultDir() string {
	if v := os.Getenv("BUNPY_CACHE_DIR"); v != "" {
		return v
	}
	switch runtime.GOOS {
	case "darwin":
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, "Library", "Caches", "bunpy")
		}
	case "windows":
		if v := os.Getenv("LOCALAPPDATA"); v != "" {
			return filepath.Join(v, "bunpy")
		}
	}
	if v := os.Getenv("XDG_CACHE_HOME"); v != "" {
		return filepath.Join(v, "bunpy")
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".cache", "bunpy")
	}
	return filepath.Join(os.TempDir(), "bunpy-cache")
}

// normalize is the PEP 503 normalised name. lowercase, then
// collapse runs of ._- into a single dash.
func normalize(name string) string {
	name = strings.ToLower(name)
	var b strings.Builder
	prevDash := false
	for _, r := range name {
		switch {
		case r == '-' || r == '_' || r == '.':
			if !prevDash {
				b.WriteByte('-')
				prevDash = true
			}
		default:
			b.WriteRune(r)
			prevDash = false
		}
	}
	return b.String()
}
