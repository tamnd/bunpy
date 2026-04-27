package main

import (
	"context"
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

// addSubcommand wires `bunpy add <pkg>[<spec>]`. v0.1.5 hands the
// requirement to the PubGrub-inspired resolver, which walks
// transitive Requires-Dist edges, evaluates PEP 508 markers, and
// picks platform-aware wheels via wheel.Pick. Every pin lands in
// bunpy.lock; the install layer materialises each wheel under
// .bunpy/site-packages.
func addSubcommand(args []string, stdout, stderr io.Writer) (int, error) {
	var (
		spec      string
		noInstall bool
		noWrite   bool
		target    = filepath.Join(".bunpy", "site-packages")
		baseURL   string
		cacheDir  string
	)
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "-h", "--help":
			return printHelp("add", stdout, stderr)
		case "--no-install":
			noInstall = true
		case "--no-write":
			noWrite = true
		case "--target":
			if i+1 >= len(args) {
				return 1, fmt.Errorf("bunpy add: --target requires a value")
			}
			i++
			target = args[i]
		case "--index":
			if i+1 >= len(args) {
				return 1, fmt.Errorf("bunpy add: --index requires a value")
			}
			i++
			baseURL = args[i]
		case "--cache-dir":
			if i+1 >= len(args) {
				return 1, fmt.Errorf("bunpy add: --cache-dir requires a value")
			}
			i++
			cacheDir = args[i]
		default:
			if strings.HasPrefix(a, "--target=") {
				target = strings.TrimPrefix(a, "--target=")
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
			if strings.HasPrefix(a, "-") {
				return 1, fmt.Errorf("bunpy add: unknown flag %q (known: --no-install, --no-write, --target, --index, --cache-dir, --help)", a)
			}
			if spec != "" {
				return 1, fmt.Errorf("bunpy add: too many positional arguments (%q after %q)", a, spec)
			}
			spec = a
		}
	}
	if spec == "" {
		return 1, fmt.Errorf("usage: bunpy add <pkg>[<spec>]")
	}
	name, vSpec := splitNameSpec(spec)
	if name == "" {
		return 1, fmt.Errorf("bunpy add: invalid spec %q", spec)
	}
	parsed, err := version.ParseSpec(vSpec)
	if err != nil {
		return 1, fmt.Errorf("bunpy add: parse %q: %w", spec, err)
	}

	mf, err := manifest.Load("pyproject.toml")
	if err != nil {
		return 1, fmt.Errorf("bunpy add: %w", err)
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
		func(f pypi.File) ([]byte, error) { return fetchAddWheel(f, name, cacheDir) })
	rootReqs := []resolver.Requirement{{Name: pypi.Normalize(name), Spec: parsed}}
	res, err := resolver.New(reg).Solve(rootReqs)
	if err != nil {
		return 1, fmt.Errorf("bunpy add: %w", err)
	}

	var rootPin resolver.Pin
	for _, p := range res.Pins {
		if p.Name == pypi.Normalize(name) {
			rootPin = p
			break
		}
	}
	if rootPin.Version == "" {
		return 1, fmt.Errorf("bunpy add: resolver returned no pin for %s", name)
	}
	chosen := rootPin.Version
	rootFile, ok := reg.Pick(rootPin.Name, rootPin.Version)
	if !ok {
		return 1, fmt.Errorf("bunpy add: no wheel matches host for %s %s", name, chosen)
	}

	if !noInstall {
		for _, pin := range res.Pins {
			f, ok := reg.Pick(pin.Name, pin.Version)
			if !ok {
				return 1, fmt.Errorf("bunpy add: no wheel for %s %s", pin.Name, pin.Version)
			}
			body, err := fetchAddWheel(f, pin.Name, cacheDir)
			if err != nil {
				return 1, fmt.Errorf("bunpy add: %w", err)
			}
			w, err := wheel.OpenReader(f.Filename, body)
			if err != nil {
				return 1, fmt.Errorf("bunpy add: %w", err)
			}
			verify := true
			if _, err := w.Install(target, wheel.InstallOptions{
				Installer:    "bunpy",
				VerifyHashes: &verify,
			}); err != nil {
				return 1, fmt.Errorf("bunpy add: %w", err)
			}
		}
	}

	if !noWrite {
		depLine := spec
		if vSpec == "" {
			depLine = name + ">=" + chosen
		}
		out, err := mf.AddDependency(depLine)
		if err != nil {
			return 1, fmt.Errorf("bunpy add: %w", err)
		}
		if err := os.WriteFile("pyproject.toml", out, 0o644); err != nil {
			return 1, fmt.Errorf("bunpy add: %w", err)
		}
		if err := updateLockfile("bunpy.lock", out, res, reg); err != nil {
			return 1, fmt.Errorf("bunpy add: %w", err)
		}
	}

	_ = rootFile
	fmt.Fprintf(stdout, "added %s %s\n", name, chosen)
	if extra := len(res.Pins) - 1; extra > 0 {
		fmt.Fprintf(stdout, "  + %d transitive\n", extra)
	}
	return 0, nil
}

