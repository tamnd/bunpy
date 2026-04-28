//go:build ignore

// build_realworld generates real-world benchmark fixture data for the
// cross-tool comparison suite. It creates:
//
//   - A wheel file + PEP 691 index page for each of 54 named packages
//     with realistic transitive Requires-Dist metadata.
//   - Four pyproject.toml files representing common project profiles:
//     fastapi-app, django-app, datascience, cli-tool.
//
// Output goes to benchmarks/fixtures/realworld/. Run once and commit:
//
//	go run benchmarks/fixtures/build_realworld.go
package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// pkgDeps maps each package to its direct Requires-Dist entries.
// All packages are pinned at version 1.0.0 in the fixture.
var pkgDeps = map[string][]string{
	// FastAPI ecosystem
	"fastapi":          {"starlette>=1.0", "pydantic>=1.0", "anyio>=1.0", "typing-extensions>=1.0"},
	"starlette":        {"anyio>=1.0", "typing-extensions>=1.0"},
	"pydantic":         {"pydantic-core>=1.0", "annotated-types>=1.0", "typing-extensions>=1.0"},
	"pydantic-core":    {"typing-extensions>=1.0"},
	"anyio":            {"sniffio>=1.0", "typing-extensions>=1.0"},
	"sniffio":          {},
	"typing-extensions": {},
	"annotated-types":  {},
	"uvicorn":          {"click>=1.0", "h11>=1.0"},
	"click":            {},
	"h11":              {},
	"httpx":            {"httpcore>=1.0", "certifi>=1.0", "anyio>=1.0"},
	"httpcore":         {"h11>=1.0", "certifi>=1.0"},
	"certifi":          {},
	"sqlalchemy":       {"typing-extensions>=1.0"},
	"alembic":          {"sqlalchemy>=1.0", "mako>=1.0", "typing-extensions>=1.0"},
	"mako":             {"markupsafe>=1.0"},
	"markupsafe":       {},

	// Django ecosystem
	"django":               {"asgiref>=1.0", "sqlparse>=1.0"},
	"asgiref":              {"typing-extensions>=1.0"},
	"sqlparse":             {},
	"djangorestframework":  {"django>=1.0"},
	"celery":               {"kombu>=1.0", "billiard>=1.0", "vine>=1.0", "click>=1.0"},
	"kombu":                {"amqp>=1.0", "vine>=1.0"},
	"amqp":                 {"vine>=1.0"},
	"vine":                 {},
	"billiard":             {},
	"redis":                {},
	"gunicorn":             {"packaging>=1.0"},
	"packaging":            {},
	"whitenoise":           {},

	// Data science
	"numpy":          {},
	"pandas":         {"numpy>=1.0", "python-dateutil>=1.0", "pytz>=1.0", "tzdata>=1.0"},
	"python-dateutil": {"six>=1.0"},
	"six":            {},
	"pytz":           {},
	"tzdata":         {},
	"scikit-learn":   {"numpy>=1.0", "scipy>=1.0", "joblib>=1.0", "threadpoolctl>=1.0"},
	"scipy":          {"numpy>=1.0"},
	"joblib":         {},
	"threadpoolctl":  {},
	"matplotlib":     {"numpy>=1.0", "contourpy>=1.0", "cycler>=1.0", "pillow>=1.0", "pyparsing>=1.0", "python-dateutil>=1.0", "packaging>=1.0", "kiwisolver>=1.0", "fonttools>=1.0"},
	"contourpy":      {"numpy>=1.0"},
	"cycler":         {},
	"pillow":         {},
	"pyparsing":      {},
	"kiwisolver":     {},
	"fonttools":      {},

	// CLI ecosystem
	"rich":           {"markdown-it-py>=1.0", "pygments>=1.0"},
	"markdown-it-py": {"mdurl>=1.0"},
	"mdurl":          {},
	"pygments":       {},
	"typer":          {"click>=1.0", "rich>=1.0", "shellingham>=1.0", "typing-extensions>=1.0"},
	"shellingham":    {},
}

// profiles lists the direct dependencies for each real-world project profile.
var profiles = map[string][]string{
	"fastapi-app":  {"fastapi>=1.0", "uvicorn>=1.0", "httpx>=1.0", "sqlalchemy>=1.0", "alembic>=1.0"},
	"django-app":   {"django>=1.0", "djangorestframework>=1.0", "celery>=1.0", "redis>=1.0", "gunicorn>=1.0", "whitenoise>=1.0"},
	"datascience":  {"numpy>=1.0", "pandas>=1.0", "scikit-learn>=1.0", "matplotlib>=1.0"},
	"cli-tool":     {"click>=1.0", "rich>=1.0", "typer>=1.0", "httpx>=1.0", "shellingham>=1.0"},
}

