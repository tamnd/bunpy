package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/tamnd/bunpy/v1/pkg/build"
	"github.com/tamnd/bunpy/v1/pkg/publish"
)

func publishSubcommand(args []string, stdout, stderr io.Writer) (int, error) {
	var (
		sDistOnly    bool
		wheelOnly    bool
		dryRun       bool
		registry     string
		token        string
		manifestPath = "pyproject.toml"
	)
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "-h", "--help":
			return printHelp("publish", stdout, stderr)
		case "--sdist-only":
			sDistOnly = true
		case "--wheel-only":
			wheelOnly = true
		case "--dry-run":
			dryRun = true
		case "--registry":
			if i+1 >= len(args) {
				return 1, fmt.Errorf("bunpy publish: --registry requires a value")
			}
			i++
			registry = args[i]
		case "--token":
			if i+1 >= len(args) {
				return 1, fmt.Errorf("bunpy publish: --token requires a value")
			}
			i++
			token = args[i]
		case "--manifest":
			if i+1 >= len(args) {
				return 1, fmt.Errorf("bunpy publish: --manifest requires a value")
			}
			i++
			manifestPath = args[i]
		default:
			if v, ok := strings.CutPrefix(a, "--registry="); ok {
				registry = v
				continue
			}
			if v, ok := strings.CutPrefix(a, "--token="); ok {
				token = v
				continue
			}
			if v, ok := strings.CutPrefix(a, "--manifest="); ok {
				manifestPath = v
				continue
			}
			return 1, fmt.Errorf("bunpy publish: unknown flag %q", a)
		}
	}

	if sDistOnly && wheelOnly {
		return 1, fmt.Errorf("bunpy publish: --sdist-only and --wheel-only are mutually exclusive")
	}

	if token == "" {
		token = os.Getenv("PYPI_TOKEN")
	}
	if token == "" && !dryRun {
		return 1, fmt.Errorf("bunpy publish: no token: set PYPI_TOKEN or pass --token")
	}

	cwd, err := os.Getwd()
	if err != nil {
		return 1, fmt.Errorf("bunpy publish: %w", err)
	}

	python, err := build.FindPython()
	if err != nil {
		return 1, fmt.Errorf("bunpy publish: %w", err)
	}

	backend, err := build.ReadBackend(manifestPath)
	if err != nil {
		return 1, fmt.Errorf("bunpy publish: %w", err)
	}

	outDir, err := os.MkdirTemp("", "bunpy-publish-*")
	if err != nil {
		return 1, fmt.Errorf("bunpy publish: create temp dir: %w", err)
	}
	defer os.RemoveAll(outDir)

	req := build.Request{
		ProjectDir: cwd,
		OutputDir:  outDir,
		Python:     python,
		Backend:    backend,
		BuildSDist: !wheelOnly,
		BuildWheel: !sDistOnly,
	}
	res, err := build.Build(req)
	if err != nil {
		return 1, fmt.Errorf("bunpy publish: build: %w", err)
	}

	var files []string
	if res.SDist != "" {
		files = append(files, res.SDist)
		fmt.Fprintf(stdout, "built sdist: %s\n", res.SDist)
	}
	if res.Wheel != "" {
		files = append(files, res.Wheel)
		fmt.Fprintf(stdout, "built wheel: %s\n", res.Wheel)
	}

	if dryRun {
		fmt.Fprintln(stdout, "dry-run: skipping upload")
		return 0, nil
	}

	results, err := publish.Upload(publish.UploadRequest{
		Files:    files,
		Registry: registry,
		Token:    token,
	})
	if err != nil {
		return 1, fmt.Errorf("bunpy publish: upload: %w", err)
	}
	for _, r := range results {
		if r.URL != "" {
			fmt.Fprintf(stdout, "uploaded: %s\n", r.URL)
		}
	}
	return 0, nil
}
