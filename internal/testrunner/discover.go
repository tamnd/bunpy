// Package testrunner discovers and runs bunpy test files.
// Test files are named test_*.py or *_test.py. Inside each file,
// functions named test_* or Test* are collected and run.
package testrunner

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// DiscoverOptions controls how test files are found.
type DiscoverOptions struct {
	// Root is the directory to walk. Defaults to ".".
	Root string
	// Patterns is an optional list of glob patterns to limit discovery.
	// Empty means discover all.
	Patterns []string
}

// DiscoverFiles returns all test file paths under root that match
// the test_*.py / *_test.py naming convention.
func DiscoverFiles(opts DiscoverOptions) ([]string, error) {
	root := opts.Root
	if root == "" {
		root = "."
	}
	var files []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			// skip hidden dirs and common non-test dirs
			name := d.Name()
			if strings.HasPrefix(name, ".") || name == "__pycache__" || name == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		if !isTestFile(d.Name()) {
			return nil
		}
		if len(opts.Patterns) > 0 && !matchesAny(path, opts.Patterns) {
			return nil
		}
		files = append(files, path)
		return nil
	})
	return files, err
}

func isTestFile(name string) bool {
	if !strings.HasSuffix(name, ".py") {
		return false
	}
	base := strings.TrimSuffix(name, ".py")
	return strings.HasPrefix(base, "test_") || strings.HasSuffix(base, "_test")
}

func matchesAny(path string, patterns []string) bool {
	for _, p := range patterns {
		if ok, _ := filepath.Match(p, filepath.Base(path)); ok {
			return true
		}
		if strings.Contains(path, p) {
			return true
		}
	}
	return false
}

// IsTestFunction reports whether a Python function name is a test.
func IsTestFunction(name string) bool {
	return strings.HasPrefix(name, "test_") || strings.HasPrefix(name, "Test")
}

// ChangedFiles returns files under root that have been modified since
// the given git commit reference. Returns all files if git is unavailable.
func ChangedFiles(root, ref string) ([]string, error) {
	out, err := gitOutput(root, "diff", "--name-only", ref)
	if err != nil {
		// fallback: discover all
		return DiscoverFiles(DiscoverOptions{Root: root})
	}
	var files []string
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if !isTestFile(filepath.Base(line)) {
			// also include test files that share a directory with the changed file
			continue
		}
		full := filepath.Join(root, line)
		if _, serr := os.Stat(full); serr == nil {
			files = append(files, full)
		}
	}
	return files, nil
}
