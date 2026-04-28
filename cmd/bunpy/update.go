package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tamnd/bunpy/v1/internal/httpkit"
	"github.com/tamnd/bunpy/v1/pkg/lockfile"
	"github.com/tamnd/bunpy/v1/pkg/manifest"
	"github.com/tamnd/bunpy/v1/pkg/marker"
	"github.com/tamnd/bunpy/v1/pkg/pypi"
	"github.com/tamnd/bunpy/v1/pkg/resolver"
	"github.com/tamnd/bunpy/v1/pkg/uvlock"
	"github.com/tamnd/bunpy/v1/pkg/version"
	"github.com/tamnd/bunpy/v1/pkg/wheel"
)

// updateSubcommand wires `bunpy update [pkg]...`. v0.1.7 re-runs
// the resolver with Solver.Locked seeded from the existing lockfile
// (minus the packages the user named on the command line, which are
// unlocked so the resolver picks afresh). The new lockfile is
// written, and unless --no-install is set, ./.bunpy/site-packages
// is refreshed via the same install path bunpy install uses.
func updateSubcommand(args []string, stdout, stderr io.Writer) (int, error) {
	var (
		baseURL    string
		cacheDir   string
		target     = filepath.Join(".bunpy", "site-packages")
		latest     bool
		noInstall  bool
		noVerify   bool
		dev        bool
		peer       bool
		allExtras  bool
		production bool
		extras     []string
		pkgs       []string
	)
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "-h", "--help":
			return printHelp("update", stdout, stderr)
		case "--latest":
			latest = true
		case "--no-install":
			noInstall = true
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
				return 1, fmt.Errorf("bunpy update: %s requires a group name", a)
			}
			i++
			extras = append(extras, args[i])
		case "--target":
			if i+1 >= len(args) {
				return 1, fmt.Errorf("bunpy update: --target requires a value")
			}
			i++
			target = args[i]
		case "--index":
			if i+1 >= len(args) {
				return 1, fmt.Errorf("bunpy update: --index requires a value")
			}
			i++
			baseURL = args[i]
		case "--cache-dir":
			if i+1 >= len(args) {
				return 1, fmt.Errorf("bunpy update: --cache-dir requires a value")
			}
			i++
			cacheDir = args[i]
		default:
			if v, ok := strings.CutPrefix(a, "--target="); ok {
				target = v
				continue
			}
			if v, ok := strings.CutPrefix(a, "--index="); ok {
				baseURL = v
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
			if strings.HasPrefix(a, "-") {
				return 1, fmt.Errorf("bunpy update: unknown flag %q", a)
			}
			pkgs = append(pkgs, pypi.Normalize(a))
		}
	}
	if production && (dev || peer || allExtras || len(extras) > 0) {
		return 1, fmt.Errorf("bunpy update: --production cannot combine with --dev/--optional/--all-extras/--peer")
	}
	if latest && len(pkgs) == 0 {
		return 1, fmt.Errorf("bunpy update: --latest requires at least one package name")
	}

	mf, err := manifest.Load("pyproject.toml")
	if err != nil {
		return 1, fmt.Errorf("bunpy update: %w", err)
	}
	lock, err := uvlock.ReadLockfile("uv.lock")
	if err != nil && !errors.Is(err, lockfile.ErrNotFound) {
		return 1, fmt.Errorf("bunpy update: %w", err)
	}

	// A bare `bunpy update` clears every lock and re-resolves the whole
	// graph, so minor and patch upgrades can flow in. When the caller
	// names packages, only those are dropped from the lock hint;
	// everything else holds its current pin.
	locked := map[string]string{}
	if lock != nil && len(pkgs) > 0 {
		unlock := map[string]bool{}
		for _, p := range pkgs {
			unlock[p] = true
		}
		for _, p := range lock.Packages {
			if unlock[pypi.Normalize(p.Name)] {
				continue
			}
			locked[pypi.Normalize(p.Name)] = p.Version
		}
	}

	laneMap := manifestLaneMap(mf)
	if latest {
		laneMap = relaxSpecsForPackages(laneMap, pkgs)
	}
	wantHash := lockfile.HashLanes(manifestLaneMap(mf))

	client := pypi.New()
	if baseURL != "" {
		client.BaseURL = baseURL
	}
	if fix := os.Getenv("BUNPY_PYPI_FIXTURES"); fix != "" {
		client.HTTP = httpkit.FixturesFS(fix)
	}
	ctx := context.Background()
	reg := newPypiRegistry(ctx, client, wheel.HostTags(), marker.DefaultEnv(),
		func(f pypi.File) ([]byte, error) {
			body, _, err := loadWheelSource(f.URL)
			return body, err
		})

	var roots []resolver.Requirement
	seenRoot := map[string]bool{}
	for _, deps := range laneMap {
		for _, dep := range deps {
			dname, vSpec := splitNameSpec(dep)
			if dname == "" {
				return 1, fmt.Errorf("bunpy update: invalid dep %q", dep)
			}
			spec, err := version.ParseSpec(vSpec)
			if err != nil {
				return 1, fmt.Errorf("bunpy update: parse %q: %w", dep, err)
			}
			key := pypi.Normalize(dname) + "|" + dep
			if seenRoot[key] {
				continue
			}
			seenRoot[key] = true
			roots = append(roots, resolver.Requirement{Name: pypi.Normalize(dname), Spec: spec})
		}
	}
	solver := resolver.New(reg)
	solver.Locked = locked
	res, err := solver.Solve(roots)
	if err != nil {
		return 1, fmt.Errorf("bunpy update: %w", err)
	}

	previous := map[string]string{}
	if lock != nil {
		for _, p := range lock.Packages {
			previous[pypi.Normalize(p.Name)] = p.Version
		}
	}

	pinLanes, err := computePinLanes(reg, res, laneMap)
	if err != nil {
		return 1, fmt.Errorf("bunpy update: %w", err)
	}

	newLock := &lockfile.Lock{Version: lockfile.Version}
	for _, pin := range res.Pins {
		f, ok := reg.Pick(pin.Name, pin.Version)
		if !ok {
			return 1, fmt.Errorf("bunpy update: no wheel pick for %s %s", pin.Name, pin.Version)
		}
		newLock.Upsert(lockfile.Package{
			Name:     pin.Name,
			Version:  pin.Version,
			Filename: f.Filename,
			URL:      f.URL,
			Hash:     wheelSha256(f),
			Lanes:    pinLanes[pin.Name],
		})
	}
	newLock.ContentHash = wantHash
	newLock.Generated = time.Now().UTC()
	if err := uvlock.WriteLockfile("uv.lock", newLock, mf.Project.RequiresPython, uvlock.WriteOptions{}); err != nil {
		return 1, fmt.Errorf("bunpy update: %w", err)
	}

	changed := 0
	for _, p := range newLock.Packages {
		prev := previous[pypi.Normalize(p.Name)]
		if prev != "" && prev != p.Version {
			fmt.Fprintf(stdout, "%s %s -> %s\n", p.Name, prev, p.Version)
			changed++
		} else if prev == "" {
			fmt.Fprintf(stdout, "%s %s (added)\n", p.Name, p.Version)
			changed++
		}
	}
	if changed == 0 {
		fmt.Fprintln(stdout, "no changes")
	}

	if noInstall {
		return 0, nil
	}

	keep := installLaneFilter(dev, peer, allExtras, extras)
	verify := !noVerify
	for _, p := range newLock.Packages {
		if !keep(p.Lanes) {
			continue
		}
		f := pypi.File{Filename: p.Filename, URL: p.URL}
		body, err := fetchAddWheel(f, p.Name, cacheDir)
		if err != nil {
			return 1, fmt.Errorf("bunpy update: %s: %w", p.Name, err)
		}
		w, err := wheel.OpenReader(p.Filename, body)
		if err != nil {
			return 1, fmt.Errorf("bunpy update: %s: %w", p.Name, err)
		}
		if _, err := w.Install(target, wheel.InstallOptions{
			Installer:    "bunpy",
			VerifyHashes: &verify,
		}); err != nil {
			return 1, fmt.Errorf("bunpy update: %s: %w", p.Name, err)
		}
	}
	return 0, nil
}

// relaxSpecsForPackages strips the spec from any direct dep whose
// PEP 503 normalised name appears in pkgs. The resolver then picks
// the highest non-prerelease version. The original manifest is not
// mutated; only the in-memory lane map fed to the solver.
func relaxSpecsForPackages(laneMap map[string][]string, pkgs []string) map[string][]string {
	want := map[string]bool{}
	for _, p := range pkgs {
		want[p] = true
	}
	out := map[string][]string{}
	for lane, deps := range laneMap {
		next := make([]string, len(deps))
		for i, dep := range deps {
			name, _ := splitNameSpec(dep)
			if want[pypi.Normalize(name)] {
				next[i] = name
			} else {
				next[i] = dep
			}
		}
		out[lane] = next
	}
	return out
}
