//go:build ignore

// build_snapshot downloads real PyPI wheels for the snapshot benchmark
// profiles and generates a PEP 691 fixture index from them.
//
// Profiles (both pure-Python, portable across platforms):
//
//	requests-httpx — requests + httpx (10 packages)
//	httpx-rich     — httpx + rich    (11 packages)
//
// Run once and commit the output:
//
//	go run benchmarks/fixtures/build_snapshot.go
//
// Requires outbound HTTPS to pypi.org on first run.
// Output goes to benchmarks/fixtures/snapshot/.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// allPackages: every package (direct + transitive) needed across all profiles.
// All are pure-Python (py3-none-any wheels exist on PyPI).
var allPackages = []struct{ name, ver string }{
	// requests-httpx and httpx-rich shared transitive deps
	{"httpx", "0.27.0"},
	{"httpcore", "1.0.4"},
	{"h11", "0.14.0"},
	{"certifi", "2024.2.2"},
	{"anyio", "4.3.0"},
	{"sniffio", "1.3.1"},
	{"idna", "3.6"},
	// requests-httpx only
	{"requests", "2.31.0"},
	{"charset-normalizer", "3.3.2"},
	{"urllib3", "2.2.1"},
	// anyio conditional deps for Python < 3.11 (needed by uv's universal resolver)
	{"exceptiongroup", "1.2.1"},
	{"typing-extensions", "4.10.0"},
	// httpx-rich only
	{"rich", "13.7.0"},
	{"markdown-it-py", "3.0.0"},
	{"pygments", "2.17.2"},
	{"mdurl", "0.1.2"},
}

// profiles: direct dependency specs per profile (for pyproject.toml).
var profiles = map[string][]string{
	"requests-httpx": {"requests>=2.31.0", "httpx>=0.27.0"},
	"httpx-rich":     {"httpx>=0.27.0", "rich>=13.7.0"},
}

func main() {
	root, err := os.Getwd()
	if err != nil {
		die(err)
	}
	snapRoot := filepath.Join(root, "benchmarks", "fixtures", "snapshot")

	for _, pkg := range allPackages {
		whlData, filename, sha256hex, dlErr := downloadWheel(pkg.name, pkg.ver)
		if dlErr != nil {
			die(fmt.Errorf("download %s==%s: %w", pkg.name, pkg.ver, dlErr))
		}

		normName := strings.ToLower(strings.ReplaceAll(pkg.name, "_", "-"))

		mirrorDir := filepath.Join(snapRoot, "index", "files.example", normName)
		must(os.MkdirAll(mirrorDir, 0o755))
		must(os.WriteFile(filepath.Join(mirrorDir, filename), whlData, 0o644))

		writeSnapIndex(snapRoot, normName, filename, sha256hex)
		fmt.Printf("%-40s  %7d bytes  sha256=%.16s...\n",
			pkg.name+"=="+pkg.ver, len(whlData), sha256hex)
	}

	for profile, deps := range profiles {
		writeSnapProfile(snapRoot, profile, deps)
	}
	fmt.Println("done.")
}

// downloadWheel fetches the py3-none-any wheel for name==ver from the PyPI
// JSON API and verifies its SHA-256 digest.
func downloadWheel(name, ver string) (body []byte, filename, sha256hex string, err error) {
	apiURL := fmt.Sprintf("https://pypi.org/pypi/%s/%s/json", name, ver)
	resp, err := http.Get(apiURL) //nolint:noctx
	if err != nil {
		return nil, "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, "", "", fmt.Errorf("PyPI API %d for %s==%s", resp.StatusCode, name, ver)
	}

	var rel struct {
		URLs []struct {
			Filename      string            `json:"filename"`
			URL           string            `json:"url"`
			PackageType   string            `json:"packagetype"`
			PythonVersion string            `json:"python_version"`
			Digests       map[string]string `json:"digests"`
		} `json:"urls"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, "", "", err
	}

	var whlURL, whlFile, whlSHA string
	for _, u := range rel.URLs {
		if u.PackageType != "bdist_wheel" {
			continue
		}
		fn := u.Filename
		if strings.Contains(fn, "py3-none-any") || strings.Contains(fn, "py2.py3-none-any") {
			whlURL = u.URL
			whlFile = u.Filename
			whlSHA = u.Digests["sha256"]
			break
		}
	}
	if whlURL == "" {
		return nil, "", "", fmt.Errorf("no py3-none-any wheel for %s==%s", name, ver)
	}

	wresp, err := http.Get(whlURL) //nolint:noctx
	if err != nil {
		return nil, "", "", err
	}
	defer wresp.Body.Close()
	data, err := io.ReadAll(wresp.Body)
	if err != nil {
		return nil, "", "", err
	}

	sum := sha256.Sum256(data)
	got := fmt.Sprintf("%x", sum)
	if whlSHA != "" && got != whlSHA {
		return nil, "", "", fmt.Errorf("SHA256 mismatch for %s: got %s want %s", whlFile, got, whlSHA)
	}
	return data, whlFile, got, nil
}

func writeSnapIndex(snapRoot, name, filename, sha256hex string) {
	url := fmt.Sprintf("https://files.example/%s/%s", name, filename)
	dir := filepath.Join(snapRoot, "index", "pypi.org", "simple", name)
	must(os.MkdirAll(dir, 0o755))

	var b bytes.Buffer
	fmt.Fprintf(&b, "{\n  \"name\": %q,\n  \"files\": [\n", name)
	fmt.Fprintf(&b, "    {\n      \"filename\": %q,\n      \"url\": %q,\n      \"hashes\": {\"sha256\": %q}\n    }\n",
		filename, url, sha256hex)
	b.WriteString("  ],\n  \"meta\": {\"api-version\": \"1.1\", \"_last-serial\": 1}\n}\n")
	must(os.WriteFile(filepath.Join(dir, "index.json"), b.Bytes(), 0o644))
}

func writeSnapProfile(snapRoot, profile string, deps []string) {
	var sb strings.Builder
	fmt.Fprintf(&sb, "[project]\nname = %q\nversion = \"0.0.1\"\nrequires-python = \">=3.10\"\ndependencies = [\n", profile)
	for _, d := range deps {
		fmt.Fprintf(&sb, "    %q,\n", d)
	}
	sb.WriteString("]\n")
	dir := filepath.Join(snapRoot, profile)
	must(os.MkdirAll(dir, 0o755))
	must(os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte(sb.String()), 0o644))
	fmt.Printf("profile  %-20s (%d direct deps)\n", profile, len(deps))
}

func die(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

func must(err error) {
	if err != nil {
		die(err)
	}
}
