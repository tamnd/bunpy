// Package bundler collects a Python entry point and its local imports into
// a .pyz ZIP archive (PEP 441).
package bundler

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Options controls the bundler behaviour.
type Options struct {
	Outfile   string
	Outdir    string
	Minify    bool
	Target    string
	Defines   map[string]string
	Plugins   []string
	SourceMap bool
	Compile   bool
	Watch     bool
}

// Bundle holds the collected source files ready for output.
type Bundle struct {
	Entry   string            // original entry path
	Files   map[string]string // bundled-path → source (after transforms)
	Sources []SourceEntry     // ordered, for sourcemap
	Opts    Options
}

// SourceEntry records one file's mapping from bundled path to original path.
type SourceEntry struct {
	Bundled  string
	Original string
	Lines    int
}

var (
	reImport     = regexp.MustCompile(`(?m)^import\s+(\w+)`)
	reFromImport = regexp.MustCompile(`(?m)^from\s+(\.?\w+)\s+import`)
)

// scanImports returns the local module names imported by src.
// Only top-level import/from-import statements are scanned.
func scanImports(src string) []string {
	seen := map[string]bool{}
	var names []string
	for _, m := range reImport.FindAllStringSubmatch(src, -1) {
		n := m[1]
		if !seen[n] {
			seen[n] = true
			names = append(names, n)
		}
	}
	for _, m := range reFromImport.FindAllStringSubmatch(src, -1) {
		n := strings.TrimLeft(m[1], ".")
		if n != "" && !seen[n] {
			seen[n] = true
			names = append(names, n)
		}
	}
	return names
}

// collectFiles BFS-collects all reachable local .py files from entry.
// Returns bundled-path → original-absolute-path.
func collectFiles(entry string) (map[string]string, error) {
	absEntry, err := filepath.Abs(entry)
	if err != nil {
		return nil, err
	}
	root := filepath.Dir(absEntry)

	files := map[string]string{} // bundled → abs
	visited := map[string]bool{}
	queue := []string{absEntry}

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if visited[cur] {
			continue
		}
		visited[cur] = true

		src, err := os.ReadFile(cur)
		if err != nil {
			return nil, fmt.Errorf("bundler: read %s: %w", cur, err)
		}

		// Bundled path: relative to root.
		rel, err := filepath.Rel(root, cur)
		if err != nil {
			rel = filepath.Base(cur)
		}
		// Entry becomes __main__.py.
		if cur == absEntry {
			rel = "__main__.py"
		}
		files[rel] = cur

		// Follow local imports.
		for _, name := range scanImports(string(src)) {
			candidate := filepath.Join(filepath.Dir(cur), name+".py")
			if _, err := os.Stat(candidate); err == nil {
				queue = append(queue, candidate)
			}
			// Also check package: name/__init__.py
			pkgCandidate := filepath.Join(filepath.Dir(cur), name, "__init__.py")
			if _, err := os.Stat(pkgCandidate); err == nil {
				queue = append(queue, pkgCandidate)
			}
		}
	}
	return files, nil
}

// Build collects the entry point and all local imports, applies transforms,
// and returns a Bundle ready for output.
func Build(entry string, opts Options) (*Bundle, error) {
	rawFiles, err := collectFiles(entry)
	if err != nil {
		return nil, err
	}

	b := &Bundle{Entry: entry, Files: map[string]string{}, Opts: opts}

	for bundled, abs := range rawFiles {
		src, err := os.ReadFile(abs)
		if err != nil {
			return nil, err
		}
		text := string(src)

		// Apply defines.
		if len(opts.Defines) > 0 {
			text = applyDefines(text, opts.Defines)
		}

		// Apply minification.
		if opts.Minify {
			text = minifySource(text)
		}

		b.Files[bundled] = text
		b.Sources = append(b.Sources, SourceEntry{
			Bundled:  bundled,
			Original: abs,
			Lines:    strings.Count(text, "\n") + 1,
		})
	}

	return b, nil
}

// OutPath returns the resolved output file path for the bundle.
func (b *Bundle) OutPath() string {
	if b.Opts.Outfile != "" {
		return b.Opts.Outfile
	}
	stem := strings.TrimSuffix(filepath.Base(b.Entry), ".py")
	dir := b.Opts.Outdir
	if dir == "" {
		dir = "dist"
	}
	return filepath.Join(dir, stem+".pyz")
}
