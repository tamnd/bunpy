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

	"github.com/tamnd/bunpy/v1/internal/httpkit"
	"github.com/tamnd/bunpy/v1/pkg/cache"
	"github.com/tamnd/bunpy/v1/pkg/manifest"
	"github.com/tamnd/bunpy/v1/pkg/pypi"
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
	case "-h", "--help", "help":
		return printHelp("pm", stdout, stderr)
	}
	return 1, fmt.Errorf("bunpy pm: unknown verb %q (known: config, info, install-wheel, --help)", args[0])
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
