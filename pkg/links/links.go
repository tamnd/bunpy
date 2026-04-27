// Package links manages the global registry of editable
// "linked" packages. The registry lives under
// $BUNPY_LINK_DIR (or the platform's user-data dir). Each
// linked package is a single JSON file
// `<normalised-name>.json`; v0.1.9 keeps the schema minimal so
// future fields (extras, entry points) can land without a bump.
package links

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

// ErrNotFound is returned by Read when no entry exists for the
// normalised name.
var ErrNotFound = errors.New("links: not found")

// Entry is one row in the global link registry. Source is the
// absolute path to the package source root (the directory that
// holds pyproject.toml, not the import root).
type Entry struct {
	Name       string    `json:"name"`
	Version    string    `json:"version,omitempty"`
	Source     string    `json:"source"`
	Registered time.Time `json:"registered"`
}

// Dir resolves the registry root. Honours BUNPY_LINK_DIR for tests
// and CI; otherwise picks the platform's user-data directory.
func Dir() (string, error) {
	if v := os.Getenv("BUNPY_LINK_DIR"); v != "" {
		return v, nil
	}
	switch runtime.GOOS {
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, "Library", "Application Support", "bunpy", "links"), nil
	case "windows":
		if v := os.Getenv("LOCALAPPDATA"); v != "" {
			return filepath.Join(v, "bunpy", "links"), nil
		}
	}
	if v := os.Getenv("XDG_DATA_HOME"); v != "" {
		return filepath.Join(v, "bunpy", "links"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "share", "bunpy", "links"), nil
}

// Path returns the JSON file for the (already-normalised) name.
func Path(dir, name string) string {
	return filepath.Join(dir, name+".json")
}

// Read returns the entry for name, or ErrNotFound on miss.
// name must already be PEP 503 normalised (callers are expected
// to use pypi.Normalize before reaching this package).
func Read(name string) (Entry, error) {
	dir, err := Dir()
	if err != nil {
		return Entry{}, err
	}
	data, err := os.ReadFile(Path(dir, name))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Entry{}, ErrNotFound
		}
		return Entry{}, err
	}
	var e Entry
	if err := json.Unmarshal(data, &e); err != nil {
		return Entry{}, fmt.Errorf("links: parse %s: %w", name, err)
	}
	return e, nil
}

// Write upserts e into the registry. The write is atomic:
// tempfile in the same directory, then rename.
func Write(e Entry) error {
	if e.Name == "" {
		return errors.New("links: Write requires Entry.Name")
	}
	if e.Source == "" {
		return errors.New("links: Write requires Entry.Source")
	}
	if e.Registered.IsZero() {
		e.Registered = time.Now().UTC()
	}
	dir, err := Dir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	body, err := json.MarshalIndent(e, "", "  ")
	if err != nil {
		return err
	}
	body = append(body, '\n')
	final := Path(dir, e.Name)
	tmp, err := os.CreateTemp(dir, "."+e.Name+".*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()
	if _, err := tmp.Write(body); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, final)
}

// Delete removes the entry for name. Missing entry is not an
// error: the verb is idempotent.
func Delete(name string) error {
	dir, err := Dir()
	if err != nil {
		return err
	}
	if err := os.Remove(Path(dir, name)); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}

// List returns every registered entry, sorted by normalised name.
// Missing dir returns an empty slice (idempotent: the registry
// hasn't been touched yet).
func List() ([]Entry, error) {
	dir, err := Dir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	out := make([]Entry, 0, len(entries))
	for _, ent := range entries {
		if ent.IsDir() || !strings.HasSuffix(ent.Name(), ".json") {
			continue
		}
		name := strings.TrimSuffix(ent.Name(), ".json")
		e, err := Read(name)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				continue
			}
			return nil, err
		}
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}
