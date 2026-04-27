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
		target     = filepath.Join(".bunpy", "site-packages")
		cacheDir   string
		noVerify   bool
		dev        bool
		peer       bool
		allExtras  bool
		production bool
		extras     []string
	)
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "-h", "--help":
			return printHelp("install", stdout, stderr)
		case "--no-verify":
			noVerify = true
		case "-D", "--dev":
			dev = true
		case "-P", "--peer":
			peer = true
		case "--all-extras":
			allExtras = true
		case "--production":
			production = true
		case "-O", "--optional":
			if i+1 >= len(args) {
				return 1, fmt.Errorf("bunpy install: %s requires a group name", a)
			}
			i++
			extras = append(extras, args[i])
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
			if v, ok := strings.CutPrefix(a, "--optional="); ok {
				extras = append(extras, v)
				continue
			}
			return 1, fmt.Errorf("bunpy install: unknown flag %q", a)
		}
	}
	if production && (dev || peer || allExtras || len(extras) > 0) {
		return 1, fmt.Errorf("bunpy install: --production cannot combine with --dev/--optional/--all-extras/--peer")
	}

	lock, err := lockfile.Read("bunpy.lock")
	if err != nil {
		if errors.Is(err, lockfile.ErrNotFound) {
			return 1, fmt.Errorf("bunpy install: bunpy.lock missing - run `bunpy pm lock` first")
		}
		return 1, fmt.Errorf("bunpy install: %w", err)
	}

	keep := installLaneFilter(dev, peer, allExtras, extras)
	verify := !noVerify
	skipped := 0
	for _, p := range lock.Packages {
		if !keep(p.Lanes) {
			skipped++
			continue
		}
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
	if skipped > 0 {
		fmt.Fprintf(stdout, "skipped %d package%s outside the selected lanes\n", skipped, pluralS(skipped))
	}
	return 0, nil
}

// installLaneFilter returns a predicate that decides whether a
// lockfile row's lane set overlaps the user-selected lanes. The
// default (no flags) keeps main only.
func installLaneFilter(dev, peer, allExtras bool, extras []string) func([]string) bool {
	enabled := map[string]bool{lockfile.LaneMain: true}
	if dev {
		enabled[lockfile.LaneDev] = true
	}
	if peer {
		enabled[lockfile.LanePeer] = true
	}
	for _, g := range extras {
		enabled[lockfile.OptionalLane(g)] = true
	}
	return func(lanes []string) bool {
		if len(lanes) == 0 {
			return enabled[lockfile.LaneMain]
		}
		for _, l := range lanes {
			if enabled[l] {
				return true
			}
			if allExtras && strings.HasPrefix(l, "optional:") {
				return true
			}
		}
		return false
	}
}
