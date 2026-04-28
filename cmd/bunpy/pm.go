package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"time"

	"github.com/tamnd/bunpy/v1/internal/httpkit"
	"github.com/tamnd/bunpy/v1/internal/uvdetect"
	"github.com/tamnd/bunpy/v1/pkg/cache"
	"github.com/tamnd/bunpy/v1/pkg/lockfile"
	"github.com/tamnd/bunpy/v1/pkg/manifest"
	"github.com/tamnd/bunpy/v1/pkg/marker"
	"github.com/tamnd/bunpy/v1/pkg/pypi"
	"github.com/tamnd/bunpy/v1/pkg/resolver"
	"github.com/tamnd/bunpy/v1/pkg/uvlock"
	"github.com/tamnd/bunpy/v1/pkg/version"
	"github.com/tamnd/bunpy/v1/pkg/wheel"
)

// pmSubcommand routes the `bunpy pm <verb>` plumbing tree. v0.1.2
// wires `config`, `info`, and `install-wheel`. Later rungs grow
// `add`, `install`, `lock`, and the rest under the same umbrella.
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
	case "install-wheel":
		return pmInstallWheel(args[1:], stdout, stderr)
	case "lock":
		return pmLock(args[1:], stdout, stderr)
	case "-h", "--help", "help":
		return printHelp("pm", stdout, stderr)
	}
	return 1, fmt.Errorf("bunpy pm: unknown verb %q (known: config, info, install-wheel, lock, --help)", args[0])
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

// pmInstallWheel installs one wheel from a local path or https URL
// into ./.bunpy/site-packages (or --target). The fetch path goes
// through httpkit and is offline-substitutable via
// BUNPY_PYPI_FIXTURES so CI smoke tests stay offline.
func pmInstallWheel(args []string, stdout, stderr io.Writer) (int, error) {
	var (
		source    string
		target    = filepath.Join(".bunpy", "site-packages")
		installer = "bunpy"
		noVerify  bool
	)
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "-h", "--help":
			return printHelp("pm-install-wheel", stdout, stderr)
		case "--no-verify":
			noVerify = true
		case "--target":
			if i+1 >= len(args) {
				return 1, fmt.Errorf("bunpy pm install-wheel: --target requires a value")
			}
			i++
			target = args[i]
		case "--installer":
			if i+1 >= len(args) {
				return 1, fmt.Errorf("bunpy pm install-wheel: --installer requires a value")
			}
			i++
			installer = args[i]
		default:
			if strings.HasPrefix(a, "--target=") {
				target = strings.TrimPrefix(a, "--target=")
				continue
			}
			if strings.HasPrefix(a, "--installer=") {
				installer = strings.TrimPrefix(a, "--installer=")
				continue
			}
			if strings.HasPrefix(a, "-") {
				return 1, fmt.Errorf("bunpy pm install-wheel: unknown flag %q (known: --target, --installer, --no-verify, --help)", a)
			}
			if source != "" {
				return 1, fmt.Errorf("bunpy pm install-wheel: too many positional arguments (%q after %q)", a, source)
			}
			source = a
		}
	}
	if source == "" {
		return 1, fmt.Errorf("usage: bunpy pm install-wheel <url|path>")
	}

	body, filename, err := loadWheelSource(source)
	if err != nil {
		return 1, fmt.Errorf("bunpy pm install-wheel: %w", err)
	}
	w, err := wheel.OpenReader(filename, body)
	if err != nil {
		return 1, fmt.Errorf("bunpy pm install-wheel: %w", err)
	}
	verify := !noVerify
	created, err := w.Install(target, wheel.InstallOptions{
		Installer:    installer,
		VerifyHashes: &verify,
	})
	if err != nil {
		return 1, fmt.Errorf("bunpy pm install-wheel: %w", err)
	}
	for _, p := range created {
		fmt.Fprintln(stdout, p)
	}
	return 0, nil
}

