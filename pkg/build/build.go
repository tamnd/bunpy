// Package build invokes PEP 517 build hooks to produce sdist and
// wheel artefacts from a Python project.
package build

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Request describes a build job.
type Request struct {
	ProjectDir string
	OutputDir  string
	Python     string // path to Python; empty = find on PATH
	Backend    string // e.g. "hatchling.build"; empty = read from pyproject
	BuildSDist bool
	BuildWheel bool
}

// Result holds paths to the built artefacts.
type Result struct {
	SDist string // absolute path to .tar.gz, or "" if not built
	Wheel string // absolute path to .whl, or "" if not built
}

// Build invokes PEP 517 hooks and returns paths to the artefacts.
func Build(req Request) (Result, error) {
	if !req.BuildSDist && !req.BuildWheel {
		return Result{}, fmt.Errorf("build: nothing to build (BuildSDist and BuildWheel both false)")
	}

	python := req.Python
	if python == "" {
		var err error
		python, err = FindPython()
		if err != nil {
			return Result{}, err
		}
	}

	backend := req.Backend
	if backend == "" {
		pyproject := filepath.Join(req.ProjectDir, "pyproject.toml")
		var err error
		backend, err = ReadBackend(pyproject)
		if err != nil {
			return Result{}, err
		}
	}

	outDir := req.OutputDir
	if outDir == "" {
		tmp, err := os.MkdirTemp("", "bunpy-build-*")
		if err != nil {
			return Result{}, fmt.Errorf("build: create temp dir: %w", err)
		}
		outDir = tmp
	}

	var res Result

	// PEP 517 hook script: import the backend and call the hook.
	// The script prints the produced filename to stdout.
	hookScript := `
import sys, importlib, os
backend_path = sys.argv[1]
hook_name   = sys.argv[2]
output_dir  = sys.argv[3]
parts = backend_path.split(":")
mod_name = parts[0]
attr = parts[1] if len(parts) > 1 else None
mod = importlib.import_module(mod_name)
target = getattr(mod, attr) if attr else mod
result = getattr(target, hook_name)(output_dir)
print(result)
`

	if req.BuildSDist {
		out, err := runHook(python, req.ProjectDir, hookScript, backend, "build_sdist", outDir)
		if err != nil {
			return Result{}, fmt.Errorf("build: build_sdist: %w", err)
		}
		res.SDist = filepath.Join(outDir, strings.TrimSpace(out))
	}

	if req.BuildWheel {
		out, err := runHook(python, req.ProjectDir, hookScript, backend, "build_wheel", outDir)
		if err != nil {
			return Result{}, fmt.Errorf("build: build_wheel: %w", err)
		}
		res.Wheel = filepath.Join(outDir, strings.TrimSpace(out))
	}

	return res, nil
}

func runHook(python, projectDir, script, backend, hook, outDir string) (string, error) {
	absOut, err := filepath.Abs(outDir)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(absOut, 0o755); err != nil {
		return "", err
	}
	cmd := exec.Command(python, "-c", script, backend, hook, absOut)
	cmd.Dir = projectDir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%s failed: %w\n%s", hook, err, stderr.String())
	}
	return strings.TrimSpace(stdout.String()), nil
}

// FindPython returns the first python3 or python binary on PATH,
// or an error if neither is found.
func FindPython() (string, error) {
	for _, name := range []string{"python3", "python"} {
		p, err := exec.LookPath(name)
		if err == nil {
			return p, nil
		}
	}
	return "", fmt.Errorf("build: python not found on PATH (install Python 3 and try again)")
}

// ReadBackend reads [build-system].build-backend from pyproject.toml.
// Returns "hatchling.build" when [build-system] is absent.
func ReadBackend(pyprojectPath string) (string, error) {
	data, err := os.ReadFile(pyprojectPath)
	if err != nil {
		return "hatchling.build", nil
	}
	// Simple scan: look for build-backend = "..." under [build-system].
	inBuildSystem := false
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if line == "[build-system]" {
			inBuildSystem = true
			continue
		}
		if strings.HasPrefix(line, "[") {
			inBuildSystem = false
		}
		if inBuildSystem && strings.HasPrefix(line, "build-backend") {
			_, after, _ := strings.Cut(line, "=")
			backend := strings.TrimSpace(after)
			backend = strings.Trim(backend, `"'`)
			if backend != "" {
				return backend, nil
			}
		}
	}
	return "hatchling.build", nil
}
