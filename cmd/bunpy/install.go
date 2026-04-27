package main

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/tamnd/bunpy/v1/pkg/lockfile"
	"github.com/tamnd/bunpy/v1/pkg/pypi"
	"github.com/tamnd/bunpy/v1/pkg/wheel"
)

// installSubcommand wires `bunpy install`. v0.1.5 walks bunpy.lock,
// fetches every pinned wheel through the same httpkit/cache path
// `add` uses, and installs into .bunpy/site-packages. It does not
// re-resolve: the lockfile is treated as the source of truth, which
// matches Bun's verbs and keeps CI installs deterministic.
func installSubcommand(args []string, stdout, stderr io.Writer) (int, error) {
	var (
		target   = filepath.Join(".bunpy", "site-packages")
		cacheDir string
		noVerify bool
	)
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "-h", "--help":
			return printHelp("install", stdout, stderr)
		case "--no-verify":
			noVerify = true
		case "--target":
			if i+1 >= len(args) {
				return 1, fmt.Errorf("bunpy install: --target requires a value")
			}
			i++
			target = args[i]
		case "--cache-dir":
			if i+1 >= len(args) {
				return 1, fmt.Errorf("bunpy install: --cache-dir requires a value")
			}
			i++
			cacheDir = args[i]
		default:
			if v, ok := strings.CutPrefix(a, "--target="); ok {
				target = v
				continue
			}
			if v, ok := strings.CutPrefix(a, "--cache-dir="); ok {
				cacheDir = v
				continue
			}
			return 1, fmt.Errorf("bunpy install: unknown flag %q (known: --target, --cache-dir, --no-verify, --help)", a)
		}
	}

	lock, err := lockfile.Read("bunpy.lock")
	if err != nil {
		if errors.Is(err, lockfile.ErrNotFound) {
			return 1, fmt.Errorf("bunpy install: bunpy.lock missing — run `bunpy pm lock` first")
		}
		return 1, fmt.Errorf("bunpy install: %w", err)
	}

	verify := !noVerify
	for _, p := range lock.Packages {
		f := pypi.File{Filename: p.Filename, URL: p.URL}
		body, err := fetchAddWheel(f, p.Name, cacheDir)
		if err != nil {
			return 1, fmt.Errorf("bunpy install: %s: %w", p.Name, err)
		}
		w, err := wheel.OpenReader(p.Filename, body)
		if err != nil {
			return 1, fmt.Errorf("bunpy install: %s: %w", p.Name, err)
		}
		if _, err := w.Install(target, wheel.InstallOptions{
			Installer:    "bunpy",
			VerifyHashes: &verify,
		}); err != nil {
			return 1, fmt.Errorf("bunpy install: %s: %w", p.Name, err)
		}
		fmt.Fprintf(stdout, "installed %s %s\n", p.Name, p.Version)
	}
	return 0, nil
}
