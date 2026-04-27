package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/tamnd/bunpy/v1/pkg/editable"
	"github.com/tamnd/bunpy/v1/pkg/lockfile"
	"github.com/tamnd/bunpy/v1/pkg/manifest"
	"github.com/tamnd/bunpy/v1/pkg/patches"
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
		wsRoot     string
		noVerify   bool
		noPatches  bool
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
		case "--no-patches":
			noPatches = true
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
		case "--workspace":
			if i+1 >= len(args) {
				return 1, fmt.Errorf("bunpy install: --workspace requires a value")
			}
			i++
			wsRoot = args[i]
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
			if v, ok := strings.CutPrefix(a, "--workspace="); ok {
				wsRoot = v
				continue
			}
			return 1, fmt.Errorf("bunpy install: unknown flag %q", a)
		}
	}
	if production && (dev || peer || allExtras || len(extras) > 0) {
		return 1, fmt.Errorf("bunpy install: --production cannot combine with --dev/--optional/--all-extras/--peer")
	}

	// Auto-detect workspace root when --workspace is not set explicitly.
	lockPath := "bunpy.lock"
	manifestPath := "pyproject.toml"
	if wsRoot == "" {
		if cwd, err := os.Getwd(); err == nil {
			if found, err := findWorkspaceRoot(cwd); err == nil {
				wsRoot = found
			}
		}
	}
	if wsRoot != "" {
		lockPath = filepath.Join(wsRoot, "bunpy.lock")
		manifestPath = filepath.Join(wsRoot, "pyproject.toml")
	}
	_ = manifestPath

	var patchEntries []patches.Entry
	if !noPatches {
		if mf, err := manifest.LoadOpts("pyproject.toml", manifest.LoadOptions{}); err == nil {
			if pe, err := patches.Read(mf); err == nil {
				patchEntries = pe
			}
		}
	}

	lock, err := lockfile.Read(lockPath)
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
		if isLinkedPackage(target, p.Name, p.Version) {
			fmt.Fprintf(stdout, "kept linked %s %s\n", p.Name, p.Version)
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
		patched, err := applyRegisteredPatch(target, p.Name, p.Version, patchEntries)
		if err != nil {
			return 1, fmt.Errorf("bunpy install: %s: %w", p.Name, err)
		}
		if patched {
			fmt.Fprintf(stdout, "patched %s %s\n", p.Name, p.Version)
		} else {
			fmt.Fprintf(stdout, "installed %s %s\n", p.Name, p.Version)
		}
	}
	if skipped > 0 {
		fmt.Fprintf(stdout, "skipped %d package%s outside the selected lanes\n", skipped, pluralS(skipped))
	}
	return 0, nil
}

// isLinkedPackage reports whether a previously-laid-down editable
// install owns the dist-info for (name, version) under target.
// `bunpy install` skips these so a `bunpy link <pkg>` survives
// unrelated `install` runs. Callers can re-link or run
// `bunpy unlink <pkg>` to flip the install back to the pinned wheel.
func isLinkedPackage(target, name, version string) bool {
	abs, err := filepath.Abs(target)
	if err != nil {
		return false
	}
	di := filepath.Join(abs, pypi.Normalize(name)+"-"+version+".dist-info", "INSTALLER")
	body, err := os.ReadFile(di)
	if err != nil {
		// Try the verbatim name too: dist-info naming is not always
		// PEP 503-normalised by the producing tool.
		di = filepath.Join(abs, name+"-"+version+".dist-info", "INSTALLER")
		body, err = os.ReadFile(di)
		if err != nil {
			return false
		}
	}
	return strings.TrimSpace(string(body)) == editable.InstallerTag
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