// pmLockWithUV delegates `pm lock` to the real uv binary.
// For --check it runs `uv lock --check`; otherwise `uv lock`.
func pmLockWithUV(check bool, stdout, stderr io.Writer) (int, error) {
	uvBin, err := uvdetect.Find()
	if err != nil {
		return 1, err
	}
	uvArgs := []string{"lock"}
	if check {
		uvArgs = append(uvArgs, "--check")
	}
	cmd := exec.Command(uvBin, uvArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			return exit.ExitCode(), nil
		}
		return 1, fmt.Errorf("bunpy pm lock --backend=uv: %w", err)
	}
	return 0, nil
}

// pmLock regenerates uv.lock from pyproject.toml without installing.
// With --check, exits non-zero on drift: missing lockfile, content-hash
// mismatch, or a pyproject dep with no lockfile entry. v0.1.5 walks
// the resolver so transitive deps land in the lockfile alongside the
// direct ones. With --backend=uv, delegates to the real uv binary.
func pmLock(args []string, stdout, stderr io.Writer) (int, error) {
	var (
		check          bool
		frozen         bool
		offline        bool
		upgrade        bool
		upgradePkgs    []string
		baseURL        string
		cacheDir       string
		backend        string
	)
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "-h", "--help":
			return printHelp("pm-lock", stdout, stderr)
		case "--check":
			check = true
		case "--frozen":
			frozen = true
		case "--offline":
			offline = true
		case "--upgrade", "-U":
			upgrade = true
		case "--upgrade-package", "-P":
			if i+1 >= len(args) {
				return 1, fmt.Errorf("bunpy pm lock: --upgrade-package requires a value")
			}
			i++
			upgradePkgs = append(upgradePkgs, args[i])
		case "--index":
			if i+1 >= len(args) {
				return 1, fmt.Errorf("bunpy pm lock: --index requires a value")
			}
			i++
			baseURL = args[i]
		case "--cache-dir":
			if i+1 >= len(args) {
				return 1, fmt.Errorf("bunpy pm lock: --cache-dir requires a value")
			}
			i++
			cacheDir = args[i]
		case "--backend":
			if i+1 >= len(args) {
				return 1, fmt.Errorf("bunpy pm lock: --backend requires a value")
			}
			i++
			backend = args[i]
		default:
			if strings.HasPrefix(a, "--upgrade-package=") {
				upgradePkgs = append(upgradePkgs, strings.TrimPrefix(a, "--upgrade-package="))
				continue
			}
			if strings.HasPrefix(a, "--index=") {
				baseURL = strings.TrimPrefix(a, "--index=")
				continue
			}
			if strings.HasPrefix(a, "--cache-dir=") {
				cacheDir = strings.TrimPrefix(a, "--cache-dir=")
				continue
			}
			if strings.HasPrefix(a, "--backend=") {
				backend = strings.TrimPrefix(a, "--backend=")
				continue
			}
			return 1, fmt.Errorf("bunpy pm lock: unknown flag %q (known: --check, --frozen, --offline, --upgrade, --upgrade-package, --index, --cache-dir, --backend, --help)", a)
		}
	}

	if backend == "uv" {
		return pmLockWithUV(check, stdout, stderr)
	}
	mf, err := manifest.Load("pyproject.toml")
	if err != nil {
		return 1, fmt.Errorf("bunpy pm lock: %w", err)
	}
	laneMap := manifestLaneMap(mf)
	wantHash := lockfile.HashLanes(laneMap)

	if check {
		existing, err := uvlock.ReadLockfile("uv.lock")
		if err != nil {
			if errors.Is(err, uvlock.ErrNotFound) {
				fmt.Fprintln(stderr, "bunpy pm lock --check: uv.lock missing")
				return 1, fmt.Errorf("uv.lock missing")
			}
			return 1, fmt.Errorf("bunpy pm lock: %w", err)
		}
		if existing.ContentHash != wantHash {
			fmt.Fprintf(stderr, "bunpy pm lock --check: content-hash drift (lock=%s want=%s)\n", existing.ContentHash, wantHash)
			return 1, fmt.Errorf("content-hash drift")
		}
		for _, dep := range mf.Project.Dependencies {
			name, _ := splitNameSpec(dep)
			if name == "" {
				continue
			}
			norm := lockfile.Normalize(name)
			found := false
			for _, p := range existing.Packages {
				if lockfile.Normalize(p.Name) == norm {
					found = true
					break
				}
			}
			if !found {
				fmt.Fprintf(stderr, "bunpy pm lock --check: pyproject has %q not in lockfile\n", name)
				return 1, fmt.Errorf("missing lockfile entry: %s", name)
			}
		}
		return 0, nil
	}

	client := pypi.New()
	if baseURL != "" {
		client.BaseURL = baseURL
	}
	if fix := os.Getenv("BUNPY_PYPI_FIXTURES"); fix != "" {
		client.HTTP = httpkit.FixturesFS(fix)
	} else {
		// Wire disk index cache so repeated pm lock runs skip re-fetching
		// unchanged package JSON (RC-1: was silently discarding cacheDir).
		dir := cacheDir
		if dir == "" {
			dir = cache.DefaultDir() + "/index"
		}
		if idx, err := cache.NewIndex(dir); err == nil {
			if offline {
				// --offline: treat every cached page as infinitely fresh;
				// never attempt a network round-trip.
				idx.Freshness = 1<<63 - 1
			} else {
				// RC-7: skip HTTP round-trips for index pages stored within
				// the last hour. pm lock still revalidates hourly to catch
				// new releases; users can bust the cache with --no-cache.
				idx.Freshness = time.Hour
			}
			client.Cache = idx
		}
	}

	ctx := context.Background()
	reg := newPypiRegistry(ctx, client, wheel.HostTags(), marker.DefaultEnv(),
		func(f pypi.File) ([]byte, error) {
			body, _, err := loadWheelSource(f.URL)
			return body, err
		})

	// RC-5: fast no-op path — if the manifest hash matches the existing
	// lock's content-hash, the graph hasn't changed; skip re-resolution.
	// Seed the solver with existing pins so unchanged packages are not
	// re-fetched even when the manifest did change slightly.
	// --upgrade clears all pins; --upgrade-package clears named ones only.
	// --frozen reads the lock and fails if re-resolution would differ.
	solverLocked := map[string]string{}
	var existingLock *lockfile.Lock
	// G-8: preserve non-registry (git/path/editable) packages from prior lock.
	extraPkgs, _ := uvlock.ReadNonRegistryPackages("uv.lock")
	if existing, err := uvlock.ReadLockfile("uv.lock"); err == nil {
		existingLock = existing
		if !upgrade && !frozen && existing.ContentHash == wantHash {
			fmt.Fprintf(stdout, "uv.lock up-to-date (%d package%s)\n", len(existing.Packages), pluralS(len(existing.Packages)))
			return 0, nil
		}
		if !upgrade {
			// Build upgrade set for selective unlocking.
			upgradeSet := map[string]bool{}
			for _, p := range upgradePkgs {
				upgradeSet[pypi.Normalize(p)] = true
			}
			for _, p := range existing.Packages {
				if !upgradeSet[pypi.Normalize(p.Name)] {
					solverLocked[pypi.Normalize(p.Name)] = p.Version
				}
			}
		}
		// --upgrade: solverLocked stays empty — resolve everything fresh.
	}
	_ = existingLock // used below for --frozen diff

	var roots []resolver.Requirement
	seenRoot := map[string]bool{}
	for _, deps := range laneMap {
		for _, dep := range deps {
			dname, vSpec := splitNameSpec(dep)
			if dname == "" {
				return 1, fmt.Errorf("bunpy pm lock: invalid dep %q", dep)
			}
			spec, err := version.ParseSpec(vSpec)
			if err != nil {
				return 1, fmt.Errorf("bunpy pm lock: parse %q: %w", dep, err)
			}
			key := pypi.Normalize(dname) + "|" + dep
			if seenRoot[key] {
				continue
			}
			seenRoot[key] = true
			roots = append(roots, resolver.Requirement{Name: pypi.Normalize(dname), Spec: spec})
		}
	}
	slv := resolver.New(reg)
	slv.Locked = solverLocked
	res, err := slv.Solve(roots)
	if err != nil {
		return 1, fmt.Errorf("bunpy pm lock: %w", err)
	}

	// --frozen: fail if the resolved pins differ from the existing lock.
	if frozen {
		if existingLock == nil {
			fmt.Fprintln(stderr, "bunpy pm lock --frozen: uv.lock missing")
			return 1, fmt.Errorf("uv.lock missing")
		}
		existingPins := map[string]string{}
		for _, p := range existingLock.Packages {
			existingPins[pypi.Normalize(p.Name)] = p.Version
		}
		var drift []string
		for _, pin := range res.Pins {
			norm := pypi.Normalize(pin.Name)
			if v, ok := existingPins[norm]; !ok || v != pin.Version {
				drift = append(drift, fmt.Sprintf("%s: lock=%s resolved=%s", pin.Name, v, pin.Version))
			}
		}
		if len(drift) > 0 {
			fmt.Fprintf(stderr, "bunpy pm lock --frozen: lock is stale:\n")
			for _, d := range drift {
				fmt.Fprintf(stderr, "  %s\n", d)
			}
			return 1, fmt.Errorf("lock is stale (run without --frozen to update)")
		}
		fmt.Fprintf(stdout, "uv.lock is up-to-date (%d package%s)\n", len(existingLock.Packages), pluralS(len(existingLock.Packages)))
		return 0, nil
	}

	pinLanes, err := computePinLanes(reg, res, laneMap)
	if err != nil {
		return 1, fmt.Errorf("bunpy pm lock: %w", err)
	}

	lock := &lockfile.Lock{Version: lockfile.Version}
	for _, pin := range res.Pins {
		f, ok := reg.Pick(pin.Name, pin.Version)
		if !ok {
			return 1, fmt.Errorf("bunpy pm lock: no wheel pick for %s %s", pin.Name, pin.Version)
		}
		lp := lockfile.Package{
			Name:     pin.Name,
			Version:  pin.Version,
			Filename: f.Filename,
			URL:      f.URL,
			Hash:     wheelSha256(f),
			Size:     f.Size,
			Lanes:    pinLanes[pin.Name],
		}
		if sd, ok := reg.Sdist(pin.Name, pin.Version); ok {
			lp.SdistURL = sd.URL
			lp.SdistHash = "sha256:" + sd.Hashes["sha256"]
			lp.SdistSize = sd.Size
		}
		lock.Upsert(lp)
	}
	lock.ContentHash = wantHash
	root := &uvlock.RootInfo{
		Name:    mf.Project.Name,
		Version: mf.Project.Version,
		Deps:    mf.Project.Dependencies,
	}
	if err := uvlock.WriteLockfile("uv.lock", lock, mf.Project.RequiresPython, uvlock.WriteOptions{
		Root:          root,
		Graph:         reg.depGraph,
		DepExtras:     reg.depExtras,
		ExtraPackages: extraPkgs,
	}); err != nil {
		return 1, fmt.Errorf("bunpy pm lock: %w", err)
	}
	fmt.Fprintf(stdout, "wrote uv.lock (%d package%s)\n", len(lock.Packages), pluralS(len(lock.Packages)))
	return 0, nil
}

