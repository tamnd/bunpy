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
	"github.com/tamnd/bunpy/v1/pkg/version"
	"github.com/tamnd/bunpy/v1/pkg/wheel"
)

// removeSubcommand wires `bunpy remove <pkg>...`. v0.1.8 is the
// inverse of `bunpy add`: it deletes the named packages from
// pyproject.toml (every lane unless a lane flag narrows it),
// re-runs the resolver against the new lane map with surviving
// pins held via Solver.Locked, rewrites bunpy.lock, and uninstalls
// the dropped pins from ./.bunpy/site-packages unless --no-install.
func removeSubcommand(args []string, stdout, stderr io.Writer) (int, error) {
	var (
		baseURL   string
		target    = filepath.Join(".bunpy", "site-packages")
		noInstall bool
		noWrite   bool
		dev       bool
		group     string
		optional  string
		peer      bool
		pkgs      []string
	)
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "-h", "--help":
			return printHelp("remove", stdout, stderr)
		case "--no-install":
			noInstall = true
		case "--no-write":
			noWrite = true
		case "-D", "--dev":
			dev = true
		case "-P", "--peer":
			peer = true
		case "--group":
			if i+1 >= len(args) {
				return 1, fmt.Errorf("bunpy remove: --group requires a value")
			}
			i++
			group = args[i]
		case "-O", "--optional":
			if i+1 >= len(args) {
				return 1, fmt.Errorf("bunpy remove: %s requires a group name", a)
			}
			i++
			optional = args[i]
		case "--target":
			if i+1 >= len(args) {
				return 1, fmt.Errorf("bunpy remove: --target requires a value")
			}
			i++
			target = args[i]
		case "--index":
			if i+1 >= len(args) {
				return 1, fmt.Errorf("bunpy remove: --index requires a value")
			}
			i++
			baseURL = args[i]
		default:
			if v, ok := strings.CutPrefix(a, "--target="); ok {
				target = v
				continue
			}
			if v, ok := strings.CutPrefix(a, "--index="); ok {
				baseURL = v
				continue
			}
			if v, ok := strings.CutPrefix(a, "--optional="); ok {
				optional = v
				continue
			}
			if v, ok := strings.CutPrefix(a, "--group="); ok {
				group = v
				continue
			}
			if strings.HasPrefix(a, "-") {
				return 1, fmt.Errorf("bunpy remove: unknown flag %q", a)
			}
			pkgs = append(pkgs, a)
		}
	}
	if len(pkgs) == 0 {
		return 1, fmt.Errorf("bunpy remove: at least one package name is required")
	}

	laneFlags := 0
	if dev || group != "" {
		laneFlags++
	}
	if optional != "" {
		laneFlags++
	}
	if peer {
		laneFlags++
	}
	if laneFlags > 1 {
		return 1, fmt.Errorf("bunpy remove: -D/-O/-P are mutually exclusive")
	}
	if group != "" && !dev {
		return 1, fmt.Errorf("bunpy remove: --group requires -D/--dev")
	}

	mf, err := manifest.Load("pyproject.toml")
	if err != nil {
		return 1, fmt.Errorf("bunpy remove: %w", err)
	}

	totalRemoved := 0
	srcOut := mf.Source
	for _, name := range pkgs {
		nm := pypi.Normalize(name)
		var (
			out []byte
			n   int
			err error
		)
		switch {
		case dev && group == "":
			out, n, err = mf.RemoveGroupDependency("dev", nm)
		case dev && group != "":
			out, n, err = mf.RemoveGroupDependency(group, nm)
		case optional != "":
			out, n, err = mf.RemoveOptionalDependency(optional, nm)
		case peer:
			out, n, err = mf.RemovePeerDependency(nm)
		default:
			out, n, err = mf.RemoveDependencyAllLanes(nm)
		}
		if err != nil {
			return 1, fmt.Errorf("bunpy remove: %s: %w", name, err)
		}
		if n > 0 {
			next, perr := manifest.Parse(out)
			if perr != nil {
				return 1, fmt.Errorf("bunpy remove: re-parse: %w", perr)
			}
			mf = next
			srcOut = out
			totalRemoved += n
		}
	}

	if !noWrite && totalRemoved > 0 {
		if err := os.WriteFile("pyproject.toml", srcOut, 0o644); err != nil {
			return 1, fmt.Errorf("bunpy remove: write pyproject.toml: %w", err)
		}
	}

	// Re-resolve against the post-remove lane map. The solver runs even
	// when totalRemoved == 0 so a stale lockfile still gets rewritten
	// to match the manifest content-hash; this matches `bunpy update`'s
	// idempotent shape.
	lock, err := lockfile.Read("bunpy.lock")
	if err != nil && !errors.Is(err, lockfile.ErrNotFound) {
		return 1, fmt.Errorf("bunpy remove: %w", err)
	}

	removed := map[string]bool{}
	for _, p := range pkgs {
		removed[pypi.Normalize(p)] = true
	}
	locked := map[string]string{}
	if lock != nil {
		for _, p := range lock.Packages {
			if removed[pypi.Normalize(p.Name)] {
				continue
			}
			locked[pypi.Normalize(p.Name)] = p.Version
		}
	}

	laneMap := manifestLaneMap(mf)
	wantHash := lockfile.HashLanes(laneMap)

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
				return 1, fmt.Errorf("bunpy remove: invalid dep %q", dep)
			}
			spec, err := version.ParseSpec(vSpec)
			if err != nil {
				return 1, fmt.Errorf("bunpy remove: parse %q: %w", dep, err)
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
		return 1, fmt.Errorf("bunpy remove: %w", err)
	}

	pinLanes, err := computePinLanes(reg, res, laneMap)
	if err != nil {
		return 1, fmt.Errorf("bunpy remove: %w", err)
	}

	newLock := &lockfile.Lock{Version: lockfile.Version}
	stillPinned := map[string]bool{}
	for _, pin := range res.Pins {
		f, ok := reg.Pick(pin.Name, pin.Version)
		if !ok {
			return 1, fmt.Errorf("bunpy remove: no wheel pick for %s %s", pin.Name, pin.Version)
		}
		newLock.Upsert(lockfile.Package{
			Name:     pin.Name,
			Version:  pin.Version,
			Filename: f.Filename,
			URL:      f.URL,
			Hash:     wheelSha256(f),
			Lanes:    pinLanes[pin.Name],
		})
		stillPinned[pypi.Normalize(pin.Name)] = true
	}
	newLock.ContentHash = wantHash
	newLock.Generated = time.Now().UTC()
	if err := newLock.WriteFile("bunpy.lock"); err != nil {
		return 1, fmt.Errorf("bunpy remove: %w", err)
	}

	// Diff old vs new lockfile to find pins that fell off (the named
	// packages and any transitive that no longer has a root).
	var dropped []lockfile.Package
	if lock != nil {
		for _, p := range lock.Packages {
			if !stillPinned[pypi.Normalize(p.Name)] {
				dropped = append(dropped, p)
			}
		}
	}

	fmt.Fprintf(stdout, "removed %d package%s\n", totalRemoved, pluralS(totalRemoved))
	for _, p := range dropped {
		fmt.Fprintf(stdout, "  - %s %s\n", p.Name, p.Version)
	}

	if noInstall {
		return 0, nil
	}
	for _, p := range dropped {
		if err := uninstallPin(target, p); err != nil {
			fmt.Fprintf(stderr, "bunpy remove: %s: %v\n", p.Name, err)
		}
	}
	return 0, nil
}

