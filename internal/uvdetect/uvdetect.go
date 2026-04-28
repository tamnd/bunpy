// Package uvdetect locates the uv binary on PATH and reports its version.
package uvdetect

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// Find returns the absolute path to the uv binary, or an error if not found.
func Find() (string, error) {
	p, err := exec.LookPath("uv")
	if err != nil {
		return "", fmt.Errorf("uvdetect: uv not found on PATH (install from https://github.com/astral-sh/uv)")
	}
	return p, nil
}

// Version returns the version string reported by `uv --version`, e.g. "0.5.1".
// Returns an error if uv is not on PATH or the invocation fails.
func Version() (string, error) {
	uv, err := Find()
	if err != nil {
		return "", err
	}
	out, err := exec.Command(uv, "--version").Output()
	if err != nil {
		return "", fmt.Errorf("uvdetect: uv --version: %w", err)
	}
	// output is like "uv 0.5.1\n"
	s := strings.TrimSpace(string(bytes.TrimPrefix(out, []byte("uv "))))
	return s, nil
}
