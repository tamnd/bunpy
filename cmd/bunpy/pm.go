package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"time"

	"github.com/tamnd/bunpy/v1/internal/httpkit"
	"github.com/tamnd/bunpy/v1/pkg/cache"
	"github.com/tamnd/bunpy/v1/pkg/lockfile"
	"github.com/tamnd/bunpy/v1/pkg/manifest"
	"github.com/tamnd/bunpy/v1/pkg/marker"
	"github.com/tamnd/bunpy/v1/pkg/pypi"
	"github.com/tamnd/bunpy/v1/pkg/resolver"
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

// pmLock regenerates bunpy.lock from pyproject.toml without
// installing. With --check, exits non-zero on drift: missing
// lockfile, content-hash mismatch, or a pyproject dep with no
// lockfile entry. v0.1.5 walks the resolver so transitive deps
// land in the lockfile alongside the direct ones.
func pmLock(args []string, stdout, stderr io.Writer) (int, error) {
	var (
		check    bool
		baseURL  string
		cacheDir string
	)
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "-h", "--help":
			return printHelp("pm-lock", stdout, stderr)
		case "--check":
			check = true
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
		default:
			if strings.HasPrefix(a, "--index=") {
				baseURL = strings.TrimPrefix(a, "--index=")
				continue
			}
			if strings.HasPrefix(a, "--cache-dir=") {
				cacheDir = strings.TrimPrefix(a, "--cache-dir=")
				continue
			}
			return 1, fmt.Errorf("bunpy pm lock: unknown flag %q (known: --check, --index, --cache-dir, --help)", a)
		}
	}
	_ = cacheDir

	mf, err := manifest.Load("pyproject.toml")
	if err != nil {
		return 1, fmt.Errorf("bunpy pm lock: %w", err)
	}
	wantHash := lockfile.HashDependencies(mf.Project.Dependencies)

	if check {
		existing, err := lockfile.Read("bunpy.lock")
		if err != nil {
			if errors.Is(err, lockfile.ErrNotFound) {
				fmt.Fprintln(stderr, "bunpy pm lock --check: bunpy.lock missing")
				return 1, fmt.Errorf("bunpy.lock missing")
			}
			return 1, fmt.Errorf("bunpy pm lock: %w", err)
		}
		if existing.ContentHash != wantHash {
			fmt.Fprintf(stderr, "bunpy pm lock --check: content-hash drift (lock=%s want=%s)\n", existing.ContentHash, wantHash)
			return 1, fmt.Errorf("content-hash drift")
		}
		direct := map[string]struct{}{}
		for _, dep := range mf.Project.Dependencies {
			name, _ := splitNameSpec(dep)
			if name != "" {
				direct[lockfile.Normalize(name)] = struct{}{}
			}
		}
		for name := range direct {
			found := false
			for _, p := range existing.Packages {
				if lockfile.Normalize(p.Name) == name {
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
	}

	ctx := context.Background()
	reg := newPypiRegistry(ctx, client, wheel.HostTags(), marker.DefaultEnv(),
		func(f pypi.File) ([]byte, error) {
			body, _, err := loadWheelSource(f.URL)
			return body, err
		})
	var roots []resolver.Requirement
	for _, dep := range mf.Project.Dependencies {
		dname, vSpec := splitNameSpec(dep)
		if dname == "" {
			return 1, fmt.Errorf("bunpy pm lock: invalid dep %q", dep)
		}
		spec, err := version.ParseSpec(vSpec)
		if err != nil {
			return 1, fmt.Errorf("bunpy pm lock: parse %q: %w", dep, err)
		}
		roots = append(roots, resolver.Requirement{Name: pypi.Normalize(dname), Spec: spec})
	}
	res, err := resolver.New(reg).Solve(roots)
	if err != nil {
		return 1, fmt.Errorf("bunpy pm lock: %w", err)
	}
	lock := &lockfile.Lock{Version: lockfile.Version}
	for _, pin := range res.Pins {
		f, ok := reg.Pick(pin.Name, pin.Version)
		if !ok {
			return 1, fmt.Errorf("bunpy pm lock: no wheel pick for %s %s", pin.Name, pin.Version)
		}
		lock.Upsert(lockfile.Package{
			Name:     pin.Name,
			Version:  pin.Version,
			Filename: f.Filename,
			URL:      f.URL,
			Hash:     wheelSha256(f),
		})
	}
	lock.ContentHash = wantHash
	lock.Generated = time.Now().UTC()
	if err := lock.WriteFile("bunpy.lock"); err != nil {
		return 1, fmt.Errorf("bunpy pm lock: %w", err)
	}
	fmt.Fprintf(stdout, "wrote bunpy.lock (%d package%s)\n", len(lock.Packages), pluralS(len(lock.Packages)))
	return 0, nil
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