// uninstallPin removes a previously-installed pin from the target
// site-packages dir. Walks RECORD when present so script and data
// files get cleaned up; falls back to a directory-name-based purge
// (purelib only) when RECORD is missing or unreadable.
func uninstallPin(target string, p lockfile.Package) error {
	abs, err := filepath.Abs(target)
	if err != nil {
		return err
	}
	distInfoPattern := filepath.Join(abs, pypi.Normalize(p.Name)+"-"+p.Version+".dist-info")
	matches, _ := filepath.Glob(distInfoPattern)
	if len(matches) == 0 {
		// dist-info names are not always normalised; try the on-wheel
		// project name verbatim too.
		alt := filepath.Join(abs, p.Name+"-"+p.Version+".dist-info")
		if _, err := os.Stat(alt); err == nil {
			matches = append(matches, alt)
		}
	}
	for _, di := range matches {
		record := filepath.Join(di, "RECORD")
		if data, err := os.ReadFile(record); err == nil {
			for line := range strings.SplitSeq(string(data), "\n") {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				rel := strings.SplitN(line, ",", 2)[0]
				if rel == "" {
					continue
				}
				path := filepath.Join(abs, rel)
				cleaned := filepath.Clean(path)
				if !strings.HasPrefix(cleaned, abs+string(filepath.Separator)) && cleaned != abs {
					continue
				}
				_ = os.Remove(cleaned)
			}
		}
		_ = os.RemoveAll(di)
	}
	// Best-effort: drop the top-level package directory if it still
	// exists (purelib, no RECORD). PEP 503 normalisation matters for
	// the dist-info; the on-disk import name is whatever the wheel
	// shipped, so we use the lockfile name verbatim here.
	pkgDir := filepath.Join(abs, p.Name)
	if info, err := os.Stat(pkgDir); err == nil && info.IsDir() {
		_ = os.RemoveAll(pkgDir)
	}
	return nil
}
