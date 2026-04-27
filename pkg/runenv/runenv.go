// Package runenv creates a temporary install prefix for a single wheel,
// writes entry-point shims, and locates them for exec. It is the backend
// for bunpyx: run a package once without a persistent venv.
package runenv

import (
	"archive/zip"
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// RunEnv is a temporary install prefix for one wheel.
type RunEnv struct {
	Dir    string // absolute path to temp prefix
	Python string // path to Python executable
}

// Create makes a new temp directory and records the Python path.
func Create(python string) (*RunEnv, error) {
	dir, err := os.MkdirTemp("", "bunpyx-*")
	if err != nil {
		return nil, fmt.Errorf("runenv: create temp dir: %w", err)
	}
	for _, sub := range []string{"site-packages", "bin"} {
		if err := os.MkdirAll(filepath.Join(dir, sub), 0o755); err != nil {
			_ = os.RemoveAll(dir)
			return nil, fmt.Errorf("runenv: mkdir %s: %w", sub, err)
		}
	}
	return &RunEnv{Dir: dir, Python: python}, nil
}

// Install unpacks a wheel file into e.Dir/site-packages/ and writes
// entry-point shims into e.Dir/bin/.
func (e *RunEnv) Install(wheelPath string) error {
	zr, err := zip.OpenReader(wheelPath)
	if err != nil {
		return fmt.Errorf("runenv: open wheel %s: %w", wheelPath, err)
	}
	defer zr.Close()

	sitePackages := filepath.Join(e.Dir, "site-packages")
	var entryPointsBody []byte

	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		name := f.Name
		// reject unsafe paths
		if strings.Contains(name, "..") || filepath.IsAbs(name) {
			return fmt.Errorf("runenv: unsafe path in wheel: %s", name)
		}

		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("runenv: open zip entry %s: %w", name, err)
		}
		body, err := io.ReadAll(rc)
		_ = rc.Close()
		if err != nil {
			return fmt.Errorf("runenv: read zip entry %s: %w", name, err)
		}

		dest := filepath.Join(sitePackages, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return fmt.Errorf("runenv: mkdir for %s: %w", name, err)
		}
		mode := os.FileMode(0o644)
		if err := os.WriteFile(dest, body, mode); err != nil {
			return fmt.Errorf("runenv: write %s: %w", name, err)
		}

		if strings.HasSuffix(name, ".dist-info/entry_points.txt") {
			entryPointsBody = body
		}
	}

	if len(entryPointsBody) > 0 {
		scripts, err := parseConsoleScripts(entryPointsBody)
		if err != nil {
			return fmt.Errorf("runenv: parse entry_points.txt: %w", err)
		}
		for name, target := range scripts {
			if err := e.writeShim(name, target); err != nil {
				return err
			}
		}
	}
	return nil
}

// EntryPoint returns the path to the shim for the named console script,
// or ("", false) if not registered.
func (e *RunEnv) EntryPoint(name string) (string, bool) {
	shimName := name
	if runtime.GOOS == "windows" {
		shimName = name + ".cmd"
	}
	p := filepath.Join(e.Dir, "bin", shimName)
	if _, err := os.Stat(p); err == nil {
		return p, true
	}
	return "", false
}

// Cleanup removes the temp prefix entirely.
func (e *RunEnv) Cleanup() error {
	return os.RemoveAll(e.Dir)
}

// writeShim creates a launcher script for a console_scripts entry.
// target is "module:attr" or just "module".
func (e *RunEnv) writeShim(name, target string) error {
	module, attr, _ := strings.Cut(target, ":")

	var body []byte
	if runtime.GOOS == "windows" {
		py := shimPythonPath(e.Python)
		script := filepath.Join(e.Dir, "bin", name+".py")
		if attr != "" {
			scriptBody := fmt.Sprintf(
				"import sys\nfrom %s import %s\nsys.exit(%s())\n",
				module, attr, attr)
			if err := os.WriteFile(script, []byte(scriptBody), 0o644); err != nil {
				return fmt.Errorf("runenv: write shim script %s: %w", name, err)
			}
		} else {
			scriptBody := fmt.Sprintf(
				"import sys, runpy\nsys.exit(runpy.run_module(%q, run_name='__main__'))\n",
				module)
			if err := os.WriteFile(script, []byte(scriptBody), 0o644); err != nil {
				return fmt.Errorf("runenv: write shim script %s: %w", name, err)
			}
		}
		body = []byte(fmt.Sprintf("@\"%s\" \"%s\" %%*\r\n", py, script))
		dest := filepath.Join(e.Dir, "bin", name+".cmd")
		return os.WriteFile(dest, body, 0o644)
	}

	if attr != "" {
		body = []byte(fmt.Sprintf(
			"#!/usr/bin/env python3\nimport sys\nfrom %s import %s\nsys.exit(%s())\n",
			module, attr, attr))
	} else {
		body = []byte(fmt.Sprintf(
			"#!/usr/bin/env python3\nimport sys, runpy\nsys.exit(runpy.run_module(%q, run_name='__main__'))\n",
			module))
	}
	dest := filepath.Join(e.Dir, "bin", name)
	if err := os.WriteFile(dest, body, 0o755); err != nil {
		return fmt.Errorf("runenv: write shim %s: %w", name, err)
	}
	return nil
}

func shimPythonPath(python string) string {
	if python != "" {
		return python
	}
	return "python"
}

// parseConsoleScripts reads the [console_scripts] section of entry_points.txt
// and returns a map of script name -> "module:attr" target.
func parseConsoleScripts(body []byte) (map[string]string, error) {
	result := map[string]string{}
	inConsole := false
	sc := bufio.NewScanner(bytes.NewReader(body))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || line[0] == '#' {
			continue
		}
		if line[0] == '[' {
			inConsole = strings.TrimSuffix(strings.TrimPrefix(line, "["), "]") == "console_scripts"
			continue
		}
		if !inConsole {
			continue
		}
		name, target, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		result[strings.TrimSpace(name)] = strings.TrimSpace(target)
	}
	return result, sc.Err()
}

