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
	"time"

	jsonv2 "github.com/go-json-experiment/json"
)

var jsonMarshal = jsonv2.Marshal
var jsonUnmarshal = jsonv2.Unmarshal

// Index is the PEP 691 simple-index cache.
type Index struct {
	Dir string
	// Freshness is how long a cached page is considered fresh. When
	// positive, Get returns the cache body without any HTTP request if
	// the page was stored within this duration. Zero means always
	// revalidate (default; existing behaviour).
	Freshness time.Duration
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
// when no cached body is present. When the Index has a positive Freshness
// and the cached page is newer than that duration, fresh is true: the
// caller should skip the HTTP round-trip entirely and use body directly.
func (i *Index) Get(name string) (body []byte, etag string, ok bool, fresh bool) {
	dir := filepath.Join(i.Dir, normalize(name))
	path := filepath.Join(dir, "page.json")
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, "", false, false
	}
	if e, err := os.ReadFile(filepath.Join(dir, "etag")); err == nil {
		etag = strings.TrimSpace(string(e))
	}
	if i.Freshness > 0 {
		if info, err := os.Stat(path); err == nil && time.Since(info.ModTime()) < i.Freshness {
			fresh = true
		}
	}
	return b, etag, true, fresh
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

// ETag returns just the cached ETag for a project without loading the
// (potentially multi-megabyte) page body. Returns "" on any error.
func (i *Index) ETag(name string) string {
	p := filepath.Join(i.Dir, normalize(name), "etag")
	b, err := os.ReadFile(p)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

// GetPicks returns the cached (version → wheel filename) map for the
// given project, tags key, and ETag. ok is false when no entry exists
// or when the stored ETag doesn't match (meaning the index page is stale).
func (i *Index) GetPicks(name, tagsKey, etag string) (picks map[string]string, ok bool) {
	dir := filepath.Join(i.Dir, normalize(name))
	raw, err := os.ReadFile(filepath.Join(dir, "picks-"+tagsKey+".json"))
	if err != nil {
		return nil, false
	}
	var stored struct {
		ETag  string            `json:"etag"`
		Picks map[string]string `json:"picks"`
	}
	if err := jsonUnmarshal(raw, &stored); err != nil {
		return nil, false
	}
	if stored.ETag != etag {
		return nil, false
	}
	return stored.Picks, true
}

// PutPicks stores the (version → wheel filename) map for (name, tagsKey)
// associated with the given ETag so it can be validated on future reads.
func (i *Index) PutPicks(name, tagsKey string, picks map[string]string, etag string) error {
	dir := filepath.Join(i.Dir, normalize(name))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	stored := struct {
		ETag  string            `json:"etag"`
		Picks map[string]string `json:"picks"`
	}{ETag: etag, Picks: picks}
	body, err := jsonMarshal(stored)
	if err != nil {
		return err
	}
	return atomicWrite(filepath.Join(dir, "picks-"+tagsKey+".json"), body)
}

// GetMetadata returns the cached METADATA bytes for a wheel filename.
// ok is false when no entry is present.
func (i *Index) GetMetadata(filename string) (body []byte, ok bool) {
	p := filepath.Join(i.Dir, "metadata", filename+".metadata")
	b, err := os.ReadFile(p)
	if err != nil {
		return nil, false
	}
	return b, true
}

// PutMetadata caches METADATA bytes for a wheel filename atomically.
func (i *Index) PutMetadata(filename string, body []byte) error {
	dir := filepath.Join(i.Dir, "metadata")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return atomicWrite(filepath.Join(dir, filename+".metadata"), body)
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
