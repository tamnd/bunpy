package bundler

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type cacheSourceEntry struct {
	Path   string `json:"path"`   // absolute path to source file
	SHA256 string `json:"sha256"` // hex SHA-256 of file contents
}

type cacheOutputInfo struct {
	Path   string `json:"path"`   // output archive path
	SHA256 string `json:"sha256"` // hex SHA-256 of archive bytes
}

type cacheManifest struct {
	BunpyVersion string                      `json:"bunpy_version"`
	Entry        string                      `json:"entry"`
	FlagsHash    string                      `json:"flags_hash"`
	Sources      map[string]cacheSourceEntry `json:"sources"`
	Output       cacheOutputInfo             `json:"output"`
}

// ManifestPath returns the path of the build cache manifest for the given
// entry-file directory.
func ManifestPath(entryDir string) string {
	return filepath.Join(entryDir, ".bunpy", "build-cache", "manifest.json")
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func flagsHash(opts Options) string {
	type key struct {
		Outfile   string
		Outdir    string
		Minify    bool
		Target    string
		Defines   map[string]string
		Plugins   []string
		SourceMap bool
		Compile   bool
	}
	k := key{
		Outfile:   opts.Outfile,
		Outdir:    opts.Outdir,
		Minify:    opts.Minify,
		Target:    opts.Target,
		Defines:   opts.Defines,
		Plugins:   opts.Plugins,
		SourceMap: opts.SourceMap,
		Compile:   opts.Compile,
	}
	b, _ := json.Marshal(k)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

// CheckCache checks whether a previously cached build is still valid for the
// given entry and options. Returns hit=true if all source hashes and the
// output archive hash match the manifest. A missing or malformed manifest is
// treated as a miss (no error).
func CheckCache(entryAbs string, opts Options, bunpyVersion string) (hit bool, err error) {
	data, err := os.ReadFile(ManifestPath(filepath.Dir(entryAbs)))
	if err != nil {
		return false, nil
	}
	var m cacheManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return false, nil
	}
	if m.BunpyVersion != bunpyVersion || m.Entry != entryAbs || m.FlagsHash != flagsHash(opts) {
		return false, nil
	}
	for _, se := range m.Sources {
		cur, err := fileSHA256(se.Path)
		if err != nil || cur != se.SHA256 {
			return false, nil
		}
	}
	outHash, err := fileSHA256(m.Output.Path)
	if err != nil || outHash != m.Output.SHA256 {
		return false, nil
	}
	return true, nil
}

// UpdateCache saves a build manifest after a successful build.
func UpdateCache(entryAbs string, opts Options, bundle *Bundle, archivePath string, bunpyVersion string) error {
	m := cacheManifest{
		BunpyVersion: bunpyVersion,
		Entry:        entryAbs,
		FlagsHash:    flagsHash(opts),
		Sources:      make(map[string]cacheSourceEntry, len(bundle.Sources)),
	}
	for _, se := range bundle.Sources {
		h, err := fileSHA256(se.Original)
		if err != nil {
			return fmt.Errorf("cache: hash %s: %w", se.Original, err)
		}
		m.Sources[se.Bundled] = cacheSourceEntry{Path: se.Original, SHA256: h}
	}
	outHash, err := fileSHA256(archivePath)
	if err != nil {
		return fmt.Errorf("cache: hash output %s: %w", archivePath, err)
	}
	m.Output = cacheOutputInfo{Path: archivePath, SHA256: outHash}

	mPath := ManifestPath(filepath.Dir(entryAbs))
	if err := os.MkdirAll(filepath.Dir(mPath), 0o755); err != nil {
		return fmt.Errorf("cache: mkdir: %w", err)
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(mPath, data, 0o644)
}
