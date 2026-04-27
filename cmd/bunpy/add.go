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

	"github.com/tamnd/bunpy/v1/internal/httpkit"
	"github.com/tamnd/bunpy/v1/pkg/cache"
	"github.com/tamnd/bunpy/v1/pkg/manifest"
	"github.com/tamnd/bunpy/v1/pkg/pypi"
	"github.com/tamnd/bunpy/v1/pkg/version"
	"github.com/tamnd/bunpy/v1/pkg/wheel"
)

// addSubcommand wires `bunpy add <pkg>[<spec>]`. v0.1.3 is the
// naive single-package porcelain: load pyproject.toml, fetch the
// PyPI project page, pick the highest universal wheel matching the
// caller's spec, install it, and write the resolved spec back. No
// transitive walk, no lockfile, no resolver yet; those land in
// v0.1.4 and v0.1.5.
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
	proj, err := client.Get(context.Background(), name)
	if err != nil {
		var nf *pypi.NotFoundError
		if errors.As(err, &nf) {
			return 1, fmt.Errorf("bunpy add: %w", err)
		}
		return 1, fmt.Errorf("bunpy add: %w", err)
	}

	chosen, file, ok := pickUniversalWheel(proj, parsed)
	if !ok {
		return 1, fmt.Errorf("bunpy add: no py3-none-any wheel matches %q", spec)
	}

	if !noInstall {
		body, err := fetchAddWheel(file, name, cacheDir)
		if err != nil {
			return 1, fmt.Errorf("bunpy add: %w", err)
		}
		w, err := wheel.OpenReader(file.Filename, body)
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
	}

	fmt.Fprintf(stdout, "added %s %s\n", name, chosen)
	return 0, nil
}

// pickUniversalWheel filters proj.Files to py3-none-any wheels and
// returns the highest version matching spec (and its file entry).
func pickUniversalWheel(proj *pypi.Project, spec version.Spec) (string, pypi.File, bool) {
	byVer := map[string]pypi.File{}
	var versions []string
	for _, f := range proj.Files {
		if f.Kind != "wheel" {
			continue
		}
		if !strings.HasSuffix(f.Filename, "-py3-none-any.whl") {
			continue
		}
		if f.Yanked {
			continue
		}
		if f.Version == "" {
			continue
		}
		if _, seen := byVer[f.Version]; seen {
			continue
		}
		byVer[f.Version] = f
		versions = append(versions, f.Version)
	}
	chosen := version.Highest(spec, versions)
	if chosen == "" {
		return "", pypi.File{}, false
	}
	return chosen, byVer[chosen], true
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