// computePinLanes returns pinName -> sorted lane labels by walking
// each lane's direct deps through the resolution graph and tagging
// every reachable pin.
func computePinLanes(reg *pypiRegistry, res *resolver.Resolution, laneMap map[string][]string) (map[string][]string, error) {
	pinned := map[string]string{}
	for _, p := range res.Pins {
		pinned[p.Name] = p.Version
	}
	out := map[string]map[string]bool{}
	for lane, deps := range laneMap {
		closure, err := laneClosure(reg, pinned, deps)
		if err != nil {
			return nil, err
		}
		for pkg := range closure {
			set, ok := out[pkg]
			if !ok {
				set = map[string]bool{}
				out[pkg] = set
			}
			set[lane] = true
		}
	}
	final := map[string][]string{}
	for pkg, set := range out {
		lanes := make([]string, 0, len(set))
		for l := range set {
			lanes = append(lanes, l)
		}
		sort.Strings(lanes)
		final[pkg] = lanes
	}
	return final, nil
}

// laneClosure walks every dependency reachable from rootSpecs through the
// resolution graph. It uses the depGraph already populated during Solve()
// (RC-3) to avoid any additional network fetches. Falls back to
// reg.Dependencies() only for pins not yet in the cache.
func laneClosure(reg *pypiRegistry, pinned map[string]string, rootSpecs []string) (map[string]bool, error) {
	visited := map[string]bool{}
	queue := []string{}
	for _, dep := range rootSpecs {
		name, _ := splitNameSpec(dep)
		if name == "" {
			continue
		}
		queue = append(queue, pypi.Normalize(name))
	}
	for len(queue) > 0 {
		pkg := queue[0]
		queue = queue[1:]
		if visited[pkg] {
			continue
		}
		visited[pkg] = true
		ver, ok := pinned[pkg]
		if !ok {
			continue
		}
		// RC-3: prefer the cached dep graph to avoid extra HTTP calls.
		reg.mu.Lock()
		names, cached := reg.depGraph[pkg]
		reg.mu.Unlock()
		if cached {
			queue = append(queue, names...)
			continue
		}
		deps, err := reg.Dependencies(pkg, ver)
		if err != nil {
			return nil, err
		}
		for _, d := range deps {
			queue = append(queue, d.Name)
		}
	}
	return visited, nil
}

