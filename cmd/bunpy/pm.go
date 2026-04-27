package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/tamnd/bunpy/v1/internal/httpkit"
	"github.com/tamnd/bunpy/v1/pkg/cache"
	"github.com/tamnd/bunpy/v1/pkg/manifest"
	"github.com/tamnd/bunpy/v1/pkg/pypi"
)

// pmSubcommand routes the `bunpy pm <verb>` plumbing tree. v0.1.0
// only wires `config`; later rungs grow `info`, `install-wheel`, and
// the rest under the same umbrella.
func pmSubcommand(args []string, stdout, stderr io.Writer) (int, error) {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: bunpy pm <verb>")
		return 1, fmt.Errorf("bunpy pm requires a verb (known: config)")
	}
	switch args[0] {
	case "config":
		return pmConfig(args[1:], stdout, stderr)
	case "info":
		return pmInfo(args[1:], stdout, stderr)
	case "-h", "--help", "help":
		return printHelp("pm", stdout, stderr)
	}
	return 1, fmt.Errorf("bunpy pm: unknown verb %q (known: config, info, --help)", args[0])
}

// pmConfig parses pyproject.toml and prints the structured manifest
// as indented JSON. Default path is ./pyproject.toml; an optional
// positional argument overrides.
func pmConfig(args []string, stdout, stderr io.Writer) (int, error) {
	path := "pyproject.toml"
	for _, a := range args {
		switch a {
		case "-h", "--help":
			return printHelp("pm-config", stdout, stderr)
		default:
			if strings.HasPrefix(a, "-") {
				return 1, fmt.Errorf("bunpy pm config: unknown flag %q (known: --help)", a)
			}
			path = a
		}
	}
	m, err := manifest.Load(path)
	if err != nil {
		return 1, err
	}
	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(m); err != nil {
		return 1, fmt.Errorf("bunpy pm config: %w", err)
	}
	return 0, nil
}

// pmInfo fetches a project's PEP 691 simple-index page and prints
// the parsed view as JSON. Defaults to https://pypi.org/simple/
// and the user's bunpy cache; flags override.
func pmInfo(args []string, stdout, stderr io.Writer) (int, error) {
	var (
		pkgName  string
		baseURL  string
		cacheDir string
		noCache  bool
	)
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "-h", "--help":
			return printHelp("pm-info", stdout, stderr)
		case "--no-cache":
			noCache = true
		case "--index":
			if i+1 >= len(args) {
				return 1, fmt.Errorf("bunpy pm info: --index requires a value")
			}
			i++
			baseURL = args[i]
		case "--cache-dir":
			if i+1 >= len(args) {
				return 1, fmt.Errorf("bunpy pm info: --cache-dir requires a value")
			}
			i++
			cacheDir = args[i]
		default:
			if strings.HasPrefix(a, "--index=") {
				baseURL = strings.TrimPrefix(a, "--index=")
				continue
			}
			if strings.HasPrefix(a, "--cache-dir=") {
				cacheDir = strings.TrimPrefix(a, "--cache-dir=")
				continue
			}
			if strings.HasPrefix(a, "-") {
				return 1, fmt.Errorf("bunpy pm info: unknown flag %q (known: --index, --cache-dir, --no-cache, --help)", a)
			}
			if pkgName != "" {
				return 1, fmt.Errorf("bunpy pm info: too many positional arguments (%q after %q)", a, pkgName)
			}
			pkgName = a
		}
	}
	if pkgName == "" {
		return 1, fmt.Errorf("usage: bunpy pm info <package>")
	}

	client := pypi.New()
	if baseURL != "" {
		client.BaseURL = baseURL
	}
	if fix := os.Getenv("BUNPY_PYPI_FIXTURES"); fix != "" {
		client.HTTP = httpkit.FixturesFS(fix)
	}
	if !noCache {
		dir := cacheDir
		if dir == "" {
			dir = cache.DefaultDir() + "/index"
		}
		if idx, err := cache.NewIndex(dir); err == nil {
			client.Cache = idx
		}
	}

	proj, err := client.Get(context.Background(), pkgName)
	if err != nil {
		var nf *pypi.NotFoundError
		if errors.As(err, &nf) {
			fmt.Fprintln(stderr, "bunpy pm info:", err)
			return 1, err
		}
		return 1, fmt.Errorf("bunpy pm info: %w", err)
	}
	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(proj); err != nil {
		return 1, fmt.Errorf("bunpy pm info: %w", err)
	}
	return 0, nil
}