// updateLockfile rewrites bunpy.lock so every pin in res lands in
// the file. Existing entries are upserted; the content-hash is
// recomputed from the freshly written pyproject's
// [project].dependencies.
func updateLockfile(path string, manifestBytes []byte, res *resolver.Resolution, reg *pypiRegistry) error {
	mf, err := manifest.Parse(manifestBytes)
	if err != nil {
		return fmt.Errorf("re-parse manifest: %w", err)
	}
	lock, err := lockfile.Read(path)
	if err != nil && !errors.Is(err, lockfile.ErrNotFound) {
		return err
	}
	if lock == nil {
		lock = &lockfile.Lock{Version: lockfile.Version}
	}
	for _, pin := range res.Pins {
		f, ok := reg.Pick(pin.Name, pin.Version)
		if !ok {
			return fmt.Errorf("missing wheel pick for %s %s", pin.Name, pin.Version)
		}
		lock.Upsert(lockfile.Package{
			Name:     pin.Name,
			Version:  pin.Version,
			Filename: f.Filename,
			URL:      f.URL,
			Hash:     wheelSha256(f),
		})
	}
	lock.ContentHash = lockfile.HashDependencies(mf.Project.Dependencies)
	lock.Generated = time.Now().UTC()
	return lock.WriteFile(path)
}

// wheelSha256 returns "sha256:<hex>" from f.Hashes if present.
// Empty when the index gave no sha256 entry.
func wheelSha256(f pypi.File) string {
	if h, ok := f.Hashes["sha256"]; ok && h != "" {
		return "sha256:" + h
	}
	return ""
}

// splitNameSpec separates "widget>=1.0" into ("widget", ">=1.0").
// A bare name returns ("name", "").
func splitNameSpec(s string) (string, string) {
	s = strings.TrimSpace(s)
	for i, r := range s {
		if !isAddNameRune(r) {
			return strings.TrimSpace(s[:i]), strings.TrimSpace(s[i:])
		}
	}
	return s, ""
}

func isAddNameRune(r rune) bool {
	switch {
	case r >= 'a' && r <= 'z':
		return true
	case r >= 'A' && r <= 'Z':
		return true
	case r >= '0' && r <= '9':
		return true
	case r == '-' || r == '_' || r == '.':
		return true
	}
	return false
}

func fetchAddWheel(f pypi.File, pkgName, cacheDir string) ([]byte, error) {
	root := cacheDir
	if root == "" {
		root = cache.DefaultDir()
	}
	wc, err := cache.NewWheelCache(filepath.Join(root, "wheels"))
	if err == nil && wc.Has(pkgName, f.Filename) {
		body, err := os.ReadFile(wc.Path(pkgName, f.Filename))
		if err == nil {
			return body, nil
		}
	}
	var rt httpkit.RoundTripper = httpkit.Default(4)
	if fix := os.Getenv("BUNPY_PYPI_FIXTURES"); fix != "" {
		rt = httpkit.FixturesFS(fix)
	}
	req, err := http.NewRequest("GET", f.URL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := rt.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("get %s: %s", f.URL, resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if wc != nil {
		_ = wc.Put(pkgName, f.Filename, body)
	}
	return body, nil
}