func main() {
	root, err := os.Getwd()
	if err != nil {
		die(err)
	}
	rwRoot := filepath.Join(root, "benchmarks", "fixtures", "realworld")

	for name, deps := range pkgDeps {
		ver := "1.0.0"
		body := buildWheel(name, ver, deps)
		// Wheel filename uses underscores for the distribution name.
		distName := strings.ReplaceAll(name, "-", "_")
		filename := fmt.Sprintf("%s-%s-py3-none-any.whl", distName, ver)

		mirrorDir := filepath.Join(rwRoot, "index", "files.example", name)
		must(os.MkdirAll(mirrorDir, 0o755))
		must(os.WriteFile(filepath.Join(mirrorDir, filename), body, 0o644))

		sum := sha256.Sum256(body)
		hexSum := fmt.Sprintf("%x", sum)
		writeIndex(rwRoot, name, filename, hexSum)
		fmt.Printf("package  %-30s  %d bytes  sha256=%.16s...\n", name, len(body), hexSum)
	}

	for profile, deps := range profiles {
		writeProfile(rwRoot, profile, deps)
	}

	fmt.Println("done.")
}

func writeIndex(rwRoot, name, filename, sha256hex string) {
	url := fmt.Sprintf("https://files.example/%s/%s", name, filename)
	dir := filepath.Join(rwRoot, "index", "pypi.org", "simple", name)
	must(os.MkdirAll(dir, 0o755))

	var b bytes.Buffer
	fmt.Fprintf(&b, "{\n  \"name\": %q,\n  \"files\": [\n", name)
	fmt.Fprintf(&b, "    {\n      \"filename\": %q,\n      \"url\": %q,\n      \"hashes\": {\"sha256\": %q}\n    }\n", filename, url, sha256hex)
	b.WriteString("  ],\n  \"meta\": {\"api-version\": \"1.1\", \"_last-serial\": 1}\n}\n")
	must(os.WriteFile(filepath.Join(dir, "index.json"), b.Bytes(), 0o644))
}

func writeProfile(rwRoot, profile string, deps []string) {
	var sb strings.Builder
	fmt.Fprintf(&sb, "[project]\nname = %q\nversion = \"0.0.1\"\nrequires-python = \">=3.10\"\ndependencies = [\n", profile)
	for _, d := range deps {
		fmt.Fprintf(&sb, "    %q,\n", d)
	}
	sb.WriteString("]\n")

	dir := filepath.Join(rwRoot, profile)
	must(os.MkdirAll(dir, 0o755))
	must(os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte(sb.String()), 0o644))
	fmt.Printf("profile  %s (%d direct deps)\n", profile, len(deps))
}

func buildWheel(name, ver string, deps []string) []byte {
	distName := strings.ReplaceAll(name, "-", "_")
	distInfo := fmt.Sprintf("%s-%s.dist-info", distName, ver)

	initContent := fmt.Sprintf("# %s %s\nVERSION = %q\n", name, ver, ver)
	wheelContent := "Wheel-Version: 1.0\nGenerator: build_realworld\nRoot-Is-Purelib: true\nTag: py3-none-any\n"

	var metaBuf strings.Builder
	fmt.Fprintf(&metaBuf, "Metadata-Version: 2.1\nName: %s\nVersion: %s\nSummary: Real-world benchmark fixture.\n", name, ver)
	for _, dep := range deps {
		fmt.Fprintf(&metaBuf, "Requires-Dist: %s\n", dep)
	}
	metaContent := metaBuf.String()

	type entry struct {
		path    string
		content []byte
	}
	files := []entry{
		{distName + "/__init__.py", []byte(initContent)},
		{distInfo + "/WHEEL", []byte(wheelContent)},
		{distInfo + "/METADATA", []byte(metaContent)},
	}

	var rec bytes.Buffer
	w := csv.NewWriter(&rec)
	for _, f := range files {
		sum := sha256.Sum256(f.content)
		h := base64.RawURLEncoding.EncodeToString(sum[:])
		must(w.Write([]string{f.path, "sha256=" + h, strconv.Itoa(len(f.content))}))
	}
	must(w.Write([]string{distInfo + "/RECORD", "", ""}))
	w.Flush()
	must(w.Error())
	files = append(files, entry{distInfo + "/RECORD", rec.Bytes()})

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, f := range files {
		fw, err := zw.Create(f.path)
		if err != nil {
			die(err)
		}
		if _, err := fw.Write(f.content); err != nil {
			die(err)
		}
	}
	must(zw.Close())
	return buf.Bytes()
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
