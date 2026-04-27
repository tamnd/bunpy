package cache

import (
	"errors"
	"os"
	"path/filepath"
)

// WheelCache stores wheel bodies under <Dir>/<normalized-name>/<filename>.
//
// Atomic writes via tempfile + rename, same shape as Index.Put. The
// installer reaches Put once per fetched wheel; later runs hit Has
// and read the body straight off disk so we never refetch a frozen
// artefact.
type WheelCache struct {
	Dir string
}

// NewWheelCache creates the directory if missing and returns a
// WheelCache rooted there.
func NewWheelCache(dir string) (*WheelCache, error) {
	if dir == "" {
		return nil, errors.New("cache: empty wheel cache dir")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &WheelCache{Dir: dir}, nil
}

// Path returns the on-disk path for a wheel filename under name.
// The file may or may not exist.
func (c *WheelCache) Path(name, filename string) string {
	return filepath.Join(c.Dir, normalize(name), filename)
}

// Has reports whether a wheel body for name/filename is on disk.
func (c *WheelCache) Has(name, filename string) bool {
	if st, err := os.Stat(c.Path(name, filename)); err == nil && !st.IsDir() {
		return true
	}
	return false
}

// Put writes body atomically to Path(name, filename).
func (c *WheelCache) Put(name, filename string, body []byte) error {
	dir := filepath.Join(c.Dir, normalize(name))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return atomicWrite(filepath.Join(dir, filename), body)
}