func pluralS(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// loadWheelSource resolves source to (body, filename). A path source
// must end in .whl; an https:// URL is fetched through httpkit and
// cached under ${BUNPY_CACHE_DIR or XDG default}/wheels/<name>/.
func loadWheelSource(source string) ([]byte, string, error) {
	if strings.HasPrefix(source, "https://") || strings.HasPrefix(source, "http://") {
		return fetchWheel(source)
	}
	if !strings.HasSuffix(source, ".whl") {
		return nil, "", fmt.Errorf("source %q: must end in .whl or be an http(s) URL", source)
	}
	body, err := os.ReadFile(source)
	if err != nil {
		return nil, "", err
	}
	return body, filepath.Base(source), nil
}

func fetchWheel(rawURL string) ([]byte, string, error) {
	filename := rawURL
	if i := strings.LastIndex(rawURL, "/"); i >= 0 {
		filename = rawURL[i+1:]
	}
	if !strings.HasSuffix(filename, ".whl") {
		return nil, "", fmt.Errorf("url %q: does not end in .whl", rawURL)
	}
	pkgName := wheelProjectName(filename)
	wc, err := cache.NewWheelCache(filepath.Join(cache.DefaultDir(), "wheels"))
	if err == nil && wc.Has(pkgName, filename) {
		body, err := os.ReadFile(wc.Path(pkgName, filename))
		if err == nil {
			return body, filename, nil
		}
	}

	var rt httpkit.RoundTripper = httpkit.Default(4)
	if fix := os.Getenv("BUNPY_PYPI_FIXTURES"); fix != "" {
		rt = httpkit.FixturesFS(fix)
	}
	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return nil, "", err
	}
	resp, err := rt.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return nil, "", fmt.Errorf("get %s: %s", rawURL, resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}
	if wc != nil {
		_ = wc.Put(pkgName, filename, body)
	}
	return body, filename, nil
}

// wheelProjectName extracts the project name segment from a wheel
// filename: <name>-<version>-...whl. Returns the segment as-is so
// the cache key matches what the resolver will consume.
func wheelProjectName(filename string) string {
	base := strings.TrimSuffix(filename, ".whl")
	if i := strings.IndexByte(base, '-'); i >= 0 {
		return base[:i]
	}
	return base
}
