// Package bytecache caches gocopy-compiled IR alongside Python source files.
// Cache files live in __pycache__/<stem>.<hash16>.marshal next to the source.
package bytecache

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
)

func cacheDir(srcPath string) string {
	return filepath.Join(filepath.Dir(srcPath), "__pycache__")
}

func cachePath(srcPath string, src []byte) string {
	h := sha256.Sum256(src)
	hash := hex.EncodeToString(h[:])[:16]
	stem := strings.TrimSuffix(filepath.Base(srcPath), ".py")
	return filepath.Join(cacheDir(srcPath), stem+"."+hash+".marshal")
}

// Load returns cached IR for srcPath if the cache is valid.
// Returns (nil, false) on miss or error.
func Load(srcPath string, src []byte) ([]byte, bool) {
	path := cachePath(srcPath, src)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	return data, true
}

// Save writes IR bytes to the cache alongside srcPath.
func Save(srcPath string, src []byte, ir []byte) error {
	dir := cacheDir(srcPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(cachePath(srcPath, src), ir, 0o644)
}
